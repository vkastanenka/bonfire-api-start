package outbox

import (
	"bonfire-api/internal/appctx"
	"bonfire-api/internal/token"
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
)

// Metadata contains contextual information attached to outbox events.
type Metadata struct {
	ActorID   uuid.UUID `json:"actor_id,omitempty"`
	SessionID uuid.UUID `json:"session_id,omitempty"`
	TraceID   string    `json:"trace_id,omitempty"`
}

// Envelope wraps event metadata and payload for database storage.
type Envelope[T any] struct {
	Metadata Metadata `json:"metadata"`
	Payload  T        `json:"payload"`
}

// Handler processes raw outbox payloads.
type Handler func(ctx context.Context, rawPayload json.RawMessage) error

// TypedHandler processes strongly typed outbox payloads and metadata.
type TypedHandler[T any] func(ctx context.Context, meta Metadata, payload T) error

// BindHandler adapts a TypedHandler into a standard Handler by unmarshaling the raw envelope.
func BindHandler[T any](fn TypedHandler[T]) Handler {
	return func(ctx context.Context, rawPayload json.RawMessage) error {
		var env Envelope[T]
		if err := json.Unmarshal(rawPayload, &env); err != nil {
			return fmt.Errorf("outbox: failed to unmarshal envelope: %w", err)
		}
		return fn(ctx, env.Metadata, env.Payload)
	}
}

// GetMetadata extracts actor, session, and tracing context for outbox events.
func GetMetadata(ctx context.Context, claims *token.Claims) Metadata {
	meta := Metadata{
		TraceID: appctx.GetTraceID(ctx),
	}

	if claims != nil {
		meta.ActorID = claims.UserID
		meta.SessionID = claims.SessionID
	}

	return meta
}
