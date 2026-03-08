package repository

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/avinashtandon/business-tracker-backend/internal/models"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// TokenRepository defines the data access interface for refresh tokens.
type TokenRepository interface {
	Create(ctx context.Context, token *models.RefreshToken) error
	FindByJTI(ctx context.Context, jti string) (*models.RefreshToken, error)
	Revoke(ctx context.Context, jti string, replacedByJTI *string) error
	RevokeAllForUser(ctx context.Context, userID uuid.UUID) error
	DeleteExpired(ctx context.Context) error
}

type tokenRepo struct {
	db *sqlx.DB
}

// NewTokenRepository creates a new TokenRepository backed by sqlx.
func NewTokenRepository(db *sqlx.DB) TokenRepository {
	return &tokenRepo{db: db}
}

// HashToken returns the SHA-256 hex hash of a raw token string.
// We store the hash, never the raw token, to limit exposure if the DB is compromised.
func HashToken(raw string) string {
	h := sha256.Sum256([]byte(raw))
	return fmt.Sprintf("%x", h)
}

// Create stores a new refresh token record.
func (r *tokenRepo) Create(ctx context.Context, token *models.RefreshToken) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO refresh_tokens
			(jti, user_id, token_hash, issued_at, expires_at, created_ip, user_agent)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		token.JTI,
		token.UserID[:],
		token.TokenHash,
		token.IssuedAt,
		token.ExpiresAt,
		token.CreatedIP,
		token.UserAgent,
	)
	if err != nil {
		return fmt.Errorf("creating refresh token: %w", err)
	}
	return nil
}

// FindByJTI retrieves a refresh token record by its JWT ID.
func (r *tokenRepo) FindByJTI(ctx context.Context, jti string) (*models.RefreshToken, error) {
	var row struct {
		JTI           string     `db:"jti"`
		UserID        []byte     `db:"user_id"`
		TokenHash     string     `db:"token_hash"`
		IssuedAt      time.Time  `db:"issued_at"`
		ExpiresAt     time.Time  `db:"expires_at"`
		RevokedAt     *time.Time `db:"revoked_at"`
		ReplacedByJTI *string    `db:"replaced_by_jti"`
		CreatedIP     string     `db:"created_ip"`
		UserAgent     string     `db:"user_agent"`
	}
	err := r.db.GetContext(ctx, &row,
		`SELECT jti, user_id, token_hash, issued_at, expires_at, revoked_at, replaced_by_jti, created_ip, user_agent
		 FROM refresh_tokens WHERE jti = ?`, jti)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("finding refresh token by jti: %w", err)
	}

	userID, err := uuid.FromBytes(row.UserID)
	if err != nil {
		return nil, fmt.Errorf("parsing refresh token user UUID: %w", err)
	}

	return &models.RefreshToken{
		JTI:           row.JTI,
		UserID:        userID,
		TokenHash:     row.TokenHash,
		IssuedAt:      row.IssuedAt,
		ExpiresAt:     row.ExpiresAt,
		RevokedAt:     row.RevokedAt,
		ReplacedByJTI: row.ReplacedByJTI,
		CreatedIP:     row.CreatedIP,
		UserAgent:     row.UserAgent,
	}, nil
}

// Revoke marks a refresh token as revoked and optionally sets the replacement JTI.
// This is used both for logout (replacedByJTI=nil) and rotation (replacedByJTI=newJTI).
func (r *tokenRepo) Revoke(ctx context.Context, jti string, replacedByJTI *string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE refresh_tokens SET revoked_at = ?, replaced_by_jti = ? WHERE jti = ?`,
		time.Now(), replacedByJTI, jti)
	if err != nil {
		return fmt.Errorf("revoking refresh token %s: %w", jti, err)
	}
	return nil
}

// RevokeAllForUser marks all refresh tokens for a user as revoked (used in replay attack remediation).
func (r *tokenRepo) RevokeAllForUser(ctx context.Context, userID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE refresh_tokens SET revoked_at = ? WHERE user_id = ? AND revoked_at IS NULL`,
		time.Now(), userID[:])
	if err != nil {
		return fmt.Errorf("revoking all refresh tokens for user %s: %w", userID, err)
	}
	return nil
}

// DeleteExpired removes expired or old revoked refresh tokens to keep the table clean.
// Tokens are retained for 7 days after expiry or revocation to allow replay detection.
func (r *tokenRepo) DeleteExpired(ctx context.Context) error {
	cutoff := time.Now().Add(-7 * 24 * time.Hour)
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM refresh_tokens WHERE expires_at < ? OR (revoked_at IS NOT NULL AND revoked_at < ?)`,
		cutoff, cutoff)
	if err != nil {
		return fmt.Errorf("deleting expired refresh tokens: %w", err)
	}
	return nil
}
