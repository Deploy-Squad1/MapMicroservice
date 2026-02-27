package handlers

import (
	"encoding/json"
	"log"
	"map-service/internal/middlewares"
	"map-service/internal/models"
	"map-service/internal/storage"
	"net/http"
	"strconv"
)

type ArtifactHandler struct {
	store *storage.Storage
}

func NewArtifactHandler(store *storage.Storage) *ArtifactHandler {
	return &ArtifactHandler{store: store}
}

func (h *ArtifactHandler) GetArtifacts(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	category := r.URL.Query().Get("category")
	var artifacts []models.Artifact
	var err error
	if category != "" {
		artifacts, err = h.store.FilterByCategory(category)
	} else {
		artifacts, err = h.store.GetAll()
	}
	if err != nil {
		http.Error(w, "Database query error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if artifacts == nil {
		artifacts = []models.Artifact{}
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
