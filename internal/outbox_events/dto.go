package outbox_events

import (
	"bonfire-api/internal/repository"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Create (TODO: Create struct for EventType)
type CreateReq struct {
	EventType string          `json:"event_type" validate:"required,max=100"`
	Payload   json.RawMessage `json:"payload" validate:"required"`
}

type CreateParams struct {
	EventType string          `json:"event_type" validate:"required,max=100"`
	Payload   json.RawMessage `json:"payload" validate:"required"`
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

type ListParams struct {
	Limit  int32 `json:"limit"`
	Offset int32 `json:"offset"`
}

type RecordFailureParams struct {
	ID    uuid.UUID
	Error string
}

type View struct {
	ID        uuid.UUID       `json:"id"`
	EventType string          `json:"event_type"`
	Payload   json.RawMessage `json:"payload"`
	Status    string          `json:"status"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

func NewView(row repository.OutboxEvent) View {
	status := "pending"
	if row.ProcessedAt.Valid {
		status = "processed"
	}

	return View{
		ID:        uuid.UUID(row.ID.Bytes),
		EventType: row.EventType,
		Payload:   row.Payload,
		Status:    status,
		CreatedAt: row.CreatedAt.Time,
		UpdatedAt: row.UpdatedAt.Time,
	}
}
