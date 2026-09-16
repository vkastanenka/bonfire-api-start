package logger

import (
	"context"
	"log/slog"
)

const (
	ctxReqIDKey   = "requestId"
	ctxTraceIDKey = "traceId"
)

// Handler wraps an slog.Handler to inject contextual metadata into log records.
type Handler struct {
	slog.Handler
}

// NewHandler constructs a Handler wrapping the provided slog.Handler.
func NewHandler(handler slog.Handler) *Handler {
	return &Handler{Handler: handler}
}

// Handle extracts request and trace IDs from ctx and appends them as attributes before logging.
func (h *Handler) Handle(ctx context.Context, r slog.Record) error {
	if reqID, ok := ctx.Value(ctxReqIDKey).(string); ok && reqID != "" {
		r.AddAttrs(slog.String(ctxReqIDKey, reqID))
	}

	if traceID, ok := ctx.Value(ctxTraceIDKey).(string); ok && traceID != "" {
		r.AddAttrs(slog.String(ctxTraceIDKey, traceID))
	}

	return h.Handler.Handle(ctx, r)
}

// WithAttrs returns a new Handler with pre-populated attributes.
func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &Handler{Handler: h.Handler.WithAttrs(attrs)}
}

// WithGroup returns a new Handler with the given group name applied.
func (h *Handler) WithGroup(name string) slog.Handler {
	return &Handler{Handler: h.Handler.WithGroup(name)}
}
