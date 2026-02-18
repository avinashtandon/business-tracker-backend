// Package handler implements HTTP request handlers for the auth API.
package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/linktag/auth-backend/internal/middleware"
	"github.com/linktag/auth-backend/internal/service"
	"github.com/linktag/auth-backend/pkg/response"
	"github.com/linktag/auth-backend/pkg/validator"
)

// AuthHandler handles authentication-related HTTP requests.
type AuthHandler struct {
	authSvc service.AuthService
}

// NewAuthHandler creates a new AuthHandler.
func NewAuthHandler(authSvc service.AuthService) *AuthHandler {
	return &AuthHandler{authSvc: authSvc}
}

// Register handles POST /api/v1/auth/register
//
// Request:  {"email": "user@example.com", "password": "SecurePass123!"}
// Response: {"success": true, "data": {"id": "...", "email": "...", "roles": ["user"], ...}}
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var input service.RegisterInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		response.ValidationError(w, "invalid JSON body")
		return
	}

	if err := validator.Validate(input); err != nil {
		response.ValidationError(w, err.Error())
		return
	}

	user, err := h.authSvc.Register(r.Context(), input)
	if err != nil {
		switch err {
		case service.ErrEmailTaken:
			response.Error(w, http.StatusConflict, "EMAIL_TAKEN", "email address is already registered")
		default:
			response.InternalServerError(w)
		}
		return
	}

	response.Success(w, http.StatusCreated, user)
}

// Login handles POST /api/v1/auth/login
//
// Request:  {"email": "user@example.com", "password": "SecurePass123!"}
// Response: {"success": true, "data": {"access_token": "...", "refresh_token": "...", "expires_in": 900}}
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var input service.LoginInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		response.ValidationError(w, "invalid JSON body")
		return
	}

	if err := validator.Validate(input); err != nil {
		response.ValidationError(w, err.Error())
		return
	}

	ip := clientIP(r)
	userAgent := r.UserAgent()

	tokens, err := h.authSvc.Login(r.Context(), input, ip, userAgent)
	if err != nil {
		switch err {
		case service.ErrInvalidCredentials:
			response.Error(w, http.StatusUnauthorized, "INVALID_CREDENTIALS", "invalid email or password")
		case service.ErrUserInactive:
			response.Error(w, http.StatusForbidden, "ACCOUNT_INACTIVE", "account is not active")
		default:
			response.InternalServerError(w)
		}
		return
	}

	response.Success(w, http.StatusOK, tokens)
}

// Refresh handles POST /api/v1/auth/refresh
//
// Request:  {"refresh_token": "eyJ..."}
// Response: {"success": true, "data": {"access_token": "...", "refresh_token": "...", "expires_in": 900}}
func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var body struct {
		RefreshToken string `json:"refresh_token" validate:"required"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.ValidationError(w, "invalid JSON body")
		return
	}
	if err := validator.Validate(body); err != nil {
		response.ValidationError(w, err.Error())
		return
	}

	ip := clientIP(r)
	userAgent := r.UserAgent()

	tokens, err := h.authSvc.RefreshTokens(r.Context(), body.RefreshToken, ip, userAgent)
	if err != nil {
		switch err {
		case service.ErrTokenInvalid:
			response.Error(w, http.StatusUnauthorized, "TOKEN_INVALID", "refresh token is invalid or expired")
		case service.ErrTokenRevoked:
			response.Error(w, http.StatusUnauthorized, "TOKEN_REVOKED", "refresh token has been revoked")
		default:
			response.InternalServerError(w)
		}
		return
	}

	response.Success(w, http.StatusOK, tokens)
}

// Logout handles POST /api/v1/auth/logout
//
// Request:  {"refresh_token": "eyJ..."}
// Response: {"success": true, "data": {"message": "logged out successfully"}}
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	var body struct {
		RefreshToken string `json:"refresh_token" validate:"required"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.ValidationError(w, "invalid JSON body")
		return
	}
	if err := validator.Validate(body); err != nil {
		response.ValidationError(w, err.Error())
		return
	}

	// Logout is idempotent — always returns success.
	_ = h.authSvc.Logout(r.Context(), body.RefreshToken)
	response.Success(w, http.StatusOK, map[string]string{"message": "logged out successfully"})
}

// Me handles GET /api/v1/auth/me
//
// Response: {"success": true, "data": {"id": "...", "email": "...", "roles": [...], ...}}
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if claims == nil {
		response.Unauthorized(w, "authentication required")
		return
	}

	user, err := h.authSvc.GetMe(r.Context(), claims.Subject)
	if err != nil {
		response.InternalServerError(w)
		return
	}

	response.Success(w, http.StatusOK, user)
}

// clientIP extracts the real client IP from the request.
func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// Take the first IP in the chain.
		parts := strings.SplitN(xff, ",", 2)
		return strings.TrimSpace(parts[0])
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	return r.RemoteAddr
}
