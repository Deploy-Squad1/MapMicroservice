package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"map-service/internal/middlewares"
	"map-service/internal/models"
	"map-service/internal/storage"
	"net/http"
	"os"
	"strconv"

	"github.com/golang-migrate/migrate/v4"
	"github.com/joho/godotenv"
)

type ArtifactHandler struct {
	store *storage.Storage
	s3    *storage.S3Storage
}

func NewArtifactHandler(store *storage.Storage, s3 *storage.S3Storage) *ArtifactHandler {
	return &ArtifactHandler{store: store, s3: s3}
}

func (h *ArtifactHandler) GetArtifacts(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	userIDContext := r.Context().Value(middlewares.UserIDKey)
	if userIDContext == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	userID := userIDContext.(int)

	category := r.URL.Query().Get("category")
	var artifacts []models.Artifact
	var err error

	if category != "" {
		artifacts, err = h.store.FilterByCategory(category, userID)
	} else {
		artifacts, err = h.store.GetAll(userID)
	}

	if err != nil {
		http.Error(w, "Database query error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if artifacts == nil {
		artifacts = []models.Artifact{}
	}

	for i := range artifacts {
		if artifacts[i].PhotoKey != "" {
			presignedURL, err := h.s3.GeneratePresignedURL(r.Context(), artifacts[i].PhotoKey)
			if err != nil {
				log.Println("Error generating presigned URL for artifact ID", artifacts[i].ID, ":", err)
				continue
			}
			artifacts[i].PhotoURL = presignedURL
		}
	}

	json.NewEncoder(w).Encode(artifacts)
}

func (h *ArtifactHandler) CreateArtifact(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var a models.Artifact
	if err := json.NewDecoder(r.Body).Decode(&a); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	userID := r.Context().Value(middlewares.UserIDKey)
	if userID == nil {
		http.Error(w, "Unauthorized: user ID not found in context", http.StatusUnauthorized)
		return
	}
	a.CreatedBy = userID.(int)
	if err := h.store.Create(&a); err != nil {
		log.Println("Insert error:", err)
		http.Error(w, "Failed to create artifact", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(a)
}

func (h *ArtifactHandler) DeleteArtifact(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid ID format", http.StatusBadRequest)
		return
	}
	photoKey, err := h.store.GetPhotoKey(id)
	if err == nil && photoKey != "" {
		errS3 := h.s3.DeleteFromS3(r.Context(), photoKey)
		if errS3 != nil {
			log.Println("Failed to delete photo from S3:", errS3)
		} else {
			log.Println("Successfully deleted photo from S3:", photoKey)
		}
	}
	rowsAffected, err := h.store.Delete(id)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	if rowsAffected == 0 {
		http.Error(w, "Artifact not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
func (h *ArtifactHandler) UploadPhoto(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	artifactID, err := strconv.Atoi(idStr)
	if err != nil {
		log.Println("Invalid ID format", idStr)
		http.Error(w, "Invalid ID format", http.StatusBadRequest)
		return
	}

	userIDContext := r.Context().Value(middlewares.UserIDKey)
	if userIDContext == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	userID := userIDContext.(int)

	creatorID, err := h.store.GetCreatorID(artifactID)
	if err != nil {
		log.Println("Error retrieving creator ID:", err)
		http.Error(w, "Artifact not found", http.StatusNotFound)
		return
	}

	if creatorID != userID {
		log.Printf("Security alert: User %d tried to upload photo to artifact %d (owned by %d)\n", userID, artifactID, creatorID)
		http.Error(w, "Only the creator of the artifact can upload an evidence photo", http.StatusForbidden)
		return
	}

	r.ParseMultipartForm(10 << 20)

	file, fileHeader, err := r.FormFile("photo")
	if err != nil {
		log.Println("Error Retrieving the File")
		http.Error(w, "Error retrieving the file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	photoKey, err := h.s3.UploadToS3(r.Context(), file, fileHeader)
	if err != nil {
		log.Println("Error Uploading File", err)
		http.Error(w, "Failed to upload to S3", http.StatusInternalServerError)
		return
	}

	err = h.store.UpdatePhotoKey(artifactID, photoKey)
	if err != nil {
		log.Println("Error updating database with photo key:", err)
		http.Error(w, "Failed to update database", http.StatusInternalServerError)
		return
	}

	presignedURL, err := h.s3.GeneratePresignedURL(r.Context(), photoKey)
	if err != nil {
		log.Println("Error generating presigned URL:", err)
		http.Error(w, "Failed to generate presigned URL", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message":   "Photo uploaded successfully",
		"photo_url": presignedURL,
	})
}

func (h *ArtifactHandler) ConfirmArtifact(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	artifactID, err := strconv.Atoi(idStr)
	if err != nil {
		log.Println("Invalid ID format", idStr)
		http.Error(w, "Invalid ID format", http.StatusBadRequest)
		return
	}

	userIDContext := r.Context().Value(middlewares.UserIDKey)
	if userIDContext == nil {
		log.Println("User ID not found in context")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	userID := userIDContext.(int)

	creatorID, err := h.store.GetCreatorID(artifactID)
	if err != nil {
		log.Println("Error Retrieving the Creator ID")
		http.Error(w, "Artifact not found", http.StatusNotFound)
		return
	}

	if creatorID == userID {
		log.Println("User ID is the same as the creator ID")
		http.Error(w, "You cannot confirm your own artifact", http.StatusForbidden)
		return
	}

	err = h.store.AddConfirmation(artifactID, userID)
	if err != nil {
		log.Println("Error Adding Confirmation", err)
		http.Error(w, "You have already confirmed this artifact or DB error", http.StatusConflict)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Confirmed!"})
}

func (h *ArtifactHandler) RemoveConfirmation(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	artifactID, err := strconv.Atoi(idStr)
	if err != nil {
		log.Println("Invalid ID format", idStr)
		http.Error(w, "Invalid ID format", http.StatusBadRequest)
		return
	}

	userIDContext := r.Context().Value(middlewares.UserIDKey)
	if userIDContext == nil {
		log.Println("User ID not found in context")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	userID := userIDContext.(int)

	err = h.store.DeleteConfirmation(artifactID, userID)
	if err != nil {
		log.Println("Error Removing Confirmation", err)
		http.Error(w, "You have not confirmed this artifact or DB error", http.StatusConflict)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Confirmation removed!"})
}

type DatabaseMigrationHandler struct {
	store *storage.Storage
}

func NewDatabaseMigrationHandler(store *storage.Storage) *DatabaseMigrationHandler {
	return &DatabaseMigrationHandler{store: store}
}

func (h *DatabaseMigrationHandler) ApplyMigration(w http.ResponseWriter, r *http.Request) {
	if err := godotenv.Load(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	dbUser := os.Getenv("DB_USER")
	dbPassword := os.Getenv("DB_PASSWORD")
	dbHost := os.Getenv("DB_HOST")
	dbPort := os.Getenv("DB_PORT")
	dbName := os.Getenv("DB_NAME")

	connectionString := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		dbUser,
		dbPassword,
		dbHost,
		dbPort,
		dbName,
	)

	m, err := migrate.New("file://migrations", connectionString)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = m.Down()
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = m.Up()
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
