//go:build integration

// Package integration contains integration tests that spin up a real MySQL
// instance using testcontainers-go and test the full auth flow end-to-end.
//
// Run with: go test -tags=integration ./tests/integration/... -v -timeout 120s
package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"crypto/rand"
	"crypto/rsa"
	"errors"
	"log/slog"

	"github.com/avinashtandon/business-tracker-backend/config"
	"github.com/avinashtandon/business-tracker-backend/internal/database"
	"github.com/avinashtandon/business-tracker-backend/internal/repository"
	"github.com/avinashtandon/business-tracker-backend/internal/router"
	"github.com/avinashtandon/business-tracker-backend/internal/service"
	"github.com/avinashtandon/business-tracker-backend/pkg/jwtpkg"
	"github.com/avinashtandon/business-tracker-backend/pkg/logger"
	"github.com/golang-migrate/migrate/v4"
	mysqlmigrate "github.com/golang-migrate/migrate/v4/database/mysql"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/mysql"
)

// testServer holds the test HTTP server and its dependencies.
type testServer struct {
	server *httptest.Server
	db     *database.DB
}

// setupTestServer spins up MySQL in Docker, runs migrations, and returns a
// fully configured test HTTP server.
func setupTestServer(t *testing.T) *testServer {
	t.Helper()
	ctx := context.Background()

	// ── Start MySQL container ─────────────────────────────────────────────────
	mysqlContainer, err := mysql.Run(ctx,
		"mysql:8.0",
		mysql.WithDatabase("business_tracker"),
		mysql.WithUsername("business_tracker"),
		mysql.WithPassword("secret"),
		testcontainers.WithWaitStrategy(
			// Wait until MySQL is ready to accept connections.
			testcontainers.NewLogConsumer(),
		),
	)
	if err != nil {
		t.Fatalf("starting MySQL container: %v", err)
	}
	t.Cleanup(func() {
		if err := mysqlContainer.Terminate(ctx); err != nil {
			t.Logf("terminating MySQL container: %v", err)
		}
	})

	// ── Get connection string ─────────────────────────────────────────────────
	dsn, err := mysqlContainer.ConnectionString(ctx, "parseTime=true", "charset=utf8mb4", "multiStatements=true")
	if err != nil {
		t.Fatalf("getting MySQL DSN: %v", err)
	}

	// ── Connect to DB ─────────────────────────────────────────────────────────
	db, err := database.Connect(ctx, database.Config{
		DSN:                dsn,
		MaxOpenConns:       5,
		MaxIdleConns:       2,
		ConnMaxLifetimeMin: 1,
	})
	if err != nil {
		t.Fatalf("connecting to test DB: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	// ── Run migrations ────────────────────────────────────────────────────────
	driver, err := mysqlmigrate.WithInstance(db.DB.DB, &mysqlmigrate.Config{})
	if err != nil {
		t.Fatalf("creating migrate driver: %v", err)
	}
	m, err := migrate.NewWithDatabaseInstance("file://../../migrations", "mysql", driver)
	if err != nil {
		t.Fatalf("creating migrate instance: %v", err)
	}
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		t.Fatalf("applying migrations: %v", err)
	}

	// ── Generate test RSA keys ────────────────────────────────────────────────
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generating RSA key: %v", err)
	}

	jwtMgr := jwtpkg.NewManager(
		priv, &priv.PublicKey,
		"test-issuer", "test-audience",
		15*time.Minute, 7,
	)

	// ── Wire dependencies ─────────────────────────────────────────────────────
	userRepo := repository.NewUserRepository(db.DB)
	roleRepo := repository.NewRoleRepository(db.DB)
	tokenRepo := repository.NewTokenRepository(db.DB)
	passwordResetRepo := repository.NewPasswordResetRepository(db.DB)
	authSvc := service.NewAuthService(userRepo, roleRepo, tokenRepo, passwordResetRepo, jwtMgr, 15*time.Minute)
	userSvc := service.NewUserService(userRepo)

	log := logger.New(slog.LevelError) // Suppress logs during tests.

	httpRouter := router.New(router.Config{
		DB:             db,
		JWTManager:     jwtMgr,
		AuthService:    authSvc,
		UserService:    userSvc,
		Logger:         log,
		CORSOrigins:    []string{"http://localhost:3000"},
		RateLimitRPS:   100, // High limit for tests.
		RateLimitBurst: 200,
	})

	srv := httptest.NewServer(httpRouter)
	t.Cleanup(srv.Close)

	return &testServer{server: srv, db: db}
}

// post is a helper to make a POST request to the test server.
func (ts *testServer) post(t *testing.T, path string, body interface{}, token string) *http.Response {
	t.Helper()
	b, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshaling request body: %v", err)
	}

	req, err := http.NewRequest(http.MethodPost, ts.server.URL+path, bytes.NewReader(b))
	if err != nil {
		t.Fatalf("creating request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("making request: %v", err)
	}
	return resp
}

// get is a helper to make a GET request to the test server.
func (ts *testServer) get(t *testing.T, path, token string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, ts.server.URL+path, nil)
	if err != nil {
		t.Fatalf("creating request: %v", err)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("making request: %v", err)
	}
	return resp
}

// decodeResponse decodes the JSON response body into a map.
func decodeResponse(t *testing.T, resp *http.Response) map[string]interface{} {
	t.Helper()
	defer resp.Body.Close()
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	return result
}

// ── Integration tests ─────────────────────────────────────────────────────────

func TestRegisterLoginRefreshLogout(t *testing.T) {
	ts := setupTestServer(t)

	email := fmt.Sprintf("test-%d@example.com", time.Now().UnixNano())
	password := "SecurePass123!"

	// ── 1. Register ───────────────────────────────────────────────────────────
	t.Run("register", func(t *testing.T) {
		resp := ts.post(t, "/api/v1/auth/register", map[string]string{
			"email":      email,
			"username":   "testuser",
			"first_name": "John",
			"last_name":  "Doe",
			"password":   password,
		}, "")

		if resp.StatusCode != http.StatusCreated {
			t.Errorf("expected 201, got %d", resp.StatusCode)
		}
		body := decodeResponse(t, resp)
		if body["success"] != true {
			t.Errorf("expected success=true, got %v", body["success"])
		}
	})

	// ── 2. Register duplicate email → 409 ────────────────────────────────────
	t.Run("register_duplicate_email", func(t *testing.T) {
		resp := ts.post(t, "/api/v1/auth/register", map[string]string{
			"email":      email,
			"username":   "testuser",
			"first_name": "John",
			"last_name":  "Doe",
			"password":   password,
		}, "")
		if resp.StatusCode != http.StatusConflict {
			t.Errorf("expected 409 for duplicate email, got %d", resp.StatusCode)
		}
		resp.Body.Close()
	})

	// ── 3. Login ──────────────────────────────────────────────────────────────
	var accessToken, refreshToken string
	t.Run("login", func(t *testing.T) {
		resp := ts.post(t, "/api/v1/auth/login", map[string]string{
			"email":    email,
			"password": password,
		}, "")

		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected 200, got %d", resp.StatusCode)
		}
		body := decodeResponse(t, resp)
		data, ok := body["data"].(map[string]interface{})
		if !ok {
			t.Fatal("expected data object in response")
		}
		accessToken, _ = data["access_token"].(string)
		refreshToken, _ = data["refresh_token"].(string)
		if accessToken == "" {
			t.Error("expected non-empty access_token")
		}
		if refreshToken == "" {
			t.Error("expected non-empty refresh_token")
		}
	})

	// ── 4. Login with wrong password → 401 ───────────────────────────────────
	t.Run("login_wrong_password", func(t *testing.T) {
		resp := ts.post(t, "/api/v1/auth/login", map[string]string{
			"email":    email,
			"password": "wrongpassword",
		}, "")
		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("expected 401 for wrong password, got %d", resp.StatusCode)
		}
		resp.Body.Close()
	})

	// ── 5. Get /me ────────────────────────────────────────────────────────────
	t.Run("me", func(t *testing.T) {
		resp := ts.get(t, "/api/v1/auth/me", accessToken)
		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected 200 for /me, got %d", resp.StatusCode)
		}
		body := decodeResponse(t, resp)
		data, ok := body["data"].(map[string]interface{})
		if !ok {
			t.Fatal("expected data object in /me response")
		}
		if data["email"] != email {
			t.Errorf("expected email=%s, got %v", email, data["email"])
		}
	})

	// ── 6. Refresh tokens ─────────────────────────────────────────────────────
	var newAccessToken, newRefreshToken string
	t.Run("refresh", func(t *testing.T) {
		resp := ts.post(t, "/api/v1/auth/refresh", map[string]string{
			"refresh_token": refreshToken,
		}, "")
		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected 200 for refresh, got %d", resp.StatusCode)
		}
		body := decodeResponse(t, resp)
		data, ok := body["data"].(map[string]interface{})
		if !ok {
			t.Fatal("expected data object in refresh response")
		}
		newAccessToken, _ = data["access_token"].(string)
		newRefreshToken, _ = data["refresh_token"].(string)
		if newAccessToken == "" {
			t.Error("expected non-empty new access_token")
		}
		if newRefreshToken == refreshToken {
			t.Error("expected new refresh_token to differ from old one (rotation)")
		}
	})

	// ── 7. Old refresh token should be revoked after rotation ─────────────────
	t.Run("old_refresh_token_revoked", func(t *testing.T) {
		resp := ts.post(t, "/api/v1/auth/refresh", map[string]string{
			"refresh_token": refreshToken, // old token
		}, "")
		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("expected 401 for revoked refresh token, got %d", resp.StatusCode)
		}
		resp.Body.Close()
	})

	// ── 8. New access token works ─────────────────────────────────────────────
	t.Run("new_access_token_works", func(t *testing.T) {
		resp := ts.get(t, "/api/v1/auth/me", newAccessToken)
		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected 200 with new access token, got %d", resp.StatusCode)
		}
		resp.Body.Close()
	})

	// ── 9. Logout ─────────────────────────────────────────────────────────────
	t.Run("logout", func(t *testing.T) {
		resp := ts.post(t, "/api/v1/auth/logout", map[string]string{
			"refresh_token": newRefreshToken,
		}, "")
		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected 200 for logout, got %d", resp.StatusCode)
		}
		resp.Body.Close()
	})

	// ── 10. Refresh after logout should fail ──────────────────────────────────
	t.Run("refresh_after_logout_fails", func(t *testing.T) {
		resp := ts.post(t, "/api/v1/auth/refresh", map[string]string{
			"refresh_token": newRefreshToken,
		}, "")
		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("expected 401 after logout, got %d", resp.StatusCode)
		}
		resp.Body.Close()
	})
}

func TestHealthEndpoints(t *testing.T) {
	ts := setupTestServer(t)

	t.Run("healthz", func(t *testing.T) {
		resp := ts.get(t, "/healthz", "")
		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected 200 for /healthz, got %d", resp.StatusCode)
		}
		resp.Body.Close()
	})

	t.Run("readyz", func(t *testing.T) {
		resp := ts.get(t, "/readyz", "")
		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected 200 for /readyz, got %d", resp.StatusCode)
		}
		resp.Body.Close()
	})
}

func TestAdminEndpoint_RequiresAdminRole(t *testing.T) {
	ts := setupTestServer(t)

	email := fmt.Sprintf("user-%d@example.com", time.Now().UnixNano())

	// Register a regular user.
	ts.post(t, "/api/v1/auth/register", map[string]string{
		"email":      email,
		"username":   "testuser",
		"first_name": "John",
		"last_name":  "Doe",
		"password":   "SecurePass123!",
	}, "").Body.Close()

	// Login.
	resp := ts.post(t, "/api/v1/auth/login", map[string]string{
		"email":    email,
		"password": "SecurePass123!",
	}, "")
	body := decodeResponse(t, resp)
	data := body["data"].(map[string]interface{})
	accessToken := data["access_token"].(string)

	// Regular user should get 403 on admin endpoint.
	adminResp := ts.get(t, "/api/v1/admin/users", accessToken)
	if adminResp.StatusCode != http.StatusForbidden {
		t.Errorf("expected 403 for regular user on admin endpoint, got %d", adminResp.StatusCode)
	}
	adminResp.Body.Close()
}

// Ensure config package compiles (used indirectly via setupTestServer).
var _ = config.Load
