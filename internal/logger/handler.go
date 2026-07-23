package logger

import (
	"bonfire-api/internal/httpio"
	"context"
	"log/slog"
)

type Handler struct {
	slog.Handler
}

func NewHandler(handler slog.Handler) *Handler {
	return &Handler{Handler: handler}
}

func (h *Handler) Handle(ctx context.Context, r slog.Record) error {
	if reqID, ok := ctx.Value(httpio.CtxReqIDKey).(string); ok && reqID != "" {
		r.AddAttrs(slog.String(string(httpio.CtxReqIDKey), reqID))
	}

	if traceID, ok := ctx.Value(httpio.CtxTraceIDKey).(string); ok && traceID != "" {
		r.AddAttrs(slog.String(string(httpio.CtxTraceIDKey), traceID))
	}

	return h.Handler.Handle(ctx, r)
}

func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &Handler{Handler: h.Handler.WithAttrs(attrs)}
}

func (h *Handler) WithGroup(name string) slog.Handler {
	return &Handler{Handler: h.Handler.WithGroup(name)}
}
