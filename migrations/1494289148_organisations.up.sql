CREATE TABLE organisations (
  id UUID PRIMARY KEY,
  revision varchar(255) NOT NULL UNIQUE,
  name varchar(255) NOT NULL,
  country varchar(255) NOT NULL DEFAULT 'Unknown'
);
