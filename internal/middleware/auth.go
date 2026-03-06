// Package middleware provides HTTP middleware for the application.
package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/avinashtandon/business-tracker-backend/pkg/jwtpkg"
	"github.com/avinashtandon/business-tracker-backend/pkg/response"
)

type contextKey string

const (
	// ClaimsKey is the context key for storing JWT claims.
	ClaimsKey contextKey = "jwt_claims"
)

// Authenticate is a middleware that validates the Bearer access token.
// It rejects requests with missing, malformed, or invalid tokens.
// On success, it injects the parsed Claims into the request context.
func Authenticate(jwtMgr *jwtpkg.Manager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				response.Unauthorized(w, "authorization header is required")
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
				response.Unauthorized(w, "authorization header must be 'Bearer <token>'")
				return
			}

			claims, err := jwtMgr.ValidateAccessToken(parts[1])
			if err != nil {
				response.Unauthorized(w, "invalid or expired access token")
				return
			}

			// Inject claims into context for downstream handlers.
			ctx := context.WithValue(r.Context(), ClaimsKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// ClaimsFromContext retrieves JWT claims from the request context.
// Returns nil if not present (i.e., unauthenticated request).
func ClaimsFromContext(ctx context.Context) *jwtpkg.Claims {
	claims, _ := ctx.Value(ClaimsKey).(*jwtpkg.Claims)
	return claims
}
