package repository

import (
	"context"
	"errors"
	"strconv"

	"bonfire-api/internal/apperr"
	"bonfire-api/internal/db"
	"bonfire-api/internal/outbox"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type Outbox struct {
	store db.Store
}

func NewOutbox(store db.Store) *Outbox {
	return &Outbox{store: store}
}

func (o *Outbox) Create(ctx context.Context, p outbox.CreateParams) (outbox.Event, error) {
	row, err := o.store.OutboxEventCreate(ctx, db.OutboxEventCreateParams{
		EventType: p.EventType,
		Payload:   []byte(p.Payload),
	})
	if err != nil {
		return outbox.Event{}, apperr.NewInternal(err, apperr.WithMsg("failed to create outbox event"))
	}

	return outboxFromDB(row), nil
}

func (o *Outbox) GetByID(ctx context.Context, id uuid.UUID) (outbox.Event, error) {
	row, err := o.store.OutboxEventGetByID(ctx, pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return outbox.Event{}, apperr.NewNotFound(err, apperr.WithMsg("outbox event not found"))
		}
		return outbox.Event{}, apperr.NewInternal(err, apperr.WithMsg("failed to fetch outbox event"))
	}

	return outboxFromDB(row), nil
}

func (o *Outbox) List(ctx context.Context, p outbox.ListParams) ([]outbox.Event, error) {
	var pgCursor pgtype.UUID
	if p.Cursor != nil {
		pgCursor = pgtype.UUID{Bytes: *p.Cursor, Valid: true}
	}

	rows, err := o.store.OutboxEventList(ctx, db.OutboxEventListParams{
		Column1: pgCursor,
		Limit:   p.Limit,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return []outbox.Event{}, nil
		}
		return nil, apperr.NewInternal(err, apperr.WithMsg("failed to list outbox events"))
	}

	events := make([]outbox.Event, len(rows))
	for i, row := range rows {
		events[i] = outboxFromDB(row)
	}

	return events, nil
}

func (o *Outbox) AcquireBatch(ctx context.Context, p outbox.AcquireBatchParams) ([]outbox.Event, error) {
	leaseIntervalStr := strconv.Itoa(int(p.LeaseDurationInSeconds))

	rows, err := o.store.OutboxEventAcquireBatch(ctx, db.OutboxEventAcquireBatchParams{
		Limit:    p.Limit,
		LockedBy: pgtype.UUID{Bytes: p.WorkerID, Valid: true},
		Column3:  leaseIntervalStr,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return []outbox.Event{}, nil
		}
		return nil, apperr.NewInternal(err, apperr.WithMsg("failed to acquire outbox event batch"))
	}

	events := make([]outbox.Event, len(rows))
	for i, row := range rows {
		events[i] = outboxFromDB(row)
	}

	return events, nil
}

func (o *Outbox) MarkProcessed(ctx context.Context, id uuid.UUID) (outbox.Event, error) {
	row, err := o.store.OutboxEventMarkProcessed(ctx, pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return outbox.Event{}, apperr.NewNotFound(err, apperr.WithMsg("outbox event not found"))
		}
		return outbox.Event{}, apperr.NewInternal(err, apperr.WithMsg("failed to mark outbox event processed"))
	}

	return outboxFromDB(row), nil
}

func (o *Outbox) RecordFailure(ctx context.Context, p outbox.RecordFailureParams) (outbox.Event, error) {
	row, err := o.store.OutboxEventRecordFailure(ctx, db.OutboxEventRecordFailureParams{
		ID:        pgtype.UUID{Bytes: p.ID, Valid: true},
		LastError: pgtype.Text{String: p.LastError, Valid: p.LastError != ""},
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return outbox.Event{}, apperr.NewNotFound(err, apperr.WithMsg("outbox event not found"))
		}
		return outbox.Event{}, apperr.NewInternal(err, apperr.WithMsg("failed to record outbox event failure"))
	}

	return outboxFromDB(row), nil
}

func (o *Outbox) MarkDeadLetter(ctx context.Context, p outbox.MarkDeadLetterParams) (outbox.Event, error) {
	row, err := o.store.OutboxEventMarkDeadLetter(ctx, db.OutboxEventMarkDeadLetterParams{
		ID:        pgtype.UUID{Bytes: p.ID, Valid: true},
		LastError: pgtype.Text{String: p.Reason, Valid: p.Reason != ""},
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return outbox.Event{}, apperr.NewNotFound(err, apperr.WithMsg("outbox event not found"))
		}
		return outbox.Event{}, apperr.NewInternal(err, apperr.WithMsg("failed to mark outbox event dead letter"))
	}

	return outboxFromDB(row), nil
}

func (o *Outbox) ResetAttempts(ctx context.Context, id uuid.UUID) (outbox.Event, error) {
	row, err := o.store.OutboxEventResetAttempts(ctx, pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return outbox.Event{}, apperr.NewNotFound(err, apperr.WithMsg("outbox event not found"))
		}
		return outbox.Event{}, apperr.NewInternal(err, apperr.WithMsg("failed to reset outbox event attempts"))
	}

	return outboxFromDB(row), nil
}

func (o *Outbox) DeleteByID(ctx context.Context, id uuid.UUID) error {
	err := o.store.OutboxEventDeleteByID(ctx, pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return apperr.NewNotFound(err, apperr.WithMsg("outbox event not found"))
		}
		return apperr.NewInternal(err, apperr.WithMsg("failed to delete outbox event"))
	}
	return nil
}

func (o *Outbox) PurgeProcessed(ctx context.Context) error {
	err := o.store.OutboxEventPurgeProcessed(ctx)
	if err != nil {
		return apperr.NewInternal(err, apperr.WithMsg("failed to purge processed outbox events"))
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

var _ outbox.Repository = (*Outbox)(nil)
