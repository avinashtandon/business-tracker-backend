// Package repository provides data access interfaces and implementations.
package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/avinashtandon/business-tracker-backend/internal/models"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// ErrNotFound is returned when a record is not found.
var ErrNotFound = errors.New("record not found")

// ErrDuplicateEmail is returned when a user with the same email already exists.
var ErrDuplicateEmail = errors.New("email already registered")

// UserRepository defines the data access interface for users.
type UserRepository interface {
	Create(ctx context.Context, user *models.User) error
	FindByEmail(ctx context.Context, email string) (*models.User, error)
	FindByID(ctx context.Context, id uuid.UUID) (*models.User, error)
	GetRoles(ctx context.Context, userID uuid.UUID) ([]string, error)
	ListAll(ctx context.Context) ([]*models.User, error)
	UpdatePassword(ctx context.Context, userID uuid.UUID, newPasswordHash string) error
}

// userRepo is the sqlx-backed implementation of UserRepository.
type userRepo struct {
	db *sqlx.DB
}

// NewUserRepository creates a new UserRepository backed by sqlx.
func NewUserRepository(db *sqlx.DB) UserRepository {
	return &userRepo{db: db}
}

// userRow is the internal scan target for user DB rows.
// Using time.Time directly because parseTime=true is set in the DSN.
type userRow struct {
	ID           []byte            `db:"id"`
	Email        string            `db:"email"`
	Username     string            `db:"username"`
	FirstName    string            `db:"first_name"`
	LastName     string            `db:"last_name"`
	PasswordHash string            `db:"password_hash"`
	Status       models.UserStatus `db:"status"`
	CreatedAt    time.Time         `db:"created_at"`
	UpdatedAt    time.Time         `db:"updated_at"`
}

// toModel converts a userRow to a models.User.
func (r userRow) toModel() (*models.User, error) {
	id, err := uuid.FromBytes(r.ID)
	if err != nil {
		return nil, fmt.Errorf("parsing user UUID: %w", err)
	}
	return &models.User{
		ID:           id,
		Email:        r.Email,
		Username:     r.Username,
		FirstName:    r.FirstName,
		LastName:     r.LastName,
		PasswordHash: r.PasswordHash,
		Status:       r.Status,
		CreatedAt:    r.CreatedAt,
		UpdatedAt:    r.UpdatedAt,
	}, nil
}

// Create inserts a new user record. Returns ErrDuplicateEmail on duplicate email.
func (r *userRepo) Create(ctx context.Context, user *models.User) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO users (id, email, username, first_name, last_name, password_hash, status, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		user.ID[:],
		user.Email,
		user.Username,
		user.FirstName,
		user.LastName,
		user.PasswordHash,
		user.Status,
		user.CreatedAt,
		user.UpdatedAt,
	)
	if err != nil {
		if isDuplicateKeyError(err) {
			return ErrDuplicateEmail
		}
		return fmt.Errorf("inserting user: %w", err)
	}
	return nil
}

// FindByEmail retrieves a user by email address.
func (r *userRepo) FindByEmail(ctx context.Context, email string) (*models.User, error) {
	var row userRow
	err := r.db.GetContext(ctx, &row,
		`SELECT id, email, username, first_name, last_name, password_hash, status, created_at, updated_at FROM users WHERE email = ?`, email)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("finding user by email: %w", err)
	}
	return row.toModel()
}

// FindByID retrieves a user by UUID.
func (r *userRepo) FindByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	var row userRow
	err := r.db.GetContext(ctx, &row,
		`SELECT id, email, username, first_name, last_name, password_hash, status, created_at, updated_at FROM users WHERE id = ?`, id[:])
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("finding user by id: %w", err)
	}
	return row.toModel()
}

// GetRoles returns the role names assigned to a user.
func (r *userRepo) GetRoles(ctx context.Context, userID uuid.UUID) ([]string, error) {
	var roles []string
	err := r.db.SelectContext(ctx, &roles,
		`SELECT r.name FROM roles r
		 JOIN user_roles ur ON ur.role_id = r.id
		 WHERE ur.user_id = ?`, userID[:])
	if err != nil {
		return nil, fmt.Errorf("getting user roles: %w", err)
	}
	return roles, nil
}

// ListAll returns all users (for admin use).
func (r *userRepo) ListAll(ctx context.Context) ([]*models.User, error) {
	var rows []userRow
	err := r.db.SelectContext(ctx, &rows,
		`SELECT id, email, username, first_name, last_name, password_hash, status, created_at, updated_at FROM users ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("listing users: %w", err)
	}

	users := make([]*models.User, 0, len(rows))
	for _, row := range rows {
		u, err := row.toModel()
		if err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, nil
}

// UpdatePassword sets a new password hash for the user.
func (r *userRepo) UpdatePassword(ctx context.Context, userID uuid.UUID, newPasswordHash string) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE users SET password_hash = ?, updated_at = ? WHERE id = ?`,
		newPasswordHash, time.Now(), userID[:])
	if err != nil {
		return fmt.Errorf("updating user password: %w", err)
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// isDuplicateKeyError checks if the error is a MySQL duplicate key violation.
func isDuplicateKeyError(err error) bool {
	if err == nil {
		return false
	}
	s := err.Error()
	return strings.Contains(s, "Duplicate entry") || strings.Contains(s, "1062")
}
