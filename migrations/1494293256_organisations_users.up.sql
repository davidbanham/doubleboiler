CREATE TABLE organisations_users (
  id UUID PRIMARY KEY,
  revision TEXT NOT NULL UNIQUE,
  user_id UUID REFERENCES users (id) ON UPDATE CASCADE ON DELETE CASCADE,
  organisation_id UUID REFERENCES organisations (id) ON UPDATE CASCADE ON DELETE CASCADE,
  roles JSONB NOT NULL DEFAULT '{}',
  created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT (now() at time zone 'utc'),
  updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT (now() at time zone 'utc')
);
