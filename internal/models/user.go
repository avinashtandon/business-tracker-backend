// Package models defines the domain models for the application.
package models

import (
	"time"

	"github.com/google/uuid"
)

// UserStatus represents the account state of a user.
type UserStatus string

const (
	UserStatusActive   UserStatus = "active"
	UserStatusInactive UserStatus = "inactive"
	UserStatusBanned   UserStatus = "banned"
)

// User represents a registered account in the system.
type User struct {
	ID           uuid.UUID  `db:"id"`
	Email        string     `db:"email"`
	PasswordHash string     `db:"password_hash"`
	Status       UserStatus `db:"status"`
	CreatedAt    time.Time  `db:"created_at"`
	UpdatedAt    time.Time  `db:"updated_at"`
	// Roles is populated by a JOIN query, not stored in users table.
	Roles []string `db:"-"`
}

// PublicUser is the safe representation returned to clients (no password hash).
type PublicUser struct {
	ID        string     `json:"id"`
	Email     string     `json:"email"`
	Status    UserStatus `json:"status"`
	Roles     []string   `json:"roles"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

// ToPublic converts a User to its safe public representation.
func (u *User) ToPublic() *PublicUser {
	roles := u.Roles
	if roles == nil {
		roles = []string{}
	}
	return &PublicUser{
		ID:        u.ID.String(),
		Email:     u.Email,
		Status:    u.Status,
		Roles:     roles,
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
	}
}
