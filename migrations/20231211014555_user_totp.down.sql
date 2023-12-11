ALTER TABLE users DROP COLUMN totp_secret;
ALTER TABLE users DROP COLUMN totp_active;
ALTER TABLE users DROP COLUMN recovery_codes;
ALTER TABLE users DROP COLUMN totp_failure_count;
ALTER TABLE users DROP COLUMN totp_last_failure;
