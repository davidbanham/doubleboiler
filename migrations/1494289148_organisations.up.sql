CREATE TABLE organisations (
  id UUID PRIMARY KEY,
  revision TEXT NOT NULL UNIQUE,
  name TEXT NOT NULL,
  country TEXT NOT NULL DEFAULT 'Unknown',
  created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT (now() at time zone 'utc'),
  updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT (now() at time zone 'utc')
);

ALTER TABLE organisations ADD COLUMN ts tsvector
  GENERATED ALWAYS AS
    (  to_tsvector('english', coalesce(name, ''))
    || to_tsvector('english', coalesce(country, ''))
  ) STORED;

CREATE INDEX organisations_ts_idx ON organisations USING GIN (ts);
