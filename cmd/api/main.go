// cmd/api/main.go is the application entry point.
// It bootstraps configuration, database, migrations, and the HTTP server.
// Graceful shutdown is handled via OS signal interception.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"

	"github.com/avinashtandon/business-tracker-backend/config"
	"github.com/avinashtandon/business-tracker-backend/internal/database"
	"github.com/avinashtandon/business-tracker-backend/internal/repository"
	"github.com/avinashtandon/business-tracker-backend/internal/router"
	"github.com/avinashtandon/business-tracker-backend/internal/service"
	"github.com/avinashtandon/business-tracker-backend/pkg/jwtpkg"
	"github.com/avinashtandon/business-tracker-backend/pkg/logger"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/mysql"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func main() {
	// Auto-load .env if it exists. In production, env vars are set externally
	// (e.g. docker-compose, Kubernetes secrets) so this is a no-op there.
	_ = godotenv.Load()

	log := logger.New(slog.LevelInfo)

	if err := run(log); err != nil {
		log.Error("application error", "error", err)
		os.Exit(1)
	}
}

func run(log *slog.Logger) error {
	// ── Load configuration ────────────────────────────────────────────────────
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}
	log.Info("configuration loaded", "env", cfg.App.Env, "port", cfg.App.Port)

	// ── Connect to database ───────────────────────────────────────────────────
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	db, err := database.Connect(ctx, database.Config{
		DSN:                cfg.DB.DSN(),
		MaxOpenConns:       cfg.DB.MaxOpenConns,
		MaxIdleConns:       cfg.DB.MaxIdleConns,
		ConnMaxLifetimeMin: cfg.DB.ConnMaxLifetimeMin,
	})
	if err != nil {
		return fmt.Errorf("connecting to database: %w", err)
	}
	defer db.Close()
	log.Info("database connected")

	// ── Run migrations ────────────────────────────────────────────────────────
	if err := runMigrations(db, cfg.Migrations.Path, log); err != nil {
		return fmt.Errorf("running migrations: %w", err)
	}

	// ── Wire dependencies ─────────────────────────────────────────────────────
	jwtMgr := jwtpkg.NewManager(
		cfg.JWT.PrivateKey,
		cfg.JWT.PublicKey,
		cfg.JWT.Issuer,
		cfg.JWT.Audience,
		cfg.JWT.AccessTokenTTL,
		cfg.JWT.RefreshTokenTTLDays,
	)

	userRepo := repository.NewUserRepository(db.DB)
	roleRepo := repository.NewRoleRepository(db.DB)
	tokenRepo := repository.NewTokenRepository(db.DB)
	loanRepo := repository.NewLoanRepository(db.DB)

	authSvc := service.NewAuthService(userRepo, roleRepo, tokenRepo, jwtMgr, cfg.JWT.AccessTokenTTL)
	userSvc := service.NewUserService(userRepo)
	loanSvc := service.NewLoanService(loanRepo)

	// ── Build router ──────────────────────────────────────────────────────────
	httpRouter := router.New(router.Config{
		DB:             db,
		JWTManager:     jwtMgr,
		AuthService:    authSvc,
		UserService:    userSvc,
		LoanService:    loanSvc,
		Logger:         log,
		CORSOrigins:    cfg.CORS.AllowedOrigins,
		RateLimitRPS:   cfg.RateLimit.RPS,
		RateLimitBurst: cfg.RateLimit.Burst,
	})

	// ── Start HTTP server ─────────────────────────────────────────────────────
	srv := &http.Server{
		Addr:         ":" + cfg.App.Port,
		Handler:      httpRouter,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Channel to receive server errors.
	serverErr := make(chan error, 1)
	go func() {
		log.Info("server starting", "addr", srv.Addr)
		serverErr <- srv.ListenAndServe()
	}()

	// ── Graceful shutdown ─────────────────────────────────────────────────────
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-serverErr:
		if !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("server error: %w", err)
		}
	case sig := <-quit:
		log.Info("shutdown signal received", "signal", sig)
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	log.Info("shutting down server gracefully...")
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("server shutdown: %w", err)
	}

	log.Info("server stopped cleanly")
	return nil
}

// runMigrations applies all pending database migrations.
func runMigrations(db *database.DB, migrationsPath string, log *slog.Logger) error {
	driver, err := mysql.WithInstance(db.DB.DB, &mysql.Config{})
	if err != nil {
		return fmt.Errorf("creating migrate driver: %w", err)
	}

	m, err := migrate.NewWithDatabaseInstance(migrationsPath, "mysql", driver)
	if err != nil {
		return fmt.Errorf("creating migrate instance: %w", err)
	}

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("applying migrations: %w", err)
	}

	version, dirty, _ := m.Version()
	log.Info("migrations applied", "version", version, "dirty", dirty)
	return nil
}
