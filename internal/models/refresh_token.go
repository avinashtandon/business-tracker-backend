package models

import (
	"time"

	"github.com/google/uuid"
)

// RefreshToken represents a stored refresh token record.
// The raw token string is never stored — only its SHA-256 hash.
type RefreshToken struct {
	JTI           string     `db:"jti"`
	UserID        uuid.UUID  `db:"user_id"`
	TokenHash     string     `db:"token_hash"`
	IssuedAt      time.Time  `db:"issued_at"`
	ExpiresAt     time.Time  `db:"expires_at"`
	RevokedAt     *time.Time `db:"revoked_at"`
	ReplacedByJTI *string    `db:"replaced_by_jti"`
	CreatedIP     string     `db:"created_ip"`
	UserAgent     string     `db:"user_agent"`
}

// IsRevoked returns true if this token has been revoked.
func (rt *RefreshToken) IsRevoked() bool {
	return rt.RevokedAt != nil
}

// IsExpired returns true if this token has passed its expiry time.
func (rt *RefreshToken) IsExpired() bool {
	return time.Now().After(rt.ExpiresAt)
}
