package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
	pkglogger "github.com/linktag/auth-backend/pkg/logger"
)

const requestIDHeader = "X-Request-ID"

// RequestID generates a UUID request ID for each request, injects it into
// the context and response headers, and logs the request with timing info.
func RequestID(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Use existing request ID from header if provided (e.g. by a proxy).
			requestID := r.Header.Get(requestIDHeader)
			if requestID == "" {
				requestID = uuid.NewString()
			}

			// Inject into context and response header.
			ctx := pkglogger.WithRequestID(r.Context(), requestID)
			w.Header().Set(requestIDHeader, requestID)

			// Wrap the ResponseWriter to capture status code.
			rw := &statusRecorder{ResponseWriter: w, status: http.StatusOK}

			start := time.Now()
			next.ServeHTTP(rw, r.WithContext(ctx))
			duration := time.Since(start)

			pkglogger.FromContext(ctx, logger).Info("request",
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Int("status", rw.status),
				slog.Duration("duration", duration),
				slog.String("remote_addr", r.RemoteAddr),
			)
		})
	}
}

// statusRecorder wraps http.ResponseWriter to capture the status code.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

// RequestIDFromCtx is a convenience alias.
func RequestIDFromCtx(ctx context.Context) string {
	return pkglogger.RequestIDFromContext(ctx)
}
