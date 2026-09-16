package logger

import (
	"context"
	"log/slog"
)

// Handler wraps an slog.Handler to inject contextual metadata into log records.
type Handler struct {
	slog.Handler
}

// NewHandler constructs a Handler wrapping the provided slog.Handler.
func NewHandler(handler slog.Handler) *Handler {
	return &Handler{Handler: handler}
}

// Handle extracts attributes stored in ctx and appends them to the record before logging.
func (h *Handler) Handle(ctx context.Context, r slog.Record) error {
	if attrs, ok := ctx.Value(ctxAttrsKey{}).([]slog.Attr); ok && len(attrs) > 0 {
		r.AddAttrs(attrs...)
	}
	return h.Handler.Handle(ctx, r)
}

// WithAttrs returns a new Handler wrapping the underlying handler's WithAttrs output.
func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &Handler{Handler: h.Handler.WithAttrs(attrs)}
}

// WithGroup returns a new Handler wrapping the underlying handler's WithGroup output.
func (h *Handler) WithGroup(name string) slog.Handler {
	return &Handler{Handler: h.Handler.WithGroup(name)}
}
