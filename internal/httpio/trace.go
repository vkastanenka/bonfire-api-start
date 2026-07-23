package httpio

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
)

func Trace(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		if reqID := middleware.GetReqID(ctx); reqID != "" {
			ctx = context.WithValue(ctx, CtxReqIDKey, reqID)
		}

		traceID := r.Header.Get("X-Trace-ID")
		if traceID == "" {
			traceID = generateW3CTraceID()
		}
		r.Header.Set("X-Trace-ID", traceID)

		ctx = context.WithValue(ctx, CtxTraceIDKey, traceID)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func generateW3CTraceID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
