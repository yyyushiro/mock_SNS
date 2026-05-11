CREATE UNIQUE INDEX users_username_unique ON users (username) WHERE username IS NOT NULL AND username <> '';
