DROP INDEX IF EXISTS users_email_unique;

ALTER TABLE users
    DROP COLUMN email,
    DROP COLUMN hashed_password,
    DROP COLUMN email_verified,
    ALTER COLUMN google_sub SET NOT NULL;
