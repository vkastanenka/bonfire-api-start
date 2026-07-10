package outbox

import (
	"context"
	"encoding/json"
	"strconv"

	"bonfire-api/internal/repository"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type Store interface {
	repository.Querier
}

type Service struct {
	store Store
}

func NewService(store Store) *Service {
	return &Service{store: store}
}

type CreateParams struct {
	EventType string
	Payload   json.RawMessage
}

func (s *Service) Create(ctx context.Context, p CreateParams) (Event, error) {
	row, err := s.store.OutboxEventCreate(ctx, repository.OutboxEventCreateParams{
		EventType: p.EventType,
		Payload:   []byte(p.Payload),
	})
	if err != nil {
		return Event{}, repository.NewError(err, repository.ScopeOutboxEvent)
	}

	return FromRepository(row), nil
}

func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (Event, error) {
	row, err := s.store.OutboxEventGetByID(ctx, pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		return Event{}, repository.NewError(err, repository.ScopeOutboxEvent)
	}

	return FromRepository(row), nil
}

type ListParams struct {
	Cursor *uuid.UUID
	Limit  int32
}

func (s *Service) List(ctx context.Context, p ListParams) ([]Event, error) {
	var pgCursor pgtype.UUID
	if p.Cursor != nil {
		pgCursor = pgtype.UUID{Bytes: *p.Cursor, Valid: true}
	}

	rows, err := s.store.OutboxEventList(ctx, repository.OutboxEventListParams{
		Column1: pgCursor,
		Limit:   p.Limit,
	})
	if err != nil {
		return nil, repository.NewError(err, repository.ScopeOutboxEvent)
	}

	events := make([]Event, len(rows))
	for i, row := range rows {
		events[i] = FromRepository(row)
	}

	return events, nil
}

type AcquireBatchParams struct {
	Limit                  int32
	WorkerID               uuid.UUID
	LeaseDurationInSeconds int32
}

func (s *Service) AcquireBatch(ctx context.Context, p AcquireBatchParams) ([]Event, error) {
	leaseIntervalStr := strconv.Itoa(int(p.LeaseDurationInSeconds))

	rows, err := s.store.OutboxEventAcquireBatch(ctx, repository.OutboxEventAcquireBatchParams{
		Limit:    p.Limit,
		LockedBy: pgtype.UUID{Bytes: p.WorkerID, Valid: true},
		Column3:  leaseIntervalStr,
	})
	if err != nil {
		return nil, repository.NewError(err, repository.ScopeOutboxEvent)
	}

	events := make([]Event, len(rows))
	for i, row := range rows {
		events[i] = FromRepository(row)
	}

	return events, nil
}

func (s *Service) MarkProcessed(ctx context.Context, id uuid.UUID) (Event, error) {
	row, err := s.store.OutboxEventMarkProcessed(ctx, pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		return Event{}, repository.NewError(err, repository.ScopeOutboxEvent)
	}

	return FromRepository(row), nil
}

type RecordFailureParams struct {
	ID        uuid.UUID
	LastError string
}

func (s *Service) RecordFailure(ctx context.Context, p RecordFailureParams) (Event, error) {
	row, err := s.store.OutboxEventRecordFailure(ctx, repository.OutboxEventRecordFailureParams{
		ID:        pgtype.UUID{Bytes: p.ID, Valid: true},
		LastError: pgtype.Text{String: p.LastError, Valid: p.LastError != ""},
	})
	if err != nil {
		return Event{}, repository.NewError(err, repository.ScopeOutboxEvent)
	}

	return FromRepository(row), nil
}

type MarkDeadLetterParams struct {
	ID     uuid.UUID
	Reason string
}

func (s *Service) MarkDeadLetter(ctx context.Context, p MarkDeadLetterParams) (Event, error) {
	row, err := s.store.OutboxEventMarkDeadLetter(ctx, repository.OutboxEventMarkDeadLetterParams{
		ID:        pgtype.UUID{Bytes: p.ID, Valid: true},
		LastError: pgtype.Text{String: p.Reason, Valid: p.Reason != ""},
	})
	if err != nil {
		return Event{}, repository.NewError(err, repository.ScopeOutboxEvent)
	}

	return FromRepository(row), nil
}

func (s *Service) ResetAttempts(ctx context.Context, id uuid.UUID) (Event, error) {
	row, err := s.store.OutboxEventResetAttempts(ctx, pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		return Event{}, repository.NewError(err, repository.ScopeOutboxEvent)
	}

	return FromRepository(row), nil
}

func (s *Service) DeleteByID(ctx context.Context, id uuid.UUID) error {
	err := s.store.OutboxEventDeleteByID(ctx, pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		return repository.NewError(err, repository.ScopeOutboxEvent)
	}

	return nil
}

func (s *Service) PurgeProcessed(ctx context.Context) error {
	err := s.store.OutboxEventPurgeProcessed(ctx)
	if err != nil {
		return repository.NewError(err, repository.ScopeOutboxEvent)
	}

	return nil
}
