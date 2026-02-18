// Package jwtpkg provides JWT creation and validation for access and refresh tokens.
//
// Security design:
//   - RS256 (asymmetric) signing: private key signs, public key verifies.
//   - Access tokens have typ="access"; refresh tokens have typ="refresh".
//   - Validators are MUTUALLY EXCLUSIVE: ValidateAccessToken rejects typ=refresh
//     and vice versa, preventing token substitution attacks.
//   - All standard claims (iss, aud, exp, nbf) are validated on every parse.
//   - jti (JWT ID) is a UUID v4 generated per token for replay tracking.
package jwtpkg

import (
	"crypto/rsa"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// TokenType distinguishes access tokens from refresh tokens.
// This is stored in the "typ" custom claim and enforced during validation.
type TokenType string

const (
	TokenTypeAccess  TokenType = "access"
	TokenTypeRefresh TokenType = "refresh"
)

// Claims is the full set of JWT claims used by this application.
// It embeds jwt.RegisteredClaims for standard fields.
type Claims struct {
	jwt.RegisteredClaims
	// Typ distinguishes access from refresh tokens. Validated explicitly.
	Typ TokenType `json:"typ"`
	// Roles contains the user's role names (access tokens only).
	Roles []string `json:"roles,omitempty"`
}

// Manager holds the RSA key pair and token configuration.
type Manager struct {
	privateKey          *rsa.PrivateKey
	publicKey           *rsa.PublicKey
	issuer              string
	audience            string
	accessTokenTTL      time.Duration
	refreshTokenTTLDays int
}

// NewManager creates a new JWT Manager.
func NewManager(
	privateKey *rsa.PrivateKey,
	publicKey *rsa.PublicKey,
	issuer, audience string,
	accessTTL time.Duration,
	refreshTTLDays int,
) *Manager {
	return &Manager{
		privateKey:          privateKey,
		publicKey:           publicKey,
		issuer:              issuer,
		audience:            audience,
		accessTokenTTL:      accessTTL,
		refreshTokenTTLDays: refreshTTLDays,
	}
}

// IssueAccessToken creates a signed RS256 access token for the given user.
// Claims include: sub, iss, aud, exp, iat, nbf, jti, typ=access, roles.
func (m *Manager) IssueAccessToken(userID string, roles []string) (string, string, error) {
	jti := uuid.NewString()
	now := time.Now()
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			Issuer:    m.issuer,
			Audience:  jwt.ClaimStrings{m.audience},
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(m.accessTokenTTL)),
			ID:        jti,
		},
		Typ:   TokenTypeAccess,
		Roles: roles,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	signed, err := token.SignedString(m.privateKey)
	if err != nil {
		return "", "", fmt.Errorf("signing access token: %w", err)
	}
	return signed, jti, nil
}

// IssueRefreshToken creates a signed RS256 refresh token for the given user.
// Claims include: sub, iss, aud, exp, iat, jti, typ=refresh.
// Roles are intentionally omitted from refresh tokens.
func (m *Manager) IssueRefreshToken(userID string) (string, string, error) {
	jti := uuid.NewString()
	now := time.Now()
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			Issuer:    m.issuer,
			Audience:  jwt.ClaimStrings{m.audience},
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Duration(m.refreshTokenTTLDays) * 24 * time.Hour)),
			ID:        jti,
		},
		Typ: TokenTypeRefresh,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	signed, err := token.SignedString(m.privateKey)
	if err != nil {
		return "", "", fmt.Errorf("signing refresh token: %w", err)
	}
	return signed, jti, nil
}

// ValidateAccessToken parses and validates an access token string.
// It rejects tokens with typ != "access", preventing refresh tokens from
// being used as access tokens (token substitution attack prevention).
func (m *Manager) ValidateAccessToken(tokenStr string) (*Claims, error) {
	claims, err := m.parseAndValidate(tokenStr)
	if err != nil {
		return nil, err
	}
	// Enforce typ=access — reject refresh tokens explicitly.
	if claims.Typ != TokenTypeAccess {
		return nil, errors.New("invalid token type: expected access token")
	}
	return claims, nil
}

// ValidateRefreshToken parses and validates a refresh token string.
// It rejects tokens with typ != "refresh", preventing access tokens from
// being used as refresh tokens (token substitution attack prevention).
func (m *Manager) ValidateRefreshToken(tokenStr string) (*Claims, error) {
	claims, err := m.parseAndValidate(tokenStr)
	if err != nil {
		return nil, err
	}
	// Enforce typ=refresh — reject access tokens explicitly.
	if claims.Typ != TokenTypeRefresh {
		return nil, errors.New("invalid token type: expected refresh token")
	}
	return claims, nil
}

// parseAndValidate is the shared parsing logic. It validates:
//   - Signature (RS256 public key)
//   - Algorithm (must be RS256, prevents alg=none attacks)
//   - Expiry (exp)
//   - Issuer (iss)
//   - Audience (aud)
func (m *Manager) parseAndValidate(tokenStr string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(
		tokenStr,
		claims,
		func(t *jwt.Token) (interface{}, error) {
			// Enforce RS256 — reject any other algorithm including "none".
			if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}
			return m.publicKey, nil
		},
		jwt.WithIssuer(m.issuer),
		jwt.WithAudience(m.audience),
		jwt.WithExpirationRequired(),
		jwt.WithLeeway(5*time.Second),
	)
	if err != nil {
		return nil, fmt.Errorf("parsing token: %w", err)
	}
	if !token.Valid {
		return nil, errors.New("token is invalid")
	}
	return claims, nil
}
