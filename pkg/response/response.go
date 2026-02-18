// Package response provides helpers for writing consistent JSON API responses.
package response

import (
	"encoding/json"
	"net/http"
)

// Envelope is the standard JSON response wrapper.
type Envelope struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   *APIError   `json:"error,omitempty"`
}

// APIError is the standard error payload.
type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// JSON writes a JSON response with the given status code and body.
func JSON(w http.ResponseWriter, status int, body interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

// Success writes a successful JSON response.
func Success(w http.ResponseWriter, status int, data interface{}) {
	JSON(w, status, Envelope{Success: true, Data: data})
}

// Error writes an error JSON response with a machine-readable code.
func Error(w http.ResponseWriter, status int, code, message string) {
	JSON(w, status, Envelope{
		Success: false,
		Error:   &APIError{Code: code, Message: message},
	})
}

// ValidationError writes a 422 Unprocessable Entity with validation details.
func ValidationError(w http.ResponseWriter, message string) {
	Error(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", message)
}

// Unauthorized writes a 401 response.
func Unauthorized(w http.ResponseWriter, message string) {
	Error(w, http.StatusUnauthorized, "UNAUTHORIZED", message)
}

// Forbidden writes a 403 response.
func Forbidden(w http.ResponseWriter, message string) {
	Error(w, http.StatusForbidden, "FORBIDDEN", message)
}

// NotFound writes a 404 response.
func NotFound(w http.ResponseWriter, message string) {
	Error(w, http.StatusNotFound, "NOT_FOUND", message)
}

// InternalServerError writes a 500 response (never leaks internal details).
func InternalServerError(w http.ResponseWriter) {
	Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "an unexpected error occurred")
}
