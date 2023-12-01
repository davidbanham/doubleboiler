CREATE TABLE some_things (
  id UUID PRIMARY KEY,
  revision TEXT NOT NULL UNIQUE,
  name TEXT NOT NULL,
  description TEXT NOT NULL,
  organisation_id UUID REFERENCES organisations (id) ON UPDATE CASCADE ON DELETE CASCADE NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  soft_deleted BOOLEAN NOT NULL DEFAULT false
);

ALTER TABLE some_things ADD COLUMN ts tsvector
  GENERATED ALWAYS AS
    (  to_tsvector('english', coalesce(name, ''))
    || to_tsvector('english', coalesce(description, ''))
  ) STORED;

CREATE INDEX some_things_ts_idx ON some_things USING GIN (ts);
