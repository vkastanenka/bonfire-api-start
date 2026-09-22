package outbox

import (
	"github.com/google/uuid"
)

// Metadata contains contextual information attached to outbox events.
type Metadata struct {
	ActorID      uuid.UUID   `json:"actor_id,omitempty"`
	SessionID    uuid.UUID   `json:"session_id,omitempty"`
	TraceID      string      `json:"trace_id,omitempty"`
	RecipientIDs []uuid.UUID `json:"recipient_ids,omitempty"`
}

// Envelope wraps event metadata and payload for database storage.
type Envelope[T any] struct {
	Metadata Metadata `json:"metadata"`
	Payload  T        `json:"payload"`
}
