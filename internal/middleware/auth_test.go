package middleware_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/avinashtandon/business-tracker-backend/internal/middleware"
	"github.com/avinashtandon/business-tracker-backend/pkg/jwtpkg"
)

func generateTestKeys(t *testing.T) (*rsa.PrivateKey, *rsa.PublicKey) {
	t.Helper()
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generating RSA key: %v", err)
	}
	return priv, &priv.PublicKey
}

func newTestJWTManager(t *testing.T) *jwtpkg.Manager {
	t.Helper()
	priv, pub := generateTestKeys(t)
	return jwtpkg.NewManager(priv, pub, "test-issuer", "test-audience", 15*time.Minute, 7)
}

// okHandler is a simple handler that returns 200 OK.
var okHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
})

// ── Authenticate middleware tests ─────────────────────────────────────────────

func TestAuthenticate_ValidToken_Passes(t *testing.T) {
	mgr := newTestJWTManager(t)
	token, _, err := mgr.IssueAccessToken("user-1", []string{"user"})
	if err != nil {
		t.Fatalf("IssueAccessToken: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()

	middleware.Authenticate(mgr)(okHandler).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
}

func TestAuthenticate_MissingToken_Returns401(t *testing.T) {
	mgr := newTestJWTManager(t)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()

	middleware.Authenticate(mgr)(okHandler).ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestAuthenticate_InvalidToken_Returns401(t *testing.T) {
	mgr := newTestJWTManager(t)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer this.is.not.a.valid.jwt")
	rr := httptest.NewRecorder()

	middleware.Authenticate(mgr)(okHandler).ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestAuthenticate_RefreshTokenAsAccessToken_Returns401(t *testing.T) {
	mgr := newTestJWTManager(t)
	// Issue a refresh token and try to use it as an access token.
	refreshToken, _, err := mgr.IssueRefreshToken("user-2")
	if err != nil {
		t.Fatalf("IssueRefreshToken: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+refreshToken)
	rr := httptest.NewRecorder()

	middleware.Authenticate(mgr)(okHandler).ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 when using refresh token as access token, got %d", rr.Code)
	}
}

// ── RequireRole middleware tests ──────────────────────────────────────────────

func TestRequireRole_CorrectRole_Passes(t *testing.T) {
	mgr := newTestJWTManager(t)
	token, _, err := mgr.IssueAccessToken("admin-1", []string{"admin"})
	if err != nil {
		t.Fatalf("IssueAccessToken: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()

	// Chain: Authenticate → RequireRole("admin") → okHandler
	handler := middleware.Authenticate(mgr)(
		middleware.RequireRole("admin")(okHandler),
	)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 for admin role, got %d", rr.Code)
	}
}

func TestRequireRole_WrongRole_Returns403(t *testing.T) {
	mgr := newTestJWTManager(t)
	// Issue token with "user" role, but require "admin".
	token, _, err := mgr.IssueAccessToken("user-3", []string{"user"})
	if err != nil {
		t.Fatalf("IssueAccessToken: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()

	handler := middleware.Authenticate(mgr)(
		middleware.RequireRole("admin")(okHandler),
	)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("expected 403 for wrong role, got %d", rr.Code)
	}
}

func TestRequireRole_NoClaimsInContext_Returns401(t *testing.T) {
	// Call RequireRole without Authenticate — no claims in context.
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()

	middleware.RequireRole("admin")(okHandler).ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 when no claims in context, got %d", rr.Code)
	}
}

func TestRequireRole_MultipleRoles_PassesIfAnyMatch(t *testing.T) {
	mgr := newTestJWTManager(t)
	// User has "user" and "moderator" roles; endpoint requires "admin" or "moderator".
	token, _, err := mgr.IssueAccessToken("mod-1", []string{"user", "moderator"})
	if err != nil {
		t.Fatalf("IssueAccessToken: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()

	handler := middleware.Authenticate(mgr)(
		middleware.RequireRole("admin", "moderator")(okHandler),
	)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 when user has one of the required roles, got %d", rr.Code)
	}
}

// ── ClaimsFromContext tests ───────────────────────────────────────────────────

func TestClaimsFromContext_ReturnsNilWhenAbsent(t *testing.T) {
	claims := middleware.ClaimsFromContext(context.Background())
	if claims != nil {
		t.Errorf("expected nil claims from empty context, got %+v", claims)
	}
}
