package outbox

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type Repository interface {
	ClaimPending(ctx context.Context, workerID uuid.UUID, leaseExpiresAt time.Time, now time.Time, limitVal int) ([]*Event, error)
	Create(ctx context.Context, e *Event) error
	CreateBatch(ctx context.Context, events []*Event) error
	DeleteProcessedBatch(ctx context.Context, before time.Time, limitVal int) (int64, error)
	MarkDeadLetter(ctx context.Context, e *Event, workerID uuid.UUID) error
	MarkFailure(ctx context.Context, e *Event, workerID uuid.UUID) error
	MarkProcessed(ctx context.Context, e *Event, workerID uuid.UUID) error
	Publish(ctx context.Context, eventType string, payload any, now time.Time) error
	ReleaseLease(ctx context.Context, e *Event, workerID uuid.UUID) error
	RenewLease(ctx context.Context, e *Event, workerID uuid.UUID) error
}
