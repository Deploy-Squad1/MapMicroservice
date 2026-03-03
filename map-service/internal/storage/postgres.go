package storage

import (
	"database/sql"
	"log"

	"map-service/internal/models"
)

type Storage struct {
	db *sql.DB
}

func New(db *sql.DB) *Storage {
	return &Storage{db: db}
}

func (s *Storage) GetAll() ([]models.Artifact, error) {
	rows, err := s.db.Query(`SELECT id, name, ST_Y(location) as lat, ST_X(location) as lng, category, created_by FROM map_artifacts`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var artifacts []models.Artifact
	for rows.Next() {
		var a models.Artifact
		if err := rows.Scan(&a.ID, &a.Name, &a.Lat, &a.Lng, &a.Category, &a.CreatedBy); err != nil {
			log.Println("Row scan error:", err)
			continue
		}
		artifacts = append(artifacts, a)
	}

	if artifacts == nil {
		artifacts = []models.Artifact{}
	}
	return artifacts, nil
}

func (s *Storage) Create(a *models.Artifact) error {
	validCategories := map[string]bool{"yeti": true, "ghost": true, "alien": true, "other": true}
	if !validCategories[a.Category] {
		a.Category = "other"
	}

	err := s.db.QueryRow(`
       INSERT INTO map_artifacts (name, location, category, created_by) 
       VALUES ($1, ST_SetSRID(ST_MakePoint($2, $3), 4326), $4, $5) 
       RETURNING id`,
		a.Name, a.Lng, a.Lat, a.Category, a.CreatedBy,
	).Scan(&a.ID)

	return err
}

func (s *Storage) Delete(id int) (int64, error) {
	res, err := s.db.Exec("DELETE FROM map_artifacts WHERE id = $1", id)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (s *Storage) FilterByCategory(category string) ([]models.Artifact, error) {
	rows, err := s.db.Query(`SELECT id, name, ST_Y(location) as lat, ST_X(location) as lng, category, created_by FROM map_artifacts WHERE category = $1`, category)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var artifacts []models.Artifact
	for rows.Next() {
		var a models.Artifact
		if err := rows.Scan(&a.ID, &a.Name, &a.Lat, &a.Lng, &a.Category, &a.CreatedBy); err != nil {
			log.Println("Row scan error:", err)
			continue
		}
		artifacts = append(artifacts, a)
	}

	if artifacts == nil {
		artifacts = []models.Artifact{}
	}
	return artifacts, nil
}
