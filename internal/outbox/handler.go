package outbox

import (
	"bonfire-api/internal/appctx"
	"context"
	"encoding/json"

	"github.com/google/uuid"
)

// Metadata contains contextual information attached to outbox events.
type Metadata struct {
	TraceID      string      `json:"trace_id"`
	ActorID      *uuid.UUID  `json:"actor_id,omitempty"`
	SessionID    *uuid.UUID  `json:"session_id,omitempty"`
	RecipientIDs []uuid.UUID `json:"recipient_ids,omitempty"`
}

func NewMetadata(traceID string, actorID, sessionID *uuid.UUID, recipientIDs []uuid.UUID) Metadata {
	return Metadata{
		TraceID:      traceID,
		ActorID:      actorID,
		SessionID:    sessionID,
		RecipientIDs: recipientIDs,
	}
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
			return ErrInvalidPayload
		}

		if env.Metadata.TraceID != "" {
			ctx = appctx.WithTraceID(ctx, env.Metadata.TraceID)
		}

		return fn(ctx, env.Metadata, env.Payload)
	}
}
