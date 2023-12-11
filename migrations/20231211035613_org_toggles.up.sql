ALTER TABLE organisations ADD COLUMN toggles JSONB NOT NULL DEFAULT '[]'::jsonb;
