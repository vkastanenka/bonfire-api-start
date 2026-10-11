package outbox

import (
	"bonfire-api/internal/appctx"
	"context"
	"encoding/json"
)

// Handler processes raw outbox payloads.
type Handler func(ctx context.Context, rawPayload json.RawMessage) error

// TypedHandler processes strongly typed outbox payloads and metadata.
type TypedHandler[T any] func(ctx context.Context, meta Metadata, payload T) error

// BindHandler adapts a TypedHandler into a standard Handler by unmarshaling the raw envelope.
func BindHandler[T any](fn TypedHandler[T]) Handler {
	return func(ctx context.Context, rawPayload json.RawMessage) error {
		var env Envelope[T]
		if err := json.Unmarshal(rawPayload, &env); err != nil {
			return ErrInvalidPayload
		}

		if env.Metadata.TraceID != "" {
			ctx = appctx.WithTraceID(ctx, env.Metadata.TraceID)
		}

		return fn(ctx, env.Metadata, env.Payload)
	}
}
