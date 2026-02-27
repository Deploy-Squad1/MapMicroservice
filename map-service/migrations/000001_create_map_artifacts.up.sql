CREATE EXTENSION IF NOT EXISTS postgis;

CREATE TABLE IF NOT EXISTS map_artifacts (
                                             id SERIAL PRIMARY KEY,
                                             name VARCHAR(255) NOT NULL,
    category VARCHAR(50) CHECK (category IN ('yeti', 'ghost', 'alien', 'other')),
    location GEOMETRY(Point, 4326),
    created_by INT
    );