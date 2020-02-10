CREATE TABLE things (
  id UUID PRIMARY KEY,
  revision varchar(255) NOT NULL UNIQUE,
  name varchar(255) NOT NULL,
  organisation_id UUID REFERENCES organisations (id) ON UPDATE CASCADE ON DELETE CASCADE
);
