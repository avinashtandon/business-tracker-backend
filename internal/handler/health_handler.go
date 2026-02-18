package handler

import (
	"net/http"

	"github.com/linktag/auth-backend/internal/database"
	"github.com/linktag/auth-backend/pkg/response"
)

// HealthHandler handles health and readiness check endpoints.
type HealthHandler struct {
	db *database.DB
}

// NewHealthHandler creates a new HealthHandler.
func NewHealthHandler(db *database.DB) *HealthHandler {
	return &HealthHandler{db: db}
}

// Healthz handles GET /healthz — liveness probe.
// Returns 200 if the process is alive. No DB check (fast).
func (h *HealthHandler) Healthz(w http.ResponseWriter, r *http.Request) {
	response.Success(w, http.StatusOK, map[string]string{"status": "ok"})
}

// Readyz handles GET /readyz — readiness probe.
// Returns 200 only if the DB is reachable. Used by load balancers.
func (h *HealthHandler) Readyz(w http.ResponseWriter, r *http.Request) {
	if err := h.db.Ping(r.Context()); err != nil {
		response.Error(w, http.StatusServiceUnavailable, "DB_UNAVAILABLE", "database is not reachable")
		return
	}
	response.Success(w, http.StatusOK, map[string]string{"status": "ready", "db": "ok"})
}
