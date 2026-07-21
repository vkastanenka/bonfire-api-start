package outbox

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type Status uint8

const (
	StatusUnknown    Status = 0
	StatusPending    Status = 1
	StatusProcessing Status = 2
	StatusProcessed  Status = 3
	StatusDeadLetter Status = 4
)

var statusNames = [...]string{
	StatusUnknown:    "UNKNOWN",
	StatusPending:    "PENDING",
	StatusProcessing: "PROCESSING",
	StatusProcessed:  "PROCESSED",
	StatusDeadLetter: "DEAD_LETTER",
}

func (s Status) Valid() bool {
	return int(s) > int(StatusUnknown) && int(s) < len(statusNames)
}

func (s Status) String() string {
	if int(s) >= 0 && int(s) < len(statusNames) {
		return statusNames[s]
	}
	return fmt.Sprintf("STATUS_%d", s)
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
