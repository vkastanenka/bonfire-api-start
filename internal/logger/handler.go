package logger

import (
	"context"
	"log/slog"
)

type CtxKey string

const (
	ReqIDKey   CtxKey = "request_id"
	TraceIDKey CtxKey = "trace_id"
)

type Handler struct {
	slog.Handler
}

func NewHandler(handler slog.Handler) *Handler {
	return &Handler{Handler: handler}
}

func (h *Handler) Handle(ctx context.Context, r slog.Record) error {
	if reqID, ok := ctx.Value(ReqIDKey).(string); ok && reqID != "" {
		r.AddAttrs(slog.String("request_id", reqID))
	}

	if traceID, ok := ctx.Value(TraceIDKey).(string); ok && traceID != "" {
		r.AddAttrs(slog.String("trace_id", traceID))
	}

	return h.Handler.Handle(ctx, r)
}

func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &Handler{Handler: h.Handler.WithAttrs(attrs)}
}

func (h *Handler) WithGroup(name string) slog.Handler {
	return &Handler{Handler: h.Handler.WithGroup(name)}
}
