package repository

import (
	"context"
	"time"

	"bonfire-api/internal/db"
	"bonfire-api/internal/outbox"
	"bonfire-api/internal/pkg/errs"

	"github.com/google/uuid"
)

type OutboxRepository struct {
	store *db.Store
}

func NewOutboxRepository(store *db.Store) *OutboxRepository {
	return &OutboxRepository{
		store: store,
	}
}

// Create persists a single outbox domain event.
func (r *OutboxRepository) Create(ctx context.Context, e *outbox.Event) error {
	err := r.store.OutboxEventCreate(ctx, db.OutboxEventCreateParams{
		ID:            db.ToUUID(e.ID),
		Type:          e.Type,
		Payload:       e.Payload,
		TraceID:       db.ToTextPtr(e.TraceID),
		NextAttemptAt: db.ToTimestamptz(e.NextAttemptAt),
		Attempts:      int32(e.Attempts),
		MaxAttempts:   int32(e.MaxAttempts),
		CreatedAt:     db.ToTimestamptz(e.CreatedAt),
		UpdatedAt:     db.ToTimestamptz(e.UpdatedAt),
	})
	if err != nil {
		return db.NewError(err, db.EntityOutboxEvent)
	}

	return nil
}

// CreateBatch bulk-inserts a set of domain events using pgx CopyFrom.
func (r *OutboxRepository) CreateBatch(ctx context.Context, events []*outbox.Event) error {
	params := make([]db.OutboxEventCreateBatchParams, 0, len(events))
	for _, e := range events {
		params = append(params, db.OutboxEventCreateBatchParams{
			ID:            db.ToUUID(e.ID),
			Type:          e.Type,
			Payload:       e.Payload,
			TraceID:       db.ToTextPtr(e.TraceID),
			CreatedAt:     db.ToTimestamptz(e.CreatedAt),
			UpdatedAt:     db.ToTimestamptz(e.UpdatedAt),
			NextAttemptAt: db.ToTimestamptz(e.NextAttemptAt),
			Attempts:      int32(e.Attempts),
			MaxAttempts:   int32(e.MaxAttempts),
		})
	}

	_, err := r.store.OutboxEventCreateBatch(ctx, params)
	if err != nil {
		return db.NewError(err, db.EntityOutboxEvent)
	}

	return nil
}

// ClaimPending acquires and locks available pending events for processing.
func (r *OutboxRepository) ClaimPending(
	ctx context.Context,
	workerID uuid.UUID,
	leaseExpiresAt, now time.Time,
	limitVal int,
) ([]*outbox.Event, error) {
	rows, err := r.store.OutboxEventClaimPending(ctx, db.OutboxEventClaimPendingParams{
		Now:            db.ToTimestamptz(now),
		LimitVal:       int32(limitVal),
		WorkerID:       db.ToUUID(workerID),
		LeaseExpiresAt: db.ToTimestamptz(leaseExpiresAt),
	})
	if err != nil {
		return nil, db.NewError(err, db.EntityOutboxEvent)
	}

	events := make([]*outbox.Event, 0, len(rows))
	for _, row := range rows {
		events = append(events, outboxFromRow(row))
	}

	return events, nil
}

// MarkProcessed updates the event status as completed and releases worker locks.
func (r *OutboxRepository) MarkProcessed(ctx context.Context, e *outbox.Event, workerID uuid.UUID) error {
	err := r.store.OutboxEventMarkProcessed(ctx, db.OutboxEventMarkProcessedParams{
		ProcessedAt: db.ToTimestamptzPtr(e.ProcessedAt),
		UpdatedAt:   db.ToTimestamptz(e.UpdatedAt),
		ID:          db.ToUUID(e.ID),
		WorkerID:    db.ToUUID(workerID),
	})
	if err != nil {
		return db.NewError(err, db.EntityOutboxEvent)
	}

	return nil
}

// MarkFailure updates attempt count, backoff schedule, and failure reason.
func (r *OutboxRepository) MarkFailure(ctx context.Context, e *outbox.Event, workerID uuid.UUID) error {
	err := r.store.OutboxEventMarkFailure(ctx, db.OutboxEventMarkFailureParams{
		NextAttemptAt: db.ToTimestamptz(e.NextAttemptAt),
		LastError:     db.ToTextPtr(e.LastError),
		UpdatedAt:     db.ToTimestamptz(e.UpdatedAt),
		ID:            db.ToUUID(e.ID),
		WorkerID:      db.ToUUID(workerID),
	})
	if err != nil {
		return db.NewError(err, db.EntityOutboxEvent)
	}

	return nil
}

// MarkDeadLetter transitions an event to max attempts and records the error.
func (r *OutboxRepository) MarkDeadLetter(ctx context.Context, e *outbox.Event, workerID uuid.UUID) error {
	err := r.store.OutboxEventMarkDeadLetter(ctx, db.OutboxEventMarkDeadLetterParams{
		LastError: db.ToTextPtr(e.LastError),
		UpdatedAt: db.ToTimestamptz(e.UpdatedAt),
		ID:        db.ToUUID(e.ID),
		WorkerID:  db.ToUUID(workerID),
	})
	if err != nil {
		return db.NewError(err, db.EntityOutboxEvent)
	}

	return nil
}

// RenewLease extends the worker lease reservation time on an in-flight event.
// Returns an error if the update fails or if 0 rows were affected (lock expired or reclaimed).
func (r *OutboxRepository) RenewLease(
	ctx context.Context,
	eventID uuid.UUID,
	workerID uuid.UUID,
	leaseExpiresAt time.Time,
	now time.Time,
) error {
	rowsAffected, err := r.store.OutboxEventRenewLease(ctx, db.OutboxEventRenewLeaseParams{
		LeaseExpiresAt: db.ToTimestamptz(leaseExpiresAt),
		Now:            db.ToTimestamptz(now),
		ID:             db.ToUUID(eventID),
		WorkerID:       db.ToUUID(workerID),
	})
	if err != nil {
		return db.NewError(err, db.EntityOutboxEvent)
	}

	if rowsAffected == 0 {
		return errs.NotFound("Outbox event lease renewal failed; lock lost or event reclaimed.")
	}

	return nil
}

// ReleaseLease removes the active worker lock without changing attempt counts.
func (r *OutboxRepository) ReleaseLease(ctx context.Context, e *outbox.Event, workerID uuid.UUID) error {
	err := r.store.OutboxEventReleaseLease(ctx, db.OutboxEventReleaseLeaseParams{
		UpdatedAt: db.ToTimestamptz(e.UpdatedAt),
		ID:        db.ToUUID(e.ID),
		WorkerID:  db.ToUUID(workerID),
	})
	if err != nil {
		return db.NewError(err, db.EntityOutboxEvent)
	}

	return nil
}

// DeleteProcessedBatch deletes processed events prior to the target retention timestamp.
func (r *OutboxRepository) DeleteProcessedBatch(ctx context.Context, before time.Time, limitVal int) (int64, error) {
	rowsAffected, err := r.store.OutboxEventDeleteProcessedBatch(ctx, db.OutboxEventDeleteProcessedBatchParams{
		Before:   db.ToTimestamptz(before),
		LimitVal: int32(limitVal),
	})
	if err != nil {
		return 0, db.NewError(err, db.EntityOutboxEvent)
	}

	return rowsAffected, nil
}

func outboxFromRow(row db.OutboxEvent) *outbox.Event {
	return outbox.ReconstituteEvent(
		db.FromUUID(row.ID),
		row.Type,
		row.Payload,
		db.FromTextPtr(row.TraceID),
		db.FromTimestamptzPtr(row.ProcessedAt),
		int(row.Attempts),
		int(row.MaxAttempts),
		db.FromTimestamptz(row.NextAttemptAt),
		db.FromTextPtr(row.LastError),
		db.FromUUIDPtr(row.LockedBy),
		db.FromTimestamptzPtr(row.LeaseExpiresAt),
		db.FromTimestamptz(row.CreatedAt),
		db.FromTimestamptz(row.UpdatedAt),
	)
}
