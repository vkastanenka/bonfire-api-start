package httpio

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
)

func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

		next.ServeHTTP(ww, r)

		latency := time.Since(start)

		status := ww.Status()
		logFn := slog.InfoContext

		if status >= 500 {
			logFn = slog.ErrorContext
		} else if status >= 400 {
			logFn = slog.WarnContext
		}

		logFn(r.Context(), "http request processed",
			"method", r.Method,
			"path", r.URL.Path,
			"status", status,
			"latency_ms", latency.Milliseconds(),
			"bytes_written", ww.BytesWritten(),
		)
	})
}
