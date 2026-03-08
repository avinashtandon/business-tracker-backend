// Package service implements the business logic for authentication.
package service

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/avinashtandon/business-tracker-backend/internal/models"
	"github.com/avinashtandon/business-tracker-backend/internal/repository"
	"github.com/avinashtandon/business-tracker-backend/pkg/jwtpkg"
	"github.com/avinashtandon/business-tracker-backend/pkg/password"
	"github.com/google/uuid"
)

// Sentinel errors for the auth service.
var (
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrUserInactive       = errors.New("account is not active")
	ErrTokenInvalid       = errors.New("refresh token is invalid or expired")
	ErrTokenRevoked       = errors.New("refresh token has been revoked")
	ErrEmailTaken         = errors.New("email address is already registered")
)

// AuthTokens is the response payload for login and refresh operations.
type AuthTokens struct {
	AccessToken  string             `json:"access_token"`
	RefreshToken string             `json:"refresh_token"`
	TokenType    string             `json:"token_type"`
	ExpiresIn    int                `json:"expires_in"` // seconds
	ExpiresAt    time.Time          `json:"expires_at"`
	User         *models.PublicUser `json:"user,omitempty"`
}

// RegisterInput is the validated input for user registration.
type RegisterInput struct {
	Email     string `json:"email"      validate:"required,email,max=255"`
	Username  string `json:"username"   validate:"required,min=3,max=50"`
	FirstName string `json:"first_name" validate:"required,max=100"`
	LastName  string `json:"last_name"  validate:"required,max=100"`
	Password  string `json:"password"   validate:"required,min=8,max=128"`
}

// LoginInput is the validated input for user login.
type LoginInput struct {
	Email    string `json:"email"    validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

// AuthService defines the authentication business logic interface.
type AuthService interface {
	Register(ctx context.Context, input RegisterInput) (*models.PublicUser, error)
	Login(ctx context.Context, input LoginInput, ip, userAgent string) (*AuthTokens, error)
	RefreshTokens(ctx context.Context, rawRefreshToken, ip, userAgent string) (*AuthTokens, error)
	Logout(ctx context.Context, rawRefreshToken string) error
	GetMe(ctx context.Context, userID string) (*models.PublicUser, error)
}

type authService struct {
	userRepo  repository.UserRepository
	roleRepo  repository.RoleRepository
	tokenRepo repository.TokenRepository
	jwtMgr    *jwtpkg.Manager
	accessTTL time.Duration
}

// NewAuthService creates a new AuthService.
func NewAuthService(
	userRepo repository.UserRepository,
	roleRepo repository.RoleRepository,
	tokenRepo repository.TokenRepository,
	jwtMgr *jwtpkg.Manager,
	accessTTL time.Duration,
) AuthService {
	return &authService{
		userRepo:  userRepo,
		roleRepo:  roleRepo,
		tokenRepo: tokenRepo,
		jwtMgr:    jwtMgr,
		accessTTL: accessTTL,
	}
}

// Register creates a new user account with the "user" role.
func (s *authService) Register(ctx context.Context, input RegisterInput) (*models.PublicUser, error) {
	hash, err := password.Hash(input.Password)
	if err != nil {
		return nil, fmt.Errorf("hashing password: %w", err)
	}

	now := time.Now()
	user := &models.User{
		ID:           uuid.New(),
		Email:        input.Email,
		Username:     input.Username,
		FirstName:    input.FirstName,
		LastName:     input.LastName,
		PasswordHash: hash,
		Status:       models.UserStatusActive,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		if errors.Is(err, repository.ErrDuplicateEmail) {
			return nil, ErrEmailTaken
		}
		return nil, fmt.Errorf("creating user: %w", err)
	}

	// Assign the default "user" role.
	role, err := s.roleRepo.FindByName(ctx, models.RoleUser)
	if err != nil {
		return nil, fmt.Errorf("finding user role: %w", err)
	}
	if err := s.roleRepo.AssignToUser(ctx, user.ID, role.ID); err != nil {
		return nil, fmt.Errorf("assigning role: %w", err)
	}

	user.Roles = []string{models.RoleUser}
	return user.ToPublic(), nil
}

// Login verifies credentials and issues access + refresh tokens.
func (s *authService) Login(ctx context.Context, input LoginInput, ip, userAgent string) (*AuthTokens, error) {
	user, err := s.userRepo.FindByEmail(ctx, input.Email)
	if errors.Is(err, repository.ErrNotFound) {
		// Use constant-time comparison to avoid timing attacks.
		_ = password.Verify("$2a$12$invalidhashfortimingatk", input.Password)
		return nil, ErrInvalidCredentials
	}
	if err != nil {
		return nil, fmt.Errorf("finding user: %w", err)
	}

	if err := password.Verify(user.PasswordHash, input.Password); err != nil {
		return nil, ErrInvalidCredentials
	}

	if user.Status != models.UserStatusActive {
		return nil, ErrUserInactive
	}

	roles, err := s.userRepo.GetRoles(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("getting roles: %w", err)
	}

	user.Roles = roles

	return s.issueTokenPair(ctx, user.ID.String(), roles, ip, userAgent, user.ToPublic())
}

// RefreshTokens validates the refresh token, rotates it, and issues a new pair.
// Rotation strategy: the old token is revoked and a new one is issued.
// If a revoked token is presented (replay attack), we detect it via replaced_by_jti.
func (s *authService) RefreshTokens(ctx context.Context, rawRefreshToken, ip, userAgent string) (*AuthTokens, error) {
	// 1. Validate JWT signature, expiry, iss, aud, typ=refresh.
	claims, err := s.jwtMgr.ValidateRefreshToken(rawRefreshToken)
	if err != nil {
		return nil, ErrTokenInvalid
	}

	// 2. Look up the token record in the DB by jti.
	record, err := s.tokenRepo.FindByJTI(ctx, claims.ID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrTokenInvalid
	}
	if err != nil {
		return nil, fmt.Errorf("finding refresh token: %w", err)
	}

	// 3. Check revocation — if already revoked, this may be a replay attack.
	if record.IsRevoked() {
		// Replay attack detected: revoke all user sessions to secure the account.
		if revokeErr := s.tokenRepo.RevokeAllForUser(ctx, record.UserID); revokeErr != nil {
			return nil, fmt.Errorf("revoking all tokens on replay attack: %w", revokeErr)
		}
		return nil, ErrTokenRevoked
	}

	// 4. Double-check expiry (DB-level safety net).
	if record.IsExpired() {
		return nil, ErrTokenInvalid
	}

	// 5. Verify the raw token hash matches what's stored (prevents token forgery).
	expectedHash := repository.HashToken(rawRefreshToken)
	if expectedHash != record.TokenHash {
		return nil, ErrTokenInvalid
	}

	// 6. Get current user and roles.
	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		return nil, ErrTokenInvalid
	}
	roles, err := s.userRepo.GetRoles(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("getting roles: %w", err)
	}

	// 7. Get user profile and Issue new token pair.
	userProfile, err := s.GetMe(ctx, claims.Subject)
	if err != nil {
		return nil, fmt.Errorf("getting user profile: %w", err)
	}

	tokens, err := s.issueTokenPair(ctx, claims.Subject, roles, ip, userAgent, userProfile)
	if err != nil {
		return nil, err
	}

	// 8. Revoke the old token, linking it to the new one for audit trail.
	// We need the new JTI — parse it from the new refresh token.
	newClaims, err := s.jwtMgr.ValidateRefreshToken(tokens.RefreshToken)
	if err != nil {
		return nil, fmt.Errorf("validating new refresh token: %w", err)
	}
	newJTI := newClaims.ID
	if err := s.tokenRepo.Revoke(ctx, record.JTI, &newJTI); err != nil {
		return nil, fmt.Errorf("revoking old refresh token: %w", err)
	}

	return tokens, nil
}

// Logout revokes the refresh token server-side.
func (s *authService) Logout(ctx context.Context, rawRefreshToken string) error {
	claims, err := s.jwtMgr.ValidateRefreshToken(rawRefreshToken)
	if err != nil {
		// Even if the token is expired/invalid, we treat logout as success.
		return nil
	}

	record, err := s.tokenRepo.FindByJTI(ctx, claims.ID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil // Already gone — idempotent.
	}
	if err != nil {
		return fmt.Errorf("finding refresh token: %w", err)
	}

	if record.IsRevoked() {
		return nil // Already revoked — idempotent.
	}

	return s.tokenRepo.Revoke(ctx, record.JTI, nil)
}

// GetMe returns the public profile of the authenticated user.
func (s *authService) GetMe(ctx context.Context, userID string) (*models.PublicUser, error) {
	id, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user id: %w", err)
	}
	user, err := s.userRepo.FindByID(ctx, id)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, repository.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("finding user: %w", err)
	}
	roles, err := s.userRepo.GetRoles(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("getting roles: %w", err)
	}
	user.Roles = roles
	return user.ToPublic(), nil
}

// issueTokenPair creates and stores a new access + refresh token pair.
func (s *authService) issueTokenPair(ctx context.Context, userID string, roles []string, ip, userAgent string, userProfile *models.PublicUser) (*AuthTokens, error) {
	accessToken, _, err := s.jwtMgr.IssueAccessToken(userID, roles)
	if err != nil {
		return nil, fmt.Errorf("issuing access token: %w", err)
	}

	rawRefreshToken, jti, err := s.jwtMgr.IssueRefreshToken(userID)
	if err != nil {
		return nil, fmt.Errorf("issuing refresh token: %w", err)
	}

	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("parsing user id: %w", err)
	}

	now := time.Now()
	record := &models.RefreshToken{
		JTI:       jti,
		UserID:    uid,
		TokenHash: repository.HashToken(rawRefreshToken),
		IssuedAt:  now,
		ExpiresAt: now.Add(7 * 24 * time.Hour), // matches JWT exp
		CreatedIP: ip,
		UserAgent: truncate(userAgent, 512),
	}
	if err := s.tokenRepo.Create(ctx, record); err != nil {
		return nil, fmt.Errorf("storing refresh token: %w", err)
	}

	expiresAt := now.Add(s.accessTTL)
	return &AuthTokens{
		AccessToken:  accessToken,
		RefreshToken: rawRefreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    int(s.accessTTL.Seconds()),
		ExpiresAt:    expiresAt,
		User:         userProfile,
	}, nil
}

// clientIP extracts the real client IP from the request.
func ClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return xff
	}
	return r.RemoteAddr
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max]
}
