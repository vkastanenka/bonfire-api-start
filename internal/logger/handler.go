package logger

import (
	"context"
	"log/slog"
)

// Define local context keys or string-based lookups to avoid tight coupling.
type CtxKey string

const (
	ReqIDKey   CtxKey = "request_id"
	TraceIDKey CtxKey = "trace_id"
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
	if reqID, ok := ctx.Value(ReqIDKey).(string); ok && reqID != "" {
		r.AddAttrs(slog.String("request_id", reqID))
	}

	if traceID, ok := ctx.Value(TraceIDKey).(string); ok && traceID != "" {
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
