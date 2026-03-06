package handler

import (
	"net/http"

	"github.com/avinashtandon/business-tracker-backend/internal/service"
	"github.com/avinashtandon/business-tracker-backend/pkg/response"
)

// AdminHandler handles admin-only HTTP requests.
type AdminHandler struct {
	userSvc service.UserService
}

// NewAdminHandler creates a new AdminHandler.
func NewAdminHandler(userSvc service.UserService) *AdminHandler {
	return &AdminHandler{userSvc: userSvc}
}

// ListUsers handles GET /api/v1/admin/users
// Requires admin role (enforced by RBAC middleware in the router).
//
// Response: {"success": true, "data": {"users": [...], "total": N}}
func (h *AdminHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := h.userSvc.ListUsers(r.Context())
	if err != nil {
		response.InternalServerError(w)
		return
	}

	response.Success(w, http.StatusOK, map[string]interface{}{
		"users": users,
		"total": len(users),
	})
}
