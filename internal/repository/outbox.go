package repository

import (
	"context"
	"fmt"

	"bonfire-api/internal/db"
	"bonfire-api/internal/errs"
	"bonfire-api/internal/fields"
	"bonfire-api/internal/outbox"

	"github.com/google/uuid"
)

type OutboxRepository struct {
	store *db.Store
}

func NewOutboxRepository(store *db.Store) *OutboxRepository {
	return &OutboxRepository{
		store: store.WithEntity(db.EntityOutboxEvent),
	}
}

// OutboxEventCreate persists a single outbox domain event.
func (r *OutboxRepository) OutboxEventCreate(ctx context.Context, e *outbox.Event) error {
	err := r.store.OutboxEventCreate(ctx, db.OutboxEventCreateParams{
		ID:            db.ToUUID(e.ID().UUID()),
		AggregateID:   db.ToUUIDPtr(e.ID().UUIDPtr()),
		AggregateType: db.ToTextPtr(nil),
		EventType:     e.EventType().String(),
		Payload:       e.Payload(),
		TraceID:       db.ToTextPtr(nil),
		CreatedAt:     db.ToTimestamptz(e.CreatedAt().Time()),
		UpdatedAt:     db.ToTimestamptz(e.UpdatedAt().Time()),
		NextAttemptAt: db.ToTimestamptz(e.NextAttemptAt().Time()),
		Attempts:      e.Attempts(),
		MaxAttempts:   e.MaxAttempts(),
	})
	if err != nil {
		return r.store.Err(err)
	}

	return nil
}

// OutboxEventCreateBatch bulk-inserts a set of domain events using pgx CopyFrom.
func (r *OutboxRepository) OutboxEventCreateBatch(ctx context.Context, events []*outbox.Event) error {
	if len(events) == 0 {
		return nil
	}

	params := make([]db.OutboxEventCreateBatchParams, 0, len(events))
	for _, e := range events {
		params = append(params, db.OutboxEventCreateBatchParams{
			ID:            db.ToUUID(e.ID().UUID()),
			AggregateID:   db.ToUUIDPtr(nil),
			AggregateType: db.ToTextPtr(nil),
			EventType:     e.EventType().String(),
			Payload:       e.Payload(),
			TraceID:       db.ToTextPtr(nil),
			CreatedAt:     db.ToTimestamptz(e.CreatedAt().Time()),
			UpdatedAt:     db.ToTimestamptz(e.UpdatedAt().Time()),
			NextAttemptAt: db.ToTimestamptz(e.NextAttemptAt().Time()),
			Attempts:      e.Attempts(),
			MaxAttempts:   e.MaxAttempts(),
		})
	}

	_, err := r.store.OutboxEventCreateBatch(ctx, params)
	if err != nil {
		return r.store.Err(err)
	}

	return nil
}

// OutboxEventClaimPending acquires and locks available pending events for processing.
func (r *OutboxRepository) OutboxEventClaimPending(
	ctx context.Context,
	workerID fields.ID,
	leaseExpiresAt, now fields.Timestamp,
	limitVal int32,
) ([]*outbox.Event, error) {
	rows, err := r.store.OutboxEventClaimPending(ctx, db.OutboxEventClaimPendingParams{
		Now:            db.ToTimestamptz(now.Time()),
		LimitVal:       limitVal,
		WorkerID:       db.ToUUID(workerID.UUID()),
		LeaseExpiresAt: db.ToTimestamptz(leaseExpiresAt.Time()),
	})
	if err != nil {
		return nil, r.store.Err(err)
	}

	events := make([]*outbox.Event, 0, len(rows))
	for _, row := range rows {
		evt, err := outboxFromRow(row)
		if err != nil {
			return nil, err
		}
		events = append(events, evt)
	}

	return events, nil
}

// OutboxEventMarkProcessed updates the event status as completed and releases worker locks.
func (r *OutboxRepository) OutboxEventMarkProcessed(ctx context.Context, e *outbox.Event, workerID fields.ID) error {
	err := r.store.OutboxEventMarkProcessed(ctx, db.OutboxEventMarkProcessedParams{
		ProcessedAt: db.ToTimestamptz(e.ProcessedAt().Time()),
		UpdatedAt:   db.ToTimestamptz(e.UpdatedAt().Time()),
		ID:          db.ToUUID(e.ID().UUID()),
		WorkerID:    db.ToUUID(workerID.UUID()),
	})
	if err != nil {
		return r.store.Err(err)
	}

	return nil
}

// OutboxEventMarkFailure updates attempt count, backoff schedule, and failure reason.
func (r *OutboxRepository) OutboxEventMarkFailure(ctx context.Context, e *outbox.Event, workerID fields.ID) error {
	err := r.store.OutboxEventMarkFailure(ctx, db.OutboxEventMarkFailureParams{
		NextAttemptAt: db.ToTimestamptz(e.NextAttemptAt().Time()),
		LastError:     db.ToTextPtr(e.LastError().StringPtr()),
		UpdatedAt:     db.ToTimestamptz(e.UpdatedAt().Time()),
		ID:            db.ToUUID(e.ID().UUID()),
		WorkerID:      db.ToUUID(workerID.UUID()),
	})
	if err != nil {
		return r.store.Err(err)
	}

	return nil
}

// OutboxEventMarkDeadLetter transitions an event to max attempts and records the error.
func (r *OutboxRepository) OutboxEventMarkDeadLetter(ctx context.Context, e *outbox.Event, workerID fields.ID) error {
	err := r.store.OutboxEventMarkDeadLetter(ctx, db.OutboxEventMarkDeadLetterParams{
		LastError: db.ToTextPtr(e.LastError().StringPtr()),
		UpdatedAt: db.ToTimestamptz(e.UpdatedAt().Time()),
		ID:        db.ToUUID(e.ID().UUID()),
		WorkerID:  db.ToUUID(workerID.UUID()),
	})
	if err != nil {
		return r.store.Err(err)
	}

	return nil
}

// OutboxEventRenewLease extends the worker lease reservation time on an in-flight event.
func (r *OutboxRepository) OutboxEventRenewLease(ctx context.Context, e *outbox.Event, workerID fields.ID) error {
	err := r.store.OutboxEventRenewLease(ctx, db.OutboxEventRenewLeaseParams{
		LeaseExpiresAt: db.ToTimestamptz(e.LeaseExpiresAt().Time()),
		UpdatedAt:      db.ToTimestamptz(e.UpdatedAt().Time()),
		ID:             db.ToUUID(e.ID().UUID()),
		WorkerID:       db.ToUUID(workerID.UUID()),
	})
	if err != nil {
		return r.store.Err(err)
	}

	return nil
}

// OutboxEventReleaseLease removes the active worker lock without changing attempt counts.
func (r *OutboxRepository) OutboxEventReleaseLease(ctx context.Context, e *outbox.Event, workerID fields.ID) error {
	err := r.store.OutboxEventReleaseLease(ctx, db.OutboxEventReleaseLeaseParams{
		UpdatedAt: db.ToTimestamptz(e.UpdatedAt().Time()),
		ID:        db.ToUUID(e.ID().UUID()),
		WorkerID:  db.ToUUID(workerID.UUID()),
	})
	if err != nil {
		return r.store.Err(err)
	}

	return nil
}

// OutboxEventDeleteProcessedBatch deletes processed events prior to the target retention timestamp.
func (r *OutboxRepository) OutboxEventDeleteProcessedBatch(ctx context.Context, before fields.Timestamp, limitVal int32) (int64, error) {
	rowsAffected, err := r.store.OutboxEventDeleteProcessedBatch(ctx, db.OutboxEventDeleteProcessedBatchParams{
		Before:   db.ToTimestamptz(before.Time()),
		LimitVal: limitVal,
	})
	if err != nil {
		return 0, r.store.Err(err)
	}

	return rowsAffected, nil
}

// ============================================================================
// Helpers
// ============================================================================

func outboxFromRow(row db.OutboxEvent) (*outbox.Event, error) {
	eventID := db.FromUUID[uuid.UUID](row.ID)
	eventIDStr := eventID.String()

	mapErr := func(msg, key string, val any, err error) *errs.Error {
		return errs.Internal(msg).
			Wrap(err).
			Reason("CORRUPT_DATABASE_RECORD").
			Meta(key, fmt.Sprintf("%v", val)).
			Resource("OutboxEvent", eventIDStr, "", "database row mapping")
	}

	id, err := fields.ParseRequiredID("id", eventID)
	if err != nil {
		return nil, mapErr("failed to parse outbox event id from database", "id", eventIDStr, err)
	}

	eventType, err := outbox.ParseEventType("event_type", row.EventType)
	if err != nil {
		return nil, mapErr("failed to parse outbox event type from database", "event_type", row.EventType, err)
	}

	var lockedBy fields.ID
	if row.LockedBy.Valid {
		lockedByUUID := db.FromUUID[uuid.UUID](row.LockedBy)
		lockedBy, err = fields.ParseRequiredID("locked_by", lockedByUUID)
		if err != nil {
			return nil, mapErr("failed to parse locked_by from database", "locked_by", lockedByUUID.String(), err)
		}
	}

	var lastError outbox.LastError
	if lastErrPtr := db.FromTextPtr[string](row.LastError); lastErrPtr != nil && *lastErrPtr != "" {
		lastError, err = outbox.ParseLastError("last_error", *lastErrPtr)
		if err != nil {
			return nil, mapErr("failed to parse last_error from database", "last_error", *lastErrPtr, err)
		}
	}

	processedAt := fields.NewTimestampFromTime(db.FromTimestamptz(row.ProcessedAt))
	nextAttemptAt := fields.NewTimestampFromTime(db.FromTimestamptz(row.NextAttemptAt))
	leaseExpiresAt := fields.NewTimestampFromTime(db.FromTimestamptz(row.LeaseExpiresAt))
	createdAt := fields.NewTimestampFromTime(db.FromTimestamptz(row.CreatedAt))
	updatedAt := fields.NewTimestampFromTime(db.FromTimestamptz(row.UpdatedAt))

	return outbox.NewEvent(
		id,
		eventType,
		row.Payload,
		processedAt,
		row.Attempts,
		row.MaxAttempts,
		nextAttemptAt,
		lockedBy,
		leaseExpiresAt,
		lastError,
		createdAt,
		updatedAt,
	), nil
}
