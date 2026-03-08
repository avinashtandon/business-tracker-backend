package service

import (
	"context"
	"log/slog"
	"time"

	"github.com/avinashtandon/business-tracker-backend/internal/repository"
)

// StartTokenCleanup starts a background worker that prunes expired and old revoked tokens.
// It runs once every 24 hours.
func StartTokenCleanup(ctx context.Context, log *slog.Logger, tokenRepo repository.TokenRepository) {
	go func() {
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()

		// Run once on startup
		if err := tokenRepo.DeleteExpired(ctx); err != nil {
			log.Error("failed to cleanup expired tokens on startup", "error", err)
		} else {
			log.Info("completed initial token cleanup")
		}

		for {
			select {
			case <-ctx.Done():
				log.Info("stopping token cleanup worker")
				return
			case <-ticker.C:
				log.Info("running scheduled token cleanup")
				if err := tokenRepo.DeleteExpired(ctx); err != nil {
					log.Error("failed to cleanup expired tokens", "error", err)
				}
			}
		}
	}()
}
