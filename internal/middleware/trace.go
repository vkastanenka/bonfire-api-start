package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"

	"bonfire-api/internal/logger" // Import your standalone logger

	chiMiddleware "github.com/go-chi/chi/v5/middleware"
)

// TelemetryMiddleware unifies request tracking and trace mapping into the logging context.
func TelemetryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// 1. Extract Chi's Request ID and map it to the logger's generic key
		if reqID := chiMiddleware.GetReqID(ctx); reqID != "" {
			ctx = context.WithValue(ctx, logger.ReqIDKey, reqID)
		}

		// 2. Handle the Trace ID logic
		traceID := r.Header.Get("X-Trace-ID")
		if traceID == "" {
			traceID = generateW3CTraceID()
		}
		r.Header.Set("X-Trace-ID", traceID)

		// Map the Trace ID to the logger's generic key
		ctx = context.WithValue(ctx, logger.TraceIDKey, traceID)

		// Pass the adapted context down the chain
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func generateW3CTraceID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// GetTraceID remains handy for internal services needing to fetch the string directly
func GetTraceID(ctx context.Context) string {
	if v, ok := ctx.Value(logger.TraceIDKey).(string); ok {
		return v
	}
	return ""
}
