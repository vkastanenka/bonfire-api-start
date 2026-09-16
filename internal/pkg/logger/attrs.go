package logger

import (
	"context"
	"log/slog"
)

type ctxAttrsKey struct{}

// WithContextAttrs appends slog attributes to the context slice.
func WithContextAttrs(ctx context.Context, attrs ...slog.Attr) context.Context {
	if len(attrs) == 0 {
		return ctx
	}
	existing, _ := ctx.Value(ctxAttrsKey{}).([]slog.Attr)
	return context.WithValue(ctx, ctxAttrsKey{}, append(existing, attrs...))
}
