package models

import (
	"time"

	"github.com/google/uuid"
)

// PasswordResetToken represents a temporary token used to securely reset a user's password.
type PasswordResetToken struct {
	ID        uuid.UUID  `db:"id"`
	UserID    uuid.UUID  `db:"user_id"`
	TokenHash string     `db:"token_hash"`
	ExpiresAt time.Time  `db:"expires_at"`
	UsedAt    *time.Time `db:"used_at"`
	IPAddress *string    `db:"ip_address"`
	CreatedAt time.Time  `db:"created_at"`
}

// IsExpired returns true if the token has passed its expiration time.
func (t *PasswordResetToken) IsExpired() bool {
	return time.Now().After(t.ExpiresAt)
}

// IsUsed returns true if the token has already been consumed.
func (t *PasswordResetToken) IsUsed() bool {
	return t.UsedAt != nil
}
