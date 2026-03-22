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
	ID           uuid.UUID
	Email        string
	Username     string
	FirstName    string
	LastName     string
	PasswordHash string
	Status       UserStatus
	CreatedAt    time.Time
	UpdatedAt    time.Time
	Roles        []string
}
