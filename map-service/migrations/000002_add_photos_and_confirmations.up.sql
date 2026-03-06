ALTER TABLE map_artifacts ADD COLUMN photo_key VARCHAR(255) DEFAULT '';
ALTER TABLE map_artifacts ADD COLUMN confirmation_count INT DEFAULT 0;

CREATE TABLE artifact_confirmations (
    id SERIAL PRIMARY KEY,
    artifact_id INT NOT NULL REFERENCES map_artifacts(id) ON DELETE CASCADE,
    user_id INT NOT NULL,
    confirmed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(artifact_id, user_id)
)