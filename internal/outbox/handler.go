package outbox

import (
	"bonfire-api/internal/httpio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
)

var (
	ErrFatal          = errors.New("outbox: fatal event execution error")
	ErrLeaseLost      = errors.New("outbox: lease renewal failed")
	ErrInvalidPayload = errors.New("outbox: invalid event payload schema")
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
			return fmt.Errorf("%w: failed to unmarshal outbox envelope: %w", ErrInvalidPayload, err)
		}

		// Inject TraceID from envelope metadata into execution context if present
		if env.Metadata.TraceID != "" {
			ctx = context.WithValue(ctx, httpio.CtxKeyTraceID, env.Metadata.TraceID)
		}

		return fn(ctx, env.Metadata, env.Payload)
	}
}
