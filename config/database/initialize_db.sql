CREATE USER doubleboiler;
ALTER USER doubleboiler WITH SUPERUSER;
CREATE DATABASE doubleboiler;
CREATE DATABASE doubleboiler_test;
GRANT ALL PRIVILEGES ON DATABASE doubleboiler TO doubleboiler;
GRANT ALL PRIVILEGES ON DATABASE doubleboiler_test TO doubleboiler;
CREATE EXTENSION pgcrypto;
