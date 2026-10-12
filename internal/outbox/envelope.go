package outbox

import (
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
