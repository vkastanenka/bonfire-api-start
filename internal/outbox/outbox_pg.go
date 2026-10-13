package outbox

// import (
// 	"context"
// 	"time"

// 	"bonfire-api/internal/db"
// 	"bonfire-api/internal/outbox"

// 	"github.com/google/uuid"
// )

// type OutboxRepository struct {
// 	store *db.Store
// }

// func NewOutboxRepository(store *db.Store) *OutboxRepository {
// 	return &OutboxRepository{
// 		store: store,
// 	}
// }

// func (r *OutboxRepository) Create(ctx context.Context, e *outbox.Event) error {
// 	if e == nil {
// 		return ErrDomainPayloadRequired().
// 			Meta("resource", db.ResourceOutboxEvent.String())
// 	}

// 	err := r.store.OutboxEventCreate(ctx, db.OutboxEventCreateParams{
// 		ID:            db.ToUUID(e.ID),
// 		Type:          e.Type,
// 		Payload:       e.Payload,
// 		NextAttemptAt: db.ToTimestamptz(e.NextAttemptAt),
// 		Attempts:      int32(e.Attempts),
// 		MaxAttempts:   int32(e.MaxAttempts),
// 		CreatedAt:     db.ToTimestamptz(e.CreatedAt),
// 		UpdatedAt:     db.ToTimestamptz(e.UpdatedAt),
// 	})
// 	if err != nil {
// 		return db.NewError(err, db.ResourceOutboxEvent)
// 	}

// 	return nil
// }

// func (r *OutboxRepository) CreateBatch(ctx context.Context, events []*outbox.Event) error {
// 	if len(events) == 0 {
// 		return nil
// 	}

// 	params := make([]db.OutboxEventCreateBatchParams, 0, len(events))
// 	for _, e := range events {
// 		if e == nil {
// 			return ErrDomainPayloadRequired().
// 				Meta("resource", db.ResourceOutboxEvent.String())
// 		}

// 		params = append(params, db.OutboxEventCreateBatchParams{
// 			ID:            db.ToUUID(e.ID),
// 			Type:          e.Type,
// 			Payload:       e.Payload,
// 			CreatedAt:     db.ToTimestamptz(e.CreatedAt),
// 			UpdatedAt:     db.ToTimestamptz(e.UpdatedAt),
// 			NextAttemptAt: db.ToTimestamptz(e.NextAttemptAt),
// 			Attempts:      int32(e.Attempts),
// 			MaxAttempts:   int32(e.MaxAttempts),
// 		})
// 	}

// 	_, err := r.store.OutboxEventCreateBatch(ctx, params)
// 	if err != nil {
// 		return db.NewError(err, db.ResourceOutboxEvent)
// 	}

// 	return nil
// }

// func (r *OutboxRepository) ClaimPending(
// 	ctx context.Context,
// 	workerID uuid.UUID,
// 	leaseExpiresAt, now time.Time,
// 	limitVal int,
// ) ([]*outbox.Event, error) {
// 	rows, err := r.store.OutboxEventClaimPending(ctx, db.OutboxEventClaimPendingParams{
// 		Now:            db.ToTimestamptz(now),
// 		LimitVal:       int32(limitVal),
// 		WorkerID:       db.ToUUID(workerID),
// 		LeaseExpiresAt: db.ToTimestamptz(leaseExpiresAt),
// 	})
// 	if err != nil {
// 		return nil, db.NewError(err, db.ResourceOutboxEvent)
// 	}

// 	return outboxEventsFromRows(rows), nil
// }

// func (r *OutboxRepository) MarkProcessed(ctx context.Context, e *outbox.Event, workerID uuid.UUID) error {
// 	if e == nil {
// 		return ErrDomainPayloadRequired().
// 			Meta("resource", db.ResourceOutboxEvent.String())
// 	}

// 	rowsAffected, err := r.store.OutboxEventMarkProcessed(ctx, db.OutboxEventMarkProcessedParams{
// 		ProcessedAt: db.ToTimestamptzPtr(e.ProcessedAt),
// 		UpdatedAt:   db.ToTimestamptz(e.UpdatedAt),
// 		ID:          db.ToUUID(e.ID),
// 		WorkerID:    db.ToUUID(workerID),
// 	})
// 	if err != nil {
// 		return db.NewError(err, db.ResourceOutboxEvent)
// 	}

// 	if rowsAffected == 0 {
// 		return db.ErrWithResource(ErrOutboxEventNotFound(), db.ResourceOutboxEvent)
// 	}

// 	return nil
// }

// func (r *OutboxRepository) MarkFailure(ctx context.Context, e *outbox.Event, workerID uuid.UUID) error {
// 	if e == nil {
// 		return ErrDomainPayloadRequired().
// 			Meta("resource", db.ResourceOutboxEvent.String())
// 	}

// 	rowsAffected, err := r.store.OutboxEventMarkFailure(ctx, db.OutboxEventMarkFailureParams{
// 		Attempts:      int32(e.Attempts),
// 		NextAttemptAt: db.ToTimestamptz(e.NextAttemptAt),
// 		LastError:     db.ToTextPtr(e.LastError),
// 		UpdatedAt:     db.ToTimestamptz(e.UpdatedAt),
// 		ID:            db.ToUUID(e.ID),
// 		WorkerID:      db.ToUUID(workerID),
// 	})
// 	if err != nil {
// 		return db.NewError(err, db.ResourceOutboxEvent)
// 	}

// 	if rowsAffected == 0 {
// 		return db.ErrWithResource(ErrOutboxEventNotFound(), db.ResourceOutboxEvent)
// 	}

// 	return nil
// }

// func (r *OutboxRepository) MarkDeadLetter(ctx context.Context, e *outbox.Event, workerID uuid.UUID) error {
// 	if e == nil {
// 		return ErrDomainPayloadRequired().
// 			Meta("resource", db.ResourceOutboxEvent.String())
// 	}

// 	rowsAffected, err := r.store.OutboxEventMarkDeadLetter(ctx, db.OutboxEventMarkDeadLetterParams{
// 		LastError: db.ToTextPtr(e.LastError),
// 		UpdatedAt: db.ToTimestamptz(e.UpdatedAt),
// 		ID:        db.ToUUID(e.ID),
// 		WorkerID:  db.ToUUID(workerID),
// 	})
// 	if err != nil {
// 		return db.NewError(err, db.ResourceOutboxEvent)
// 	}

// 	if rowsAffected == 0 {
// 		return db.ErrWithResource(ErrOutboxEventNotFound(), db.ResourceOutboxEvent)
// 	}

// 	return nil
// }

// func (r *OutboxRepository) RenewLease(
// 	ctx context.Context,
// 	eventID uuid.UUID,
// 	workerID uuid.UUID,
// 	leaseExpiresAt time.Time,
// 	now time.Time,
// ) error {
// 	rowsAffected, err := r.store.OutboxEventRenewLease(ctx, db.OutboxEventRenewLeaseParams{
// 		ID:             db.ToUUID(eventID),
// 		WorkerID:       db.ToUUID(workerID),
// 		LeaseExpiresAt: db.ToTimestamptz(leaseExpiresAt),
// 		Now:            db.ToTimestamptz(now),
// 	})
// 	if err != nil {
// 		return db.NewError(err, db.ResourceOutboxEvent)
// 	}

// 	if rowsAffected == 0 {
// 		return db.ErrWithResource(ErrOutboxEventLeaseLost(), db.ResourceOutboxEvent)
// 	}

// 	return nil
// }

// func (r *OutboxRepository) ReleaseLease(ctx context.Context, e *outbox.Event, workerID uuid.UUID) error {
// 	if e == nil {
// 		return ErrDomainPayloadRequired().
// 			Meta("resource", db.ResourceOutboxEvent.String())
// 	}

// 	rowsAffected, err := r.store.OutboxEventReleaseLease(ctx, db.OutboxEventReleaseLeaseParams{
// 		UpdatedAt: db.ToTimestamptz(e.UpdatedAt),
// 		ID:        db.ToUUID(e.ID),
// 		WorkerID:  db.ToUUID(workerID),
// 	})
// 	if err != nil {
// 		return db.NewError(err, db.ResourceOutboxEvent)
// 	}

// 	if rowsAffected == 0 {
// 		return db.ErrWithResource(ErrOutboxEventLeaseLost(), db.ResourceOutboxEvent)
// 	}

// 	return nil
// }

// func (r *OutboxRepository) DeleteProcessedBatch(ctx context.Context, before time.Time, limitVal int) (int64, error) {
// 	rowsAffected, err := r.store.OutboxEventDeleteProcessedBatch(ctx, db.OutboxEventDeleteProcessedBatchParams{
// 		Before:   db.ToTimestamptz(before),
// 		LimitVal: int32(limitVal),
// 	})
// 	if err != nil {
// 		return 0, db.NewError(err, db.ResourceOutboxEvent)
// 	}

// 	return rowsAffected, nil
// }

// func outboxEventFromRow(row db.OutboxEvent) *outbox.Event {
// 	return outbox.ReconstituteEvent(
// 		db.FromUUID(row.ID),
// 		row.Type,
// 		row.Payload,
// 		db.FromTimestamptzPtr(row.ProcessedAt),
// 		int(row.Attempts),
// 		int(row.MaxAttempts),
// 		db.FromTimestamptz(row.NextAttemptAt),
// 		db.FromTextPtr(row.LastError),
// 		db.FromUUIDPtr(row.LockedBy),
// 		db.FromTimestamptzPtr(row.LeaseExpiresAt),
// 		db.FromTimestamptz(row.CreatedAt),
// 		db.FromTimestamptz(row.UpdatedAt),
// 	)
// }

// func outboxEventsFromRows(rows []db.OutboxEvent) []*outbox.Event {
// 	if len(rows) == 0 {
// 		return nil
// 	}

// 	events := make([]*outbox.Event, 0, len(rows))
// 	for _, row := range rows {
// 		events = append(events, outboxEventFromRow(row))
// 	}

// 	return events
// }
