package jwtpkg_test

import (
	"crypto/rand"
	"crypto/rsa"
	"testing"
	"time"

	"github.com/linktag/auth-backend/pkg/jwtpkg"
)

// generateTestKeys creates a fresh RSA-2048 key pair for testing.
func generateTestKeys(t *testing.T) (*rsa.PrivateKey, *rsa.PublicKey) {
	t.Helper()
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generating RSA key: %v", err)
	}
	return priv, &priv.PublicKey
}

// newTestManager creates a JWT Manager with test keys and short TTLs.
func newTestManager(t *testing.T) *jwtpkg.Manager {
	t.Helper()
	priv, pub := generateTestKeys(t)
	return jwtpkg.NewManager(priv, pub, "test-issuer", "test-audience", 15*time.Minute, 7)
}

func TestIssueAndValidateAccessToken(t *testing.T) {
	mgr := newTestManager(t)

	token, jti, err := mgr.IssueAccessToken("user-123", []string{"user"})
	if err != nil {
		t.Fatalf("IssueAccessToken: %v", err)
	}
	if token == "" {
		t.Fatal("expected non-empty token")
	}
	if jti == "" {
		t.Fatal("expected non-empty jti")
	}

	claims, err := mgr.ValidateAccessToken(token)
	if err != nil {
		t.Fatalf("ValidateAccessToken: %v", err)
	}

	if claims.Subject != "user-123" {
		t.Errorf("expected sub=user-123, got %s", claims.Subject)
	}
	if claims.Typ != jwtpkg.TokenTypeAccess {
		t.Errorf("expected typ=access, got %s", claims.Typ)
	}
	if len(claims.Roles) != 1 || claims.Roles[0] != "user" {
		t.Errorf("expected roles=[user], got %v", claims.Roles)
	}
	if claims.ID != jti {
		t.Errorf("expected jti=%s, got %s", jti, claims.ID)
	}
}

func TestIssueAndValidateRefreshToken(t *testing.T) {
	mgr := newTestManager(t)

	token, jti, err := mgr.IssueRefreshToken("user-456")
	if err != nil {
		t.Fatalf("IssueRefreshToken: %v", err)
	}

	claims, err := mgr.ValidateRefreshToken(token)
	if err != nil {
		t.Fatalf("ValidateRefreshToken: %v", err)
	}

	if claims.Subject != "user-456" {
		t.Errorf("expected sub=user-456, got %s", claims.Subject)
	}
	if claims.Typ != jwtpkg.TokenTypeRefresh {
		t.Errorf("expected typ=refresh, got %s", claims.Typ)
	}
	if claims.ID != jti {
		t.Errorf("expected jti=%s, got %s", jti, claims.ID)
	}
	// Refresh tokens must NOT contain roles.
	if len(claims.Roles) != 0 {
		t.Errorf("refresh token should not contain roles, got %v", claims.Roles)
	}
}

// TestTokenSubstitutionPrevention verifies that access tokens cannot be used
// as refresh tokens and vice versa (token substitution attack prevention).
func TestTokenSubstitutionPrevention(t *testing.T) {
	mgr := newTestManager(t)

	accessToken, _, err := mgr.IssueAccessToken("user-789", []string{"admin"})
	if err != nil {
		t.Fatalf("IssueAccessToken: %v", err)
	}

	refreshToken, _, err := mgr.IssueRefreshToken("user-789")
	if err != nil {
		t.Fatalf("IssueRefreshToken: %v", err)
	}

	t.Run("access_token_rejected_as_refresh", func(t *testing.T) {
		_, err := mgr.ValidateRefreshToken(accessToken)
		if err == nil {
			t.Error("expected error when using access token as refresh token, got nil")
		}
	})

	t.Run("refresh_token_rejected_as_access", func(t *testing.T) {
		_, err := mgr.ValidateAccessToken(refreshToken)
		if err == nil {
			t.Error("expected error when using refresh token as access token, got nil")
		}
	})
}

// TestExpiredTokenRejected verifies that expired tokens are rejected.
func TestExpiredTokenRejected(t *testing.T) {
	priv, pub := generateTestKeys(t)
	// Create manager with 1 nanosecond TTL so tokens expire immediately.
	mgr := jwtpkg.NewManager(priv, pub, "test-issuer", "test-audience", time.Nanosecond, 0)

	token, _, err := mgr.IssueAccessToken("user-exp", []string{"user"})
	if err != nil {
		t.Fatalf("IssueAccessToken: %v", err)
	}

	// Wait for token to expire (plus leeway).
	time.Sleep(10 * time.Second)

	_, err = mgr.ValidateAccessToken(token)
	if err == nil {
		t.Error("expected error for expired token, got nil")
	}
}

// TestWrongSignatureRejected verifies that tokens signed with a different key are rejected.
func TestWrongSignatureRejected(t *testing.T) {
	priv1, pub1 := generateTestKeys(t)
	priv2, _ := generateTestKeys(t)

	// Sign with key1 private, but validate with key2 public — should fail.
	mgr1 := jwtpkg.NewManager(priv1, pub1, "test-issuer", "test-audience", 15*time.Minute, 7)
	_, pub2 := priv2, &priv2.PublicKey
	mgr2 := jwtpkg.NewManager(priv2, pub2, "test-issuer", "test-audience", 15*time.Minute, 7)

	token, _, err := mgr1.IssueAccessToken("user-sig", []string{"user"})
	if err != nil {
		t.Fatalf("IssueAccessToken: %v", err)
	}

	_, err = mgr2.ValidateAccessToken(token)
	if err == nil {
		t.Error("expected error for wrong signature, got nil")
	}
}

// TestWrongIssuerRejected verifies that tokens with wrong issuer are rejected.
func TestWrongIssuerRejected(t *testing.T) {
	priv, pub := generateTestKeys(t)
	mgr1 := jwtpkg.NewManager(priv, pub, "issuer-A", "test-audience", 15*time.Minute, 7)
	mgr2 := jwtpkg.NewManager(priv, pub, "issuer-B", "test-audience", 15*time.Minute, 7)

	token, _, err := mgr1.IssueAccessToken("user-iss", []string{"user"})
	if err != nil {
		t.Fatalf("IssueAccessToken: %v", err)
	}

	_, err = mgr2.ValidateAccessToken(token)
	if err == nil {
		t.Error("expected error for wrong issuer, got nil")
	}
}
