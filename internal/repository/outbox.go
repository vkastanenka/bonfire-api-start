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

// Create persists a single outbox domain event.
func (r *OutboxRepository) Create(ctx context.Context, e *outbox.Event) error {
	err := r.store.OutboxEventCreate(ctx, db.OutboxEventCreateParams{
		ID:            db.ToUUID(e.ID().UUID()),
		AggregateID:   db.ToUUIDPtr(e.AggregateID().UUIDPtr()),
		AggregateType: db.ToTextPtr(e.AggregateType().StringPtr()),
		EventType:     e.EventType().String(),
		Payload:       e.Payload().Raw(),
		TraceID:       db.ToTextPtr(e.TraceID().StringPtr()),
		CreatedAt:     db.ToTimestamptz(e.CreatedAt().Time()),
		UpdatedAt:     db.ToTimestamptz(e.UpdatedAt().Time()),
		NextAttemptAt: db.ToTimestamptz(e.NextAttemptAt().Time()),
		Attempts:      int32(e.Attempts()),
		MaxAttempts:   int32(e.MaxAttempts()),
	})
	if err != nil {
		return r.store.Err(err)
	}

	return nil
}

// CreateBatch bulk-inserts a set of domain events using pgx CopyFrom.
func (r *OutboxRepository) CreateBatch(ctx context.Context, events []*outbox.Event) error {
	params := make([]db.OutboxEventCreateBatchParams, 0, len(events))
	for _, e := range events {
		params = append(params, db.OutboxEventCreateBatchParams{
			ID:            db.ToUUID(e.ID().UUID()),
			AggregateID:   db.ToUUIDPtr(e.AggregateID().UUIDPtr()),
			AggregateType: db.ToTextPtr(e.AggregateType().StringPtr()),
			EventType:     e.EventType().String(),
			Payload:       e.Payload().Raw(),
			TraceID:       db.ToTextPtr(e.TraceID().StringPtr()),
			CreatedAt:     db.ToTimestamptz(e.CreatedAt().Time()),
			UpdatedAt:     db.ToTimestamptz(e.UpdatedAt().Time()),
			NextAttemptAt: db.ToTimestamptz(e.NextAttemptAt().Time()),
			Attempts:      int32(e.Attempts()),
			MaxAttempts:   int32(e.MaxAttempts()),
		})
	}

	_, err := r.store.OutboxEventCreateBatch(ctx, params)
	if err != nil {
		return r.store.Err(err)
	}

	return nil
}

// ClaimPending acquires and locks available pending events for processing.
func (r *OutboxRepository) ClaimPending(
	ctx context.Context,
	workerID fields.ID,
	leaseExpiresAt, now fields.Timestamp,
	limitVal int,
) ([]*outbox.Event, error) {
	rows, err := r.store.OutboxEventClaimPending(ctx, db.OutboxEventClaimPendingParams{
		Now:            db.ToTimestamptz(now.Time()),
		LimitVal:       int32(limitVal),
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

// MarkProcessed updates the event status as completed and releases worker locks.
func (r *OutboxRepository) MarkProcessed(ctx context.Context, e *outbox.Event, workerID fields.ID) error {
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

// MarkFailure updates attempt count, backoff schedule, and failure reason.
func (r *OutboxRepository) MarkFailure(ctx context.Context, e *outbox.Event, workerID fields.ID) error {
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

// MarkDeadLetter transitions an event to max attempts and records the error.
func (r *OutboxRepository) MarkDeadLetter(ctx context.Context, e *outbox.Event, workerID fields.ID) error {
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

// RenewLease extends the worker lease reservation time on an in-flight event.
func (r *OutboxRepository) RenewLease(ctx context.Context, e *outbox.Event, workerID fields.ID) error {
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

// ReleaseLease removes the active worker lock without changing attempt counts.
func (r *OutboxRepository) ReleaseLease(ctx context.Context, e *outbox.Event, workerID fields.ID) error {
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

// DeleteProcessedBatch deletes processed events prior to the target retention timestamp.
func (r *OutboxRepository) DeleteProcessedBatch(ctx context.Context, before fields.Timestamp, limitVal int) (int64, error) {
	rowsAffected, err := r.store.OutboxEventDeleteProcessedBatch(ctx, db.OutboxEventDeleteProcessedBatchParams{
		Before:   db.ToTimestamptz(before.Time()),
		LimitVal: int32(limitVal),
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

	var aggregateID fields.ID
	if row.AggregateID.Valid {
		aggUUID := db.FromUUID[uuid.UUID](row.AggregateID)
		aggregateID, err = fields.ParseRequiredID("aggregate_id", aggUUID)
		if err != nil {
			return nil, mapErr("failed to parse aggregate_id from database", "aggregate_id", aggUUID.String(), err)
		}
	}

	var aggregateType outbox.AggregateType
	if aggTypePtr := db.FromTextPtr[string](row.AggregateType); aggTypePtr != nil && *aggTypePtr != "" {
		aggregateType, err = outbox.ParseAggregateType(*aggTypePtr)
		if err != nil {
			return nil, mapErr("failed to parse aggregate_type from database", "aggregate_type", *aggTypePtr, err)
		}
	}

	eventType, err := outbox.ParseEventType(row.EventType)
	if err != nil {
		return nil, mapErr("failed to parse outbox event type from database", "event_type", row.EventType, err)
	}

	payload, err := outbox.ParsePayload(row.Payload)
	if err != nil {
		return nil, mapErr("failed to parse payload from database", "payload", string(row.Payload), err)
	}

	var traceID fields.TraceID
	if traceIDPtr := db.FromTextPtr[string](row.TraceID); traceIDPtr != nil && *traceIDPtr != "" {
		traceID, err = fields.ParseTraceID(*traceIDPtr)
		if err != nil {
			return nil, mapErr("failed to parse trace_id from database", "trace_id", *traceIDPtr, err)
		}
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
		lastError, err = outbox.ParseLastError(*lastErrPtr)
		if err != nil {
			return nil, mapErr("failed to parse last_error from database", "last_error", *lastErrPtr, err)
		}
	}

	processedAt := fields.NewTimestamp(db.FromTimestamptz(row.ProcessedAt))
	nextAttemptAt := fields.NewTimestamp(db.FromTimestamptz(row.NextAttemptAt))
	leaseExpiresAt := fields.NewTimestamp(db.FromTimestamptz(row.LeaseExpiresAt))
	createdAt := fields.NewTimestamp(db.FromTimestamptz(row.CreatedAt))
	updatedAt := fields.NewTimestamp(db.FromTimestamptz(row.UpdatedAt))

	return outbox.ReconstituteEvent(
		id,
		aggregateID,
		aggregateType,
		eventType,
		payload,
		traceID,
		processedAt,
		int(row.Attempts),
		int(row.MaxAttempts),
		nextAttemptAt,
		lockedBy,
		leaseExpiresAt,
		lastError,
		createdAt,
		updatedAt,
	), nil
}
