package repository

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"

	"bonfire-api/internal/db"
	"bonfire-api/internal/errs"
	"bonfire-api/internal/outbox"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type OutboxStore interface {
	OutboxEventCreate(ctx context.Context, arg db.OutboxEventCreateParams) (db.OutboxEvent, error)
	OutboxEventGet(ctx context.Context, id pgtype.UUID) (db.OutboxEvent, error)
	OutboxEventList(ctx context.Context, arg db.OutboxEventListParams) ([]db.OutboxEvent, error)
	OutboxEventAcquireBatch(ctx context.Context, arg db.OutboxEventAcquireBatchParams) ([]db.OutboxEvent, error)
	OutboxEventMarkProcessed(ctx context.Context, id pgtype.UUID) (db.OutboxEvent, error)
	OutboxEventRecordFailure(ctx context.Context, arg db.OutboxEventRecordFailureParams) (db.OutboxEvent, error)
	OutboxEventMarkDeadLetter(ctx context.Context, arg db.OutboxEventMarkDeadLetterParams) (db.OutboxEvent, error)
	OutboxEventResetAttempts(ctx context.Context, id pgtype.UUID) (db.OutboxEvent, error)
	OutboxEventDelete(ctx context.Context, id pgtype.UUID) error
	OutboxEventPurgeProcessed(ctx context.Context) error
}

type Outbox struct {
	store OutboxStore
}

func NewOutbox(store OutboxStore) *Outbox {
	return &Outbox{store: store}
}

func (r *Outbox) Publish(ctx context.Context, p outbox.PublishParams) (outbox.Event, error) {
	jsonBytes, err := json.Marshal(p.Payload)
	if err != nil {
		return outbox.Event{}, errs.Internal("failed to marshal outbox event payload").Wrap(err)
	}

	row, err := r.store.OutboxEventCreate(ctx, db.OutboxEventCreateParams{
		EventType: p.Variant,
		Payload:   jsonBytes,
	})
	if err != nil {
		return outbox.Event{}, errs.Internal("failed to create outbox event").Wrap(err)
	}

	return outboxFromDB(row), nil
}

func (r *Outbox) Get(ctx context.Context, id uuid.UUID) (outbox.Event, error) {
	row, err := r.store.OutboxEventGet(ctx, pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return outbox.Event{}, errs.NotFound("outbox event not found").Wrap(err)
		}
		return outbox.Event{}, errs.Internal("failed to fetch outbox event").Wrap(err)
	}

	return outboxFromDB(row), nil
}

func (r *Outbox) List(ctx context.Context, p outbox.ListParams) ([]outbox.Event, error) {
	var pgCursor pgtype.UUID
	if p.Cursor != nil {
		pgCursor = pgtype.UUID{Bytes: *p.Cursor, Valid: true}
	}

	rows, err := r.store.OutboxEventList(ctx, db.OutboxEventListParams{
		Column1: pgCursor,
		Limit:   p.Limit,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return []outbox.Event{}, nil
		}
		return nil, errs.Internal("failed to list outbox events").Wrap(err)
	}

	events := make([]outbox.Event, len(rows))
	for i, row := range rows {
		events[i] = outboxFromDB(row)
	}

	return events, nil
}

func (r *Outbox) AcquireBatch(ctx context.Context, p outbox.AcquireBatchParams) ([]outbox.Event, error) {
	leaseIntervalStr := strconv.Itoa(int(p.LeaseDurationInSeconds))

	rows, err := r.store.OutboxEventAcquireBatch(ctx, db.OutboxEventAcquireBatchParams{
		Limit:    p.Limit,
		LockedBy: pgtype.UUID{Bytes: p.WorkerID, Valid: true},
		Column3:  leaseIntervalStr,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return []outbox.Event{}, nil
		}
		return nil, errs.Internal("failed to acquire outbox event batch").Wrap(err)
	}

	events := make([]outbox.Event, len(rows))
	for i, row := range rows {
		events[i] = outboxFromDB(row)
	}

	return events, nil
}

func (r *Outbox) MarkProcessed(ctx context.Context, id uuid.UUID) (outbox.Event, error) {
	row, err := r.store.OutboxEventMarkProcessed(ctx, pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return outbox.Event{}, errs.NotFound("outbox event not found").Wrap(err)
		}
		return outbox.Event{}, errs.Internal("failed to mark outbox event processed").Wrap(err)
	}

	return outboxFromDB(row), nil
}

func (r *Outbox) RecordFailure(ctx context.Context, p outbox.RecordFailureParams) (outbox.Event, error) {
	row, err := r.store.OutboxEventRecordFailure(ctx, db.OutboxEventRecordFailureParams{
		ID:        pgtype.UUID{Bytes: p.ID, Valid: true},
		LastError: pgtype.Text{String: p.LastError, Valid: p.LastError != ""},
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return outbox.Event{}, errs.NotFound("outbox event not found").Wrap(err)
		}
		return outbox.Event{}, errs.Internal("failed to record outbox event failure").Wrap(err)
	}

	return outboxFromDB(row), nil
}

func (r *Outbox) MarkDeadLetter(ctx context.Context, p outbox.MarkDeadLetterParams) (outbox.Event, error) {
	row, err := r.store.OutboxEventMarkDeadLetter(ctx, db.OutboxEventMarkDeadLetterParams{
		ID:        pgtype.UUID{Bytes: p.ID, Valid: true},
		LastError: pgtype.Text{String: p.Reason, Valid: p.Reason != ""},
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return outbox.Event{}, errs.NotFound("outbox event not found").Wrap(err)
		}
		return outbox.Event{}, errs.Internal("failed to mark outbox event dead letter").Wrap(err)
	}

	return outboxFromDB(row), nil
}

func (r *Outbox) ResetAttempts(ctx context.Context, id uuid.UUID) (outbox.Event, error) {
	row, err := r.store.OutboxEventResetAttempts(ctx, pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return outbox.Event{}, errs.NotFound("outbox event not found").Wrap(err)
		}
		return outbox.Event{}, errs.Internal("failed to reset outbox event attempts").Wrap(err)
	}

	return outboxFromDB(row), nil
}

func (r *Outbox) Delete(ctx context.Context, id uuid.UUID) error {
	err := r.store.OutboxEventDelete(ctx, pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return errs.NotFound("outbox event not found").Wrap(err)
		}
		return errs.Internal("failed to delete outbox event").Wrap(err)
	}
	return nil
}

func (r *Outbox) PurgeProcessed(ctx context.Context) error {
	err := r.store.OutboxEventPurgeProcessed(ctx)
	if err != nil {
		return errs.Internal("failed to purge processed outbox events").Wrap(err)
	}
	return nil
}

func outboxFromDB(row db.OutboxEvent) outbox.Event {
	e := outbox.Event{
		ID:            uuid.UUID(row.ID.Bytes),
		EventType:     row.EventType,
		Payload:       row.Payload,
		Attempts:      row.Attempts,
		MaxAttempts:   row.MaxAttempts,
		NextAttemptAt: row.NextAttemptAt.Time,
		CreatedAt:     row.CreatedAt.Time,
		UpdatedAt:     row.UpdatedAt.Time,
	}

	if row.ProcessedAt.Valid {
		t := row.ProcessedAt.Time
		e.ProcessedAt = &t
	}

	if row.LockedBy.Valid {
		id := uuid.UUID(row.LockedBy.Bytes)
		e.LockedBy = &id
	}

	if row.LeaseExpiresAt.Valid {
		t := row.LeaseExpiresAt.Time
		e.LeaseExpiresAt = &t
	}

	if row.LastError.Valid {
		e.LastError = &row.LastError.String
	}

	return e
}
