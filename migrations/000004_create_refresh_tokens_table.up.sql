-- Migration: 000004 create refresh_tokens table
-- Stores hashed refresh tokens for server-side revocation.
-- jti (JWT ID) is the primary key — a UUID generated per token.
-- token_hash stores SHA-256(raw_refresh_token) so the raw token is never stored.
-- replaced_by_jti enables token family tracking for rotation replay detection.

CREATE TABLE IF NOT EXISTS refresh_tokens (
    jti             VARCHAR(36)  NOT NULL,
    user_id         BINARY(16)   NOT NULL,
    token_hash      VARCHAR(64)  NOT NULL COMMENT 'SHA-256 hex of the raw refresh token string',
    issued_at       DATETIME(3)  NOT NULL,
    expires_at      DATETIME(3)  NOT NULL,
    revoked_at      DATETIME(3)  NULL,
    replaced_by_jti VARCHAR(36)  NULL COMMENT 'JTI of the token that replaced this one on rotation',
    created_ip      VARCHAR(45)  NOT NULL COMMENT 'IPv4 or IPv6 of the issuing request',
    user_agent      VARCHAR(512) NOT NULL,

    PRIMARY KEY (jti),
    KEY idx_rt_user_id (user_id),
    KEY idx_rt_expires_at (expires_at),
    CONSTRAINT fk_rt_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
