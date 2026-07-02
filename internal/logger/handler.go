package logger

import (
	"context"
	"log/slog"

	customMiddleware "bonfire-api/internal/middleware"

	"github.com/go-chi/chi/v5/middleware"
)

// Handler wraps a slog.Handler and injects HTTP data into logs.
type Handler struct {
	slog.Handler
}

// NewHandler creates a new middleware wrapper for slog.
func NewHandler(handler slog.Handler) *Handler {
	return &Handler{Handler: handler}
}

// Handle overrides the default log writer to inject attributes from the request context.
func (h *Handler) Handle(ctx context.Context, r slog.Record) error {
	if reqID := middleware.GetReqID(ctx); reqID != "" {
		r.AddAttrs(slog.String("request_id", reqID))
	}
	if traceID := customMiddleware.GetTraceID(ctx); traceID != "" {
		r.AddAttrs(slog.String("trace_id", traceID))
	}
	return h.Handler.Handle(ctx, r)
}

// WithAttrs ensures sub-loggers retain the custom context-handling logic.
func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &Handler{Handler: h.Handler.WithAttrs(attrs)}
}

// WithGroup ensures sub-loggers grouped by a key retain the custom context-handling logic.
func (h *Handler) WithGroup(name string) slog.Handler {
	return &Handler{Handler: h.Handler.WithGroup(name)}
}
