package logger

import (
	"context"
	"log/slog"
)

type ctxAttrsKey struct{}

// WithContextAttrs safely appends slog attributes to the context slice.
func WithContextAttrs(ctx context.Context, attrs ...slog.Attr) context.Context {
	if len(attrs) == 0 {
		return ctx
	}

	existing, _ := ctx.Value(ctxAttrsKey{}).([]slog.Attr)

	combined := make([]slog.Attr, len(existing)+len(attrs))
	copy(combined, existing)
	copy(combined[len(existing):], attrs)

	return context.WithValue(ctx, ctxAttrsKey{}, combined)
}
