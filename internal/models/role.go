package models

import "time"

// Role represents a named permission group.
type Role struct {
	ID        uint      `db:"id"`
	Name      string    `db:"name"`
	CreatedAt time.Time `db:"created_at"`
}

// Well-known role names.
const (
	RoleUser  = "user"
	RoleAdmin = "admin"
)
