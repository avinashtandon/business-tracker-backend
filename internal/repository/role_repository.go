package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/avinashtandon/business-tracker-backend/internal/models"
)

// RoleRepository defines the data access interface for roles.
type RoleRepository interface {
	FindByName(ctx context.Context, name string) (*models.Role, error)
	AssignToUser(ctx context.Context, userID uuid.UUID, roleID uint) error
}

type roleRepo struct {
	db *sqlx.DB
}

// NewRoleRepository creates a new RoleRepository backed by sqlx.
func NewRoleRepository(db *sqlx.DB) RoleRepository {
	return &roleRepo{db: db}
}

// FindByName retrieves a role by its name.
func (r *roleRepo) FindByName(ctx context.Context, name string) (*models.Role, error) {
	var role models.Role
	err := r.db.GetContext(ctx, &role, `SELECT id, name, created_at FROM roles WHERE name = ?`, name)
	if err != nil {
		return nil, fmt.Errorf("finding role by name %q: %w", name, err)
	}
	return &role, nil
}

// AssignToUser assigns a role to a user (INSERT IGNORE to be idempotent).
func (r *roleRepo) AssignToUser(ctx context.Context, userID uuid.UUID, roleID uint) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT IGNORE INTO user_roles (user_id, role_id) VALUES (?, ?)`,
		userID[:], roleID)
	if err != nil {
		return fmt.Errorf("assigning role %d to user %s: %w", roleID, userID, err)
	}
	return nil
}
