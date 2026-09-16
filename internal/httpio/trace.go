package httpio

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"strings"

	"bonfire-api/internal/pkg/logger"

	"github.com/go-chi/chi/v5/middleware"
)

// Trace attaches request correlation and trace identifiers to context and response headers.
func Trace(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Resolve Request ID
		reqID := middleware.GetReqID(ctx)
		if reqID == "" {
			reqID = r.Header.Get("X-Request-ID")
		}
		if reqID == "" {
			reqID = generateID(16)
		}

		// Resolve Trace ID
		traceID := extractW3CTraceID(r.Header.Get("traceparent"))
		if traceID == "" {
			traceID = r.Header.Get("X-Trace-ID")
		}
		if traceID == "" {
			traceID = generateID(16)
		}

		// Set response headers for client correlation
		w.Header().Set("X-Request-ID", reqID)
		w.Header().Set("X-Trace-ID", traceID)

		// Inject correlation identifiers into context and structured logger
		ctx = context.WithValue(ctx, ctxKeyReqID, reqID)
		ctx = context.WithValue(ctx, ctxKeyTraceID, traceID)

		ctx = logger.WithContextAttrs(ctx,
			slog.String("request_id", reqID),
			slog.String("trace_id", traceID),
		)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// generateID returns a random hex string of the specified byte length.
func generateID(byteLen int) string {
	b := make([]byte, byteLen)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// extractW3CTraceID extracts the 32-character trace ID from a W3C traceparent header.
func extractW3CTraceID(traceparent string) string {
	parts := strings.Split(traceparent, "-")
	if len(parts) == 4 && len(parts[1]) == 32 {
		return parts[1]
	}
	return ""
}
