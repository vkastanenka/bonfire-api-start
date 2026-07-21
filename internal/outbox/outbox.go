package outbox

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Status uint8

const (
	StatusUnknown Status = iota
	StatusPending
	StatusProcessing
	StatusProcessed
	StatusDeadLetter
	statusMax
)

func (s Status) Valid() bool {
	return s > StatusUnknown && s < statusMax
}

func (s Status) String() string {
	switch s {
	case StatusPending:
		return "pending"
	case StatusProcessing:
		return "processing"
	case StatusProcessed:
		return "processed"
	case StatusDeadLetter:
		return "dead_letter"
	default:
		return "unknown"
	}
}

func (s Status) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.String())
}

func ParseStatus(s string) Status {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "pending":
		return StatusPending
	case "processing":
		return StatusProcessing
	case "processed":
		return StatusProcessed
	case "dead_letter":
		return StatusDeadLetter
	default:
		return StatusUnknown
	}
}

type Event struct {
	ID             uuid.UUID
	EventType      string
	Payload        json.RawMessage
	ProcessedAt    *time.Time
	Attempts       int32
	MaxAttempts    int32
	NextAttemptAt  time.Time
	LockedBy       *uuid.UUID
	LeaseExpiresAt *time.Time
	LastError      *string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func (e *Event) IsProcessed() bool  { return e.ProcessedAt != nil }
func (e *Event) IsDeadLetter() bool { return e.Attempts >= e.MaxAttempts }

func (e *Event) GetStatus() Status {
	if e.IsProcessed() {
		return StatusProcessed
	}
	if e.IsDeadLetter() {
		return StatusDeadLetter
	}
	if e.LeaseExpiresAt != nil && e.LeaseExpiresAt.After(time.Now()) {
		return StatusProcessing
	}
	return StatusPending
}
