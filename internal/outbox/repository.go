package outbox

import (
	"bonfire-api/internal/fields"
	"context"
)

// Repository abstracts the Outbox persistence operations required by the Worker.
type Repository interface {
	ClaimPending(ctx context.Context, workerID fields.ID, leaseExpiresAt fields.Timestamp, now fields.Timestamp, limitVal int) ([]*Event, error)
	Create(ctx context.Context, e *Event) error
	CreateBatch(ctx context.Context, events []*Event) error
	DeleteProcessedBatch(ctx context.Context, before fields.Timestamp, limitVal int) (int64, error)
	MarkDeadLetter(ctx context.Context, e *Event, workerID fields.ID) error
	MarkFailure(ctx context.Context, e *Event, workerID fields.ID) error
	MarkProcessed(ctx context.Context, e *Event, workerID fields.ID) error
	Publish(ctx context.Context, eventType Type, payload Payload, now fields.Timestamp) error
	ReleaseLease(ctx context.Context, e *Event, workerID fields.ID) error
	RenewLease(ctx context.Context, e *Event, workerID fields.ID) error
}
