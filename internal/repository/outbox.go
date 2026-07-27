package repository

import (
	"context"
	"encoding/json"
	"time"

	"bonfire-api/internal/db"
	"bonfire-api/internal/errs"
	"bonfire-api/internal/outbox"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type OutboxStore interface {
	OutboxEventCreate(ctx context.Context, arg db.OutboxEventCreateParams) (db.OutboxEvent, error)
	OutboxEventGet(ctx context.Context, id pgtype.UUID) (db.OutboxEvent, error)
	OutboxEventList(ctx context.Context, arg db.OutboxEventListParams) ([]db.OutboxEvent, error)
	OutboxEventAcquireBatch(ctx context.Context, arg db.OutboxEventAcquireBatchParams) ([]db.OutboxEvent, error)
	OutboxEventUpdate(ctx context.Context, arg db.OutboxEventUpdateParams) (db.OutboxEvent, error)
	OutboxEventRenewLease(ctx context.Context, arg db.OutboxEventRenewLeaseParams) error
	OutboxEventDelete(ctx context.Context, id pgtype.UUID) error
	OutboxEventPurgeProcessed(ctx context.Context, retentionDays int32) error
}

type Outbox struct {
	store OutboxStore
}

func NewOutbox(store OutboxStore) *Outbox {
	return &Outbox{store: store}
}

// Publish creates and persists a new domain outbox event
func (r *Outbox) Publish(ctx context.Context, variant string, payload any) (*outbox.Event, error) {
	var payloadBytes []byte
	switch v := payload.(type) {
	case []byte:
		payloadBytes = v
	default:
		var err error
		payloadBytes, err = json.Marshal(payload)
		if err != nil {
			return nil, errs.InvalidArgument("failed to marshal outbox event payload").Wrap(err)
		}
	}

	event, err := outbox.New(variant, payloadBytes)
	if err != nil {
		return nil, errs.InvalidArgument("failed to instantiate outbox event").Wrap(err)
	}

	if err := r.Create(ctx, event); err != nil {
		return nil, err
	}

	return event, nil
}

// Create persists a newly instantiated domain event with all initial state flags
func (r *Outbox) Create(ctx context.Context, event *outbox.Event) error {
	_, err := r.store.OutboxEventCreate(ctx, db.OutboxEventCreateParams{
		ID:             db.UUID(event.ID().UUID()),
		LockedBy:       db.UUIDPtr(event.LockedBy()),
		CreatedAt:      db.Timestamptz(event.CreatedAt()),
		UpdatedAt:      db.Timestamptz(event.UpdatedAt()),
		NextAttemptAt:  db.Timestamptz(event.NextAttemptAt()),
		LeaseExpiresAt: db.TimestamptzPtr(event.LeaseExpiresAt()),
		ProcessedAt:    db.TimestamptzPtr(event.ProcessedAt()),
		Attempts:       event.Attempts(),
		MaxAttempts:    event.MaxAttempts(),
		EventType:      event.EventType(),
		LastError:      db.TextPtr(event.LastError()),
		Payload:        event.Payload(),
	})
	if err != nil {
		return db.NewError(err, db.EntityOutboxEvent)
	}
	return nil
}

func (r *Outbox) Get(ctx context.Context, id outbox.EventID) (*outbox.Event, error) {
	row, err := r.store.OutboxEventGet(ctx, db.UUID(id.UUID()))
	if err != nil {
		return nil, db.NewError(err, db.EntityOutboxEvent)
	}

	return outboxFromRow(row)
}

func (r *Outbox) Save(ctx context.Context, event *outbox.Event) error {
	_, err := r.store.OutboxEventUpdate(ctx, db.OutboxEventUpdateParams{
		ID:             db.UUID(event.ID().UUID()),
		LockedBy:       db.UUIDPtr(event.LockedBy()),
		LeaseExpiresAt: db.TimestamptzPtr(event.LeaseExpiresAt()),
		ProcessedAt:    db.TimestamptzPtr(event.ProcessedAt()),
		Attempts:       event.Attempts(),
		MaxAttempts:    event.MaxAttempts(),
		NextAttemptAt:  db.Timestamptz(event.NextAttemptAt()),
		LastError:      db.TextPtr(event.LastError()),
		UpdatedAt:      db.Timestamptz(event.UpdatedAt()),
	})
	if err != nil {
		return db.NewError(err, db.EntityOutboxEvent)
	}
	return nil
}

// List executes pagination over outbox events
func (r *Outbox) List(ctx context.Context, cursorID *outbox.EventID, limit int32) ([]*outbox.Event, error) {
	var cursorUUID *uuid.UUID
	if cursorID != nil {
		u := cursorID.UUID()
		cursorUUID = &u
	}

	rows, err := r.store.OutboxEventList(ctx, db.OutboxEventListParams{
		CursorID:    db.UUIDPtr(cursorUUID),
		ResultLimit: limit,
	})
	if err != nil {
		return nil, db.NewError(err, db.EntityOutboxEvent)
	}

	events := make([]*outbox.Event, len(rows))
	for i, row := range rows {
		evt, err := outboxFromRow(row)
		if err != nil {
			return nil, err
		}
		events[i] = evt
	}

	return events, nil
}

// AcquireBatch locks available pending events for processing by workerID
func (r *Outbox) AcquireBatch(ctx context.Context, workerID uuid.UUID, leaseDurationSec, batchSize int32) ([]*outbox.Event, error) {
	rows, err := r.store.OutboxEventAcquireBatch(ctx, db.OutboxEventAcquireBatchParams{
		WorkerID:             db.UUID(workerID),
		LeaseDurationSeconds: leaseDurationSec,
		BatchSize:            batchSize,
	})
	if err != nil {
		return nil, db.NewError(err, db.EntityOutboxEvent)
	}

	events := make([]*outbox.Event, len(rows))
	for i, row := range rows {
		evt, err := outboxFromRow(row)
		if err != nil {
			return nil, err
		}
		events[i] = evt
	}

	return events, nil
}

func (r *Outbox) MarkProcessed(ctx context.Context, id outbox.EventID) (*outbox.Event, error) {
	event, err := r.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	event.MarkProcessed(time.Now())

	if err := r.Save(ctx, event); err != nil {
		return nil, err
	}

	return event, nil
}

func (r *Outbox) RecordFailure(ctx context.Context, id outbox.EventID, lastError string) (*outbox.Event, error) {
	event, err := r.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	event.RecordFailure(lastError, time.Now())

	if err := r.Save(ctx, event); err != nil {
		return nil, err
	}

	return event, nil
}

func (r *Outbox) MarkDeadLetter(ctx context.Context, id outbox.EventID, reason string) (*outbox.Event, error) {
	event, err := r.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	event.MarkDeadLetter(reason, time.Now())

	if err := r.Save(ctx, event); err != nil {
		return nil, err
	}

	return event, nil
}

// RenewLease extends lock duration for an in-flight worker job
func (r *Outbox) RenewLease(ctx context.Context, id outbox.EventID, workerID uuid.UUID, leaseDurationSec int32) error {
	err := r.store.OutboxEventRenewLease(ctx, db.OutboxEventRenewLeaseParams{
		ID:                   db.UUID(id.UUID()),
		WorkerID:             db.UUID(workerID),
		LeaseDurationSeconds: leaseDurationSec,
	})
	if err != nil {
		return db.NewError(err, db.EntityOutboxEvent)
	}
	return nil
}

// Delete permanently removes an event by ID
func (r *Outbox) Delete(ctx context.Context, id outbox.EventID) error {
	err := r.store.OutboxEventDelete(ctx, db.UUID(id.UUID()))
	if err != nil {
		return db.NewError(err, db.EntityOutboxEvent)
	}
	return nil
}

// PurgeProcessed cleans up processed events older than retention window
func (r *Outbox) PurgeProcessed(ctx context.Context, retentionDays int32) error {
	err := r.store.OutboxEventPurgeProcessed(ctx, retentionDays)
	if err != nil {
		return db.NewError(err, db.EntityOutboxEvent)
	}
	return nil
}

func outboxFromRow(row db.OutboxEvent) (*outbox.Event, error) {
	id, err := outbox.NewEventID(uuid.UUID(row.ID.Bytes))
	if err != nil {
		return nil, errs.Internal("failed to parse outbox event ID from database").
			Wrap(err).
			Reason("CORRUPT_DATABASE_RECORD").
			Resource("OutboxEvent", uuid.UUID(row.ID.Bytes).String(), "", "database row mapping")
	}

	return outbox.Reconstitute(
		id,
		row.EventType,
		row.Payload,
		db.TimePtr(row.ProcessedAt),
		row.Attempts,
		row.MaxAttempts,
		row.NextAttemptAt.Time.UTC(),
		db.UUIDPtrFromDB(row.LockedBy),
		db.TimePtr(row.LeaseExpiresAt),
		db.StringPtr(row.LastError),
		row.CreatedAt.Time.UTC(),
		row.UpdatedAt.Time.UTC(),
	), nil
}
