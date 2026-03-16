package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"

	"map-service/internal/handlers"
	"map-service/internal/middlewares"
	"map-service/internal/storage"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found.")
	}
	dbUser := os.Getenv("DB_USER")
	dbPassword := os.Getenv("DB_PASSWORD")
	dbHost := os.Getenv("DB_HOST")
	dbPort := os.Getenv("DB_PORT")
	dbName := os.Getenv("DB_NAME")
	dbConnectionString := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", dbUser, dbPassword, dbHost, dbPort, dbName)

	runMigrations(dbConnectionString)

	db, err := sql.Open("postgres", dbConnectionString)
	if err != nil {
		log.Fatal("Error connecting to the database: ", err)
	}
	defer db.Close()

	if err = db.Ping(); err != nil {
		log.Fatal("Failed to connect to DB:", err)
	}
	fmt.Println("Connected to PostgreSQL/PostGIS!")

	store := storage.New(db)
	s3Storage, err := storage.NewS3Storage(context.Background())
	if err != nil {
		log.Fatal("Failed to initialize S3 storage:", err)
	}
	artifactHandler := handlers.NewArtifactHandler(store, s3Storage)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /map/api/artifacts", middlewares.RequireRole()(artifactHandler.GetArtifacts))
	mux.HandleFunc("POST /map/api/artifacts", middlewares.RequireRole("Silver", "Gold")(artifactHandler.CreateArtifact))
	mux.HandleFunc("DELETE /map/api/artifacts/{id}", middlewares.RequireRole("Gold")(artifactHandler.DeleteArtifact))
	mux.HandleFunc("POST /map/api/artifacts/{id}/photo", middlewares.RequireRole("Silver", "Gold")(artifactHandler.UploadPhoto))
	mux.HandleFunc("POST /map/api/artifacts/{id}/confirm", middlewares.RequireRole()(artifactHandler.ConfirmArtifact))
	mux.HandleFunc("DELETE /map/api/artifacts/{id}/unconfirm", middlewares.RequireRole()(artifactHandler.RemoveConfirmation))
	// Internal endpoints
	databaseMigrationHandler := handlers.NewDatabaseMigrationHandler(store)
	mux.HandleFunc("DELETE /map/api/internal/database/delete", databaseMigrationHandler.ApplyMigration)

	handlerWithCORS := middlewares.CORS(mux)

	fmt.Println("Server is running on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", handlerWithCORS))
}

func runMigrations(connectionString string) {
	m, err := migrate.New("file://migrations", connectionString)
	if err != nil {
		log.Fatal("Cannot create migrate instance:", err)
	}

	err = m.Up()
	if err != nil {
		if err == migrate.ErrNoChange {
			fmt.Println("No new migrations to apply.")
			return
		}
		log.Fatal("Failed to run migrate up:", err)
	}

	fmt.Println("Migrations applied successfully!")
}
