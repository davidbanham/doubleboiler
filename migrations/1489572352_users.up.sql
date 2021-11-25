CREATE TABLE users (
  id UUID PRIMARY KEY,
  revision TEXT NOT NULL UNIQUE,
  email TEXT UNIQUE NOT NULL,
  password char(60) NOT NULL,
  admin BOOL NOT NULL DEFAULT false,
  verified BOOL NOT NULL DEFAULT false
);
