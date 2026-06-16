package middleware

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
)

func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Wrap the standard ResponseWriter
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

		// Process the request downstream
		next.ServeHTTP(ww, r)

		// Calculate request length
		latency := time.Since(start)

		// Determine log level based on the HTTP status code
		status := ww.Status()
		logFn := slog.InfoContext

		if status >= 500 {
			logFn = slog.ErrorContext
		} else if status >= 400 {
			logFn = slog.WarnContext
		}

		// Log entry per request completion
		logFn(r.Context(), "http request processed",
			"method", r.Method,
			"path", r.URL.Path,
			"status", status,
			"latency_ms", latency.Milliseconds(),
			"bytes_written", ww.BytesWritten(),
		)
	})
}
