-- Migration: 000003 create user_roles join table
-- Tradeoff: many-to-many vs single role column.
-- We chose many-to-many because it allows users to hold multiple roles
-- (e.g. a user who is also an admin), is more extensible, and avoids
-- denormalization. The tradeoff is a JOIN on every auth check, which is
-- acceptable given the small size of the roles table.

CREATE TABLE IF NOT EXISTS user_roles (
    user_id BINARY(16)   NOT NULL,
    role_id INT UNSIGNED NOT NULL,

    PRIMARY KEY (user_id, role_id),
    CONSTRAINT fk_user_roles_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT fk_user_roles_role FOREIGN KEY (role_id) REFERENCES roles(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
