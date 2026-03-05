package storage

import (
	"database/sql"
	"map-service/internal/models"
)

type Storage struct {
	db *sql.DB
}

func New(db *sql.DB) *Storage {
	return &Storage{db: db}
}

func (s *Storage) GetAll(userID int) ([]models.Artifact, error) {
	query := `
		SELECT 
			a.id, a.name, ST_Y(a.location) as lat, ST_X(a.location) as lng, 
			a.category, a.created_by, COALESCE(a.photo_key, '') as photo_key, 
			a.confirmation_count,
			EXISTS(SELECT 1 FROM artifact_confirmations ac WHERE ac.artifact_id = a.id AND ac.user_id = $1) as has_confirmed
		FROM map_artifacts a
	`

	rows, err := s.db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var artifacts []models.Artifact
	for rows.Next() {
		var a models.Artifact
		if err := rows.Scan(&a.ID, &a.Name, &a.Lat, &a.Lng, &a.Category, &a.CreatedBy, &a.PhotoKey, &a.Confirmations, &a.HasConfirmed); err != nil {
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

func (s *Storage) FilterByCategory(category string, userID int) ([]models.Artifact, error) {
	query := `
		SELECT 
			a.id, a.name, ST_Y(a.location) as lat, ST_X(a.location) as lng, 
			a.category, a.created_by, COALESCE(a.photo_key, '') as photo_key, 
			a.confirmation_count,
			EXISTS(SELECT 1 FROM artifact_confirmations ac WHERE ac.artifact_id = a.id AND ac.user_id = $2) as has_confirmed
		FROM map_artifacts a
		WHERE a.category = $1
	`

	rows, err := s.db.Query(query, category, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var artifacts []models.Artifact
	for rows.Next() {
		var a models.Artifact
		if err := rows.Scan(&a.ID, &a.Name, &a.Lat, &a.Lng, &a.Category, &a.CreatedBy, &a.PhotoKey, &a.Confirmations, &a.HasConfirmed); err != nil {
			continue
		}
		artifacts = append(artifacts, a)
	}
	if artifacts == nil {
		artifacts = []models.Artifact{}
	}
	return artifacts, nil
}

func (s *Storage) UpdatePhotoKey(artifactID int, photoKey string) error {
	_, err := s.db.Exec("UPDATE map_artifacts SET photo_key = $1 WHERE id = $2", photoKey, artifactID)
	return err
}

func (s *Storage) GetCreatorID(artifactID int) (int, error) {
	var creatorID int
	err := s.db.QueryRow("SELECT created_by FROM map_artifacts WHERE id = $1", artifactID).Scan(&creatorID)
	return creatorID, err
}

func (s *Storage) AddConfirmation(artifactID int, userID int) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec("INSERT INTO artifact_confirmations (artifact_id, user_id) VALUES ($1, $2)", artifactID, userID)
	if err != nil {
		return err
	}

	_, err = tx.Exec("UPDATE map_artifacts SET confirmation_count = confirmation_count + 1 WHERE id = $1", artifactID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (s *Storage) DeleteConfirmation(artifactID int, userID int) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec("DELETE FROM artifact_confirmations WHERE artifact_id = $1 AND user_id = $2", artifactID, userID)
	if err != nil {
		return err
	}

	_, err = tx.Exec("UPDATE map_artifacts SET confirmation_count = confirmation_count - 1 WHERE id = $1", artifactID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (s *Storage) GetPhotoKey(artifactID int) (string, error) {
	var photoKey *string
	err := s.db.QueryRow("SELECT photo_key FROM map_artifacts WHERE id = $1", artifactID).Scan(&photoKey)
	if err != nil {
		return "", err
	}
	if photoKey == nil {
		return "", nil
	}
	return *photoKey, nil
}
