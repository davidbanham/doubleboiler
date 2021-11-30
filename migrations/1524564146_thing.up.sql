CREATE TABLE things (
  id UUID PRIMARY KEY,
  revision TEXT NOT NULL UNIQUE,
  name TEXT NOT NULL,
  description TEXT NOT NULL,
  organisation_id UUID REFERENCES organisations (id) ON UPDATE CASCADE ON DELETE CASCADE,
  created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT (now() at time zone 'utc'),
  updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT (now() at time zone 'utc')
);

ALTER TABLE things ADD COLUMN ts tsvector
  GENERATED ALWAYS AS
    (  to_tsvector('english', coalesce(name, ''))
    || to_tsvector('english', coalesce(description, ''))
  ) STORED;

CREATE INDEX things_ts_idx ON things USING GIN (ts);
