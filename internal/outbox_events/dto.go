package outbox_events

import (
	"bonfire-api/internal/repository"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// ==========================================
// REQUEST / PARAMS DTOs
// ==========================================

// Create (TODO: Create struct for EventType)
type CreateReq struct {
	EventType string          `json:"event_type" validate:"required,max=100"`
	Payload   json.RawMessage `json:"payload" validate:"required"`
}

type CreateParams struct {
	EventType string          `json:"event_type" validate:"required,max=100"`
	Payload   json.RawMessage `json:"payload" validate:"required"`
}

// TODO: Refactor into its own file
type ListParams struct {
	Limit  int32      `json:"limit"`
	Cursor *uuid.UUID `json:"cursor"` // Replaced Offset with Cursor for keyset pagination
}

type RecordFailureParams struct {
	ID    uuid.UUID
	Error string
}

// ==========================================
// RESPONSE / VIEW DTOs
// ==========================================

type View struct {
	ID            uuid.UUID       `json:"id"`
	EventType     string          `json:"event_type"`
	Payload       json.RawMessage `json:"payload"`
	Status        string          `json:"status"`
	Attempts      int32           `json:"attempts"`
	MaxAttempts   int32           `json:"max_attempts"`
	LastError     *string         `json:"last_error,omitempty"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
	ProcessedAt   *time.Time      `json:"processed_at,omitempty"`
	NextAttemptAt time.Time       `json:"next_attempt_at"`
}

// ==========================================
// MAPPERS
// ==========================================

func NewView(row repository.OutboxEvent) View {
	status := "pending"
	var processedAt *time.Time
	if row.ProcessedAt.Valid {
		status = "processed"
		t := row.ProcessedAt.Time
		processedAt = &t
	}

	var lastError *string
	if row.LastError.Valid {
		lastError = &row.LastError.String
	}

	return View{
		ID:            uuid.UUID(row.ID.Bytes),
		EventType:     row.EventType,
		Payload:       row.Payload,
		Status:        status,
		Attempts:      row.Attempts,
		MaxAttempts:   row.MaxAttempts,
		LastError:     lastError,
		CreatedAt:     row.CreatedAt.Time,
		UpdatedAt:     row.UpdatedAt.Time,
		ProcessedAt:   processedAt,
		NextAttemptAt: row.NextAttemptAt.Time,
	}
}

type CreateResult struct {
	View View `json:"view"`
}

type CreateRes struct {
	OutboxEvent View `json:"outbox_event"`
}

type GetByIDReq struct {
	ID uuid.UUID `json:"id"`
}

type GetByIDParams struct {
	ID uuid.UUID `json:"id"`
}

type GetByIDRes struct {
	Data uuid.UUID `json:"id"`
}
