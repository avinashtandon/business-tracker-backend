package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/avinashtandon/business-tracker-backend/internal/models"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// PasswordResetRepository defines the interface for creating and validating reset tokens.
type PasswordResetRepository interface {
	Create(ctx context.Context, token *models.PasswordResetToken) error
	FindByTokenHash(ctx context.Context, hash string) (*models.PasswordResetToken, error)
	MarkAsUsed(ctx context.Context, id uuid.UUID) error
	DeleteByUser(ctx context.Context, userID uuid.UUID) error
}

type passwordResetRepo struct {
	db *sqlx.DB
}

// NewPasswordResetRepository creates a new repo backed by sqlx.
func NewPasswordResetRepository(db *sqlx.DB) PasswordResetRepository {
	return &passwordResetRepo{db: db}
}

// Create inserts a new PasswordResetToken into the table.
func (r *passwordResetRepo) Create(ctx context.Context, token *models.PasswordResetToken) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO password_reset_tokens (id, user_id, token_hash, expires_at, used_at, ip_address, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		token.ID[:],
		token.UserID[:],
		token.TokenHash,
		token.ExpiresAt,
		token.UsedAt,
		token.IPAddress,
		token.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("inserting password reset token: %w", err)
	}
	return nil
}

// FindByTokenHash retrieves a token model via its securely hashed DB value.
func (r *passwordResetRepo) FindByTokenHash(ctx context.Context, hash string) (*models.PasswordResetToken, error) {
	var row struct {
		ID        []byte     `db:"id"`
		UserID    []byte     `db:"user_id"`
		TokenHash string     `db:"token_hash"`
		ExpiresAt time.Time  `db:"expires_at"`
		UsedAt    *time.Time `db:"used_at"`
		IPAddress *string    `db:"ip_address"`
		CreatedAt time.Time  `db:"created_at"`
	}

	err := r.db.GetContext(ctx, &row,
		`SELECT id, user_id, token_hash, expires_at, used_at, ip_address, created_at 
		 FROM password_reset_tokens WHERE token_hash = ?`, hash)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("finding reset token by hash: %w", err)
	}

	id, err := uuid.FromBytes(row.ID)
	if err != nil {
		return nil, fmt.Errorf("parsing token id: %w", err)
	}
	userID, err := uuid.FromBytes(row.UserID)
	if err != nil {
		return nil, fmt.Errorf("parsing user id: %w", err)
	}

	return &models.PasswordResetToken{
		ID:        id,
		UserID:    userID,
		TokenHash: row.TokenHash,
		ExpiresAt: row.ExpiresAt,
		UsedAt:    row.UsedAt,
		IPAddress: row.IPAddress,
		CreatedAt: row.CreatedAt,
	}, nil
}

// MarkAsUsed effectively locks out the token from being used twice.
func (r *passwordResetRepo) MarkAsUsed(ctx context.Context, id uuid.UUID) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE password_reset_tokens SET used_at = ? WHERE id = ? AND used_at IS NULL`,
		time.Now(), id[:])

	if err != nil {
		return fmt.Errorf("marking token as used: %w", err)
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// DeleteByUser deletes all reset tokens for a user, enforcing the "One Active Reset Code" rule.
func (r *passwordResetRepo) DeleteByUser(ctx context.Context, userID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM password_reset_tokens WHERE user_id = ?`, userID[:])
	if err != nil {
		return fmt.Errorf("deleting old reset tokens: %w", err)
	}
	return nil
}
