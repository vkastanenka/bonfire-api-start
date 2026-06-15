package middleware

import (
	"context"
	"log/slog"

	"github.com/go-chi/chi/v5/middleware"
)

// ContextHandler wraps any standard slog.Handler and injects context values like Request IDs.
type ContextHandler struct {
	slog.Handler
}

// NewContextHandler creates a new middleware wrapper for slog.
func NewContextHandler(next slog.Handler) *ContextHandler {
	return &ContextHandler{Handler: next}
}

// Handle overrides the default log writer to inject attributes from the request context.
func (h *ContextHandler) Handle(ctx context.Context, r slog.Record) error {
	if reqID := middleware.GetReqID(ctx); reqID != "" {
		r.AddAttrs(slog.String("request_id", reqID))
	}
	return h.Handler.Handle(ctx, r)
}
