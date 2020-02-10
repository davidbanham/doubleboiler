CREATE TABLE organisations_users (
  id UUID PRIMARY KEY,
  revision varchar(255) NOT NULL UNIQUE,
  user_id UUID REFERENCES users (id) ON UPDATE CASCADE ON DELETE CASCADE,
  organisation_id UUID REFERENCES organisations (id) ON UPDATE CASCADE ON DELETE CASCADE,
  roles JSONB NOT NULL DEFAULT '{}'
);
