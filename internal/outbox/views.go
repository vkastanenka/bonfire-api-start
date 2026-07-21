package outbox

import (
	"bonfire-api/internal/pkg/ptr"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

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
