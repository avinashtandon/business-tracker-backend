// Package router wires all HTTP routes with their middleware chains.
//
// Framework choice: Chi
// Chi is chosen over Gin because:
//   - It uses stdlib http.Handler/http.HandlerFunc — no framework lock-in.
//   - Middleware is composable via chi.Use() and inline route-level middleware.
//   - Excellent performance, zero allocations on routing.
//   - Easy to test with net/http/httptest (no special Gin test context needed).
package router

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/avinashtandon/business-tracker-backend/internal/database"
	"github.com/avinashtandon/business-tracker-backend/internal/handler"
	"github.com/avinashtandon/business-tracker-backend/internal/middleware"
	"github.com/avinashtandon/business-tracker-backend/internal/models"
	"github.com/avinashtandon/business-tracker-backend/internal/service"
	"github.com/avinashtandon/business-tracker-backend/pkg/jwtpkg"
	"github.com/avinashtandon/business-tracker-backend/pkg/ratelimit"
	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
)

// Config holds all dependencies needed to build the router.
type Config struct {
	DB             *database.DB
	JWTManager     *jwtpkg.Manager
	AuthService    service.AuthService
	UserService    service.UserService
	LoanService    service.LoanService
	Logger         *slog.Logger
	CORSOrigins    []string
	RateLimitRPS   float64
	RateLimitBurst int
}

// New builds and returns the fully configured Chi router.
func New(cfg Config) http.Handler {
	r := chi.NewRouter()

	// ── Global middleware ─────────────────────────────────────────────────────
	r.Use(chimiddleware.Recoverer)
	r.Use(middleware.RequestID(cfg.Logger))
	r.Use(middleware.CORS(cfg.CORSOrigins))

	// ── Health endpoints (no auth) ────────────────────────────────────────────
	healthHandler := handler.NewHealthHandler(cfg.DB)
	r.Get("/healthz", healthHandler.Healthz)
	r.Get("/readyz", healthHandler.Readyz)

	// ── Rate limiters for sensitive endpoints ─────────────────────────────────
	authLimiter := ratelimit.New(cfg.RateLimitRPS, cfg.RateLimitBurst)

	// ── Hello World endpoint (rate-limited) ───────────────────────────────────
	r.With(authLimiter.Middleware).Get("/hello", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"message": "Hello World"})
	})

	// ── Rate limiters for sensitive endpoints ─────────────────────────────────
	r.With(authLimiter.Middleware).Get("/test", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"message": "Test"})
	})
	// ── Auth handlers ─────────────────────────────────────────────────────────
	authHandler := handler.NewAuthHandler(cfg.AuthService)
	adminHandler := handler.NewAdminHandler(cfg.UserService)
	loanHandler := handler.NewLoanHandler(cfg.LoanService)

	// ── API v1 routes ─────────────────────────────────────────────────────────
	r.Route("/api/v1", func(r chi.Router) {

		// Auth routes — some rate-limited, some require auth
		r.Route("/auth", func(r chi.Router) {
			// Rate-limited public endpoints
			r.With(authLimiter.Middleware).Post("/register", authHandler.Register)
			r.With(authLimiter.Middleware).Post("/login", authHandler.Login)
			r.With(authLimiter.Middleware).Post("/refresh", authHandler.Refresh)

			// Logout: requires valid refresh token in body (no access token needed)
			r.Post("/logout", authHandler.Logout)

			// Protected: requires valid access token
			r.Group(func(r chi.Router) {
				r.Use(middleware.Authenticate(cfg.JWTManager))
				r.Get("/me", authHandler.Me)
			})
		})

		// Admin routes — require auth + admin role
		r.Route("/admin", func(r chi.Router) {
			r.Use(middleware.Authenticate(cfg.JWTManager))
			r.Use(middleware.RequireRole(models.RoleAdmin))
			r.Get("/users", adminHandler.ListUsers)
		})

		// Loan & Transaction routes — require auth
		r.Route("/loans", func(r chi.Router) {
			r.Use(middleware.Authenticate(cfg.JWTManager))

			// Loan CRU(D)
			r.Post("/", loanHandler.Create)
			r.Get("/", loanHandler.List)
			r.Get("/{id}", loanHandler.Get)
			r.Put("/{id}", loanHandler.Update)
			r.Delete("/{id}", loanHandler.Delete)

			// Transaction Routes
			r.Post("/{loan_id}/transactions", loanHandler.CreateTransaction)
			r.Delete("/{loan_id}/transactions/{transaction_id}", loanHandler.DeleteTransaction)
		})
	})

	return r
}
