package outbox

import (
	"context"
	"encoding/json"
	"fmt"
)

// Handler signature stays simple and non-generic
type Handler func(ctx context.Context, payload json.RawMessage) error

// TypedHandler provides type safety for the concrete handler logic
type TypedHandler[T any] func(ctx context.Context, payload T) error

// BindHandler adaptively converts a TypedHandler into a standard outbox.Handler.
func BindHandler[T any](fn TypedHandler[T]) Handler {
	return func(ctx context.Context, payload json.RawMessage) error {
		var p T
		if err := json.Unmarshal(payload, &p); err != nil {
			return fmt.Errorf("%w: %v", errInvalidPayload, err)
		}
		return fn(ctx, p)
	}
}
