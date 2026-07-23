package outbox

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
)

type PublishParams struct {
	Variant string
	Payload any
}

type ListParams struct {
	Cursor *uuid.UUID
	Limit  int32
}

type AcquireBatchParams struct {
	Limit                  int32
	WorkerID               uuid.UUID
	LeaseDurationInSeconds int32
}

type RecordFailureParams struct {
	ID        uuid.UUID
	LastError string
}

type MarkDeadLetterParams struct {
	ID     uuid.UUID
	Reason string
}

type Handler func(ctx context.Context, payload json.RawMessage) error

type Repository interface {
	Publish(ctx context.Context, p PublishParams) (Event, error)
	Get(ctx context.Context, id uuid.UUID) (Event, error)
	List(ctx context.Context, p ListParams) ([]Event, error)
	AcquireBatch(ctx context.Context, p AcquireBatchParams) ([]Event, error)
	MarkProcessed(ctx context.Context, id uuid.UUID) (Event, error)
	RecordFailure(ctx context.Context, p RecordFailureParams) (Event, error)
	MarkDeadLetter(ctx context.Context, p MarkDeadLetterParams) (Event, error)
	ResetAttempts(ctx context.Context, id uuid.UUID) (Event, error)
	Delete(ctx context.Context, id uuid.UUID) error
	PurgeProcessed(ctx context.Context) error
}
