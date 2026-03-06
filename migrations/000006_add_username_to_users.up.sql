ALTER TABLE users ADD COLUMN username VARCHAR(50) NOT NULL DEFAULT '';
ALTER TABLE users ADD UNIQUE INDEX idx_users_username (username);
