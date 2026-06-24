package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
)

type contextKey string

const TraceIDKey contextKey = "trace_id"

func TracingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		traceID := r.Header.Get("X-Trace-ID")
		if traceID == "" {
			traceID = generateW3CTraceID()
		}

		// Keep the header in sync so downstream services called by this API can forward it
		r.Header.Set("X-Trace-ID", traceID)

		// Inject into context so slog and your handlers can access it natively
		ctx := context.WithValue(r.Context(), TraceIDKey, traceID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func generateW3CTraceID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// GetTraceID fetches the active trace identity from a context safely
func GetTraceID(ctx context.Context) string {
	if v, ok := ctx.Value(TraceIDKey).(string); ok {
		return v
	}
	return ""
}
