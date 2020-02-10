CREATE TABLE users (
  id UUID PRIMARY KEY,
  revision varchar(255) NOT NULL UNIQUE,
  email varchar(255) UNIQUE NOT NULL,
  password char(60) NOT NULL,
  admin BOOL NOT NULL DEFAULT false,
  verified BOOL NOT NULL DEFAULT false
);
