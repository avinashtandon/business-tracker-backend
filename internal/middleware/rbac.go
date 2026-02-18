package middleware

import (
	"net/http"

	"github.com/linktag/auth-backend/pkg/response"
)

// RequireRole returns a middleware that checks the authenticated user has
// at least one of the specified roles. Must be used after Authenticate.
//
// RBAC enforcement: roles are read from JWT claims (set at login time).
// The JWT is the source of truth for roles during the access token's lifetime.
// Role changes take effect on the next login (or token refresh).
func RequireRole(roles ...string) func(http.Handler) http.Handler {
	allowedRoles := make(map[string]struct{}, len(roles))
	for _, r := range roles {
		allowedRoles[r] = struct{}{}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := ClaimsFromContext(r.Context())
			if claims == nil {
				response.Unauthorized(w, "authentication required")
				return
			}

			for _, role := range claims.Roles {
				if _, ok := allowedRoles[role]; ok {
					next.ServeHTTP(w, r)
					return
				}
			}

			response.Forbidden(w, "insufficient permissions")
		})
	}
}
