package outbox

import (
	"encoding/json"
	"strings"
	"time"

	"bonfire-api/internal/pkg/ptr"
	"bonfire-api/internal/repository"

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

func FromRepository(row repository.OutboxEvent) Event {
	e := Event{
		ID:            uuid.UUID(row.ID.Bytes),
		EventType:     row.EventType,
		Payload:       json.RawMessage(row.Payload),
		Attempts:      row.Attempts,
		MaxAttempts:   row.MaxAttempts,
		NextAttemptAt: row.NextAttemptAt.Time,
		CreatedAt:     row.CreatedAt.Time,
		UpdatedAt:     row.UpdatedAt.Time,
	}

	if row.ProcessedAt.Valid {
		e.ProcessedAt = ptr.To(row.ProcessedAt.Time)
	}

	if row.LockedBy.Valid {
		e.LockedBy = ptr.To(uuid.UUID(row.LockedBy.Bytes))
	}

	if row.LeaseExpiresAt.Valid {
		e.LeaseExpiresAt = ptr.To(row.LeaseExpiresAt.Time)
	}

	if row.LastError.Valid {
		e.LastError = ptr.To(row.LastError.String)
	}

	return e
}

type AuthView struct {
	ID             uuid.UUID       `json:"id"`
	EventType      string          `json:"event_type"`
	Payload        json.RawMessage `json:"payload"`
	Status         Status          `json:"status"`
	Attempts       int32           `json:"attempts"`
	MaxAttempts    int32           `json:"max_attempts"`
	NextAttemptAt  time.Time       `json:"next_attempt_at"`
	LockedBy       *uuid.UUID      `json:"locked_by,omitempty"`
	LeaseExpiresAt *time.Time      `json:"lease_expires_at,omitempty"`
	LastError      *string         `json:"last_error,omitempty"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

func ToAuthView(e Event) AuthView {
	return AuthView{
		ID:             e.ID,
		EventType:      e.EventType,
		Payload:        e.Payload,
		Status:         e.GetStatus(),
		Attempts:       e.Attempts,
		MaxAttempts:    e.MaxAttempts,
		NextAttemptAt:  e.NextAttemptAt,
		LockedBy:       ptr.Map(e.LockedBy),
		LeaseExpiresAt: ptr.Map(e.LeaseExpiresAt),
		LastError:      ptr.Map(e.LastError),
		CreatedAt:      e.CreatedAt,
		UpdatedAt:      e.UpdatedAt,
	}
}
