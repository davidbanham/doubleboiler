CREATE TABLE communications (
  id UUID PRIMARY KEY,
  revision varchar(255) NOT NULL UNIQUE,
  organisation_id UUID REFERENCES organisations (id),
  user_id UUID REFERENCES users (id),
  channel text NOT NULL default '',
  subject text NOT NULL default '',
  created_at TIMESTAMPTZ NOT NULL default NOW(),
  updated_at TIMESTAMPTZ NOT NULL default NOW()
);

CREATE INDEX communications_user_id ON communications (user_id);

ALTER TABLE communications ADD COLUMN ts tsvector
  GENERATED ALWAYS AS
    (  to_tsvector('english', coalesce(subject, ''))
  ) STORED;

CREATE INDEX communications_ts_idx ON communications USING GIN (ts);
