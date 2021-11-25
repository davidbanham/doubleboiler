CREATE TABLE organisations (
  id UUID PRIMARY KEY,
  revision TEXT NOT NULL UNIQUE,
  name TEXT NOT NULL,
  country TEXT NOT NULL DEFAULT 'Unknown'
);
