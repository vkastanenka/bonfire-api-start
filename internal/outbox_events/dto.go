package outbox_events

import (
	"bonfire-api/internal/repository"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// ==========================================
// HANDLERS
// ==========================================

type PingRes struct {
	Status string `json:"status"`
}

type CountRes struct {
	Count int64 `json:"count"`
}

// ==========================================
// SERVICES
// ==========================================

type CreateParams struct {
	EventType string          `json:"event_type" validate:"required,max=100"`
	Payload   json.RawMessage `json:"payload" validate:"required"`
}

type ListParams struct {
	Limit  int32      `json:"limit"`
	Cursor *uuid.UUID `json:"cursor"`
}

type RecordFailureParams struct {
	ID    uuid.UUID
	Error string
}

type MarkDeadLetterParams struct {
	ID    uuid.UUID
	Error string
}

// ==========================================
// VIEW
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
