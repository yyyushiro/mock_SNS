ALTER TABLE users
    ALTER COLUMN google_sub DROP NOT NULL,
    ADD COLUMN email TEXT,
    ADD COLUMN hashed_password VARCHAR(255),
    ADD COLUMN email_verified BOOLEAN NOT NULL DEFAULT FALSE;

CREATE UNIQUE INDEX users_email_unique ON users (LOWER(email))
    WHERE email IS NOT NULL AND email <> '';