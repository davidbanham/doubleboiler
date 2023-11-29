ALTER TABLE users ADD COLUMN flashes jsonb NOT NULL DEFAULT '[]';
ALTER TABLE users ADD COLUMN has_flashes boolean GENERATED ALWAYS AS (jsonb_array_length(flashes) > 0) STORED;
