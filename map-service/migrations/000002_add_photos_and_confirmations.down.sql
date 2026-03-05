DROP TABLE IF EXISTS artifact_confirmations;
ALTER TABLE map_artifacts DROP COLUMN IF EXISTS photo_key;
ALTER TABLE map_artifacts DROP COLUMN IF EXISTS confirmation_count;

