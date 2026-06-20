package outbox_events

import (
	"bonfire-api/internal/apperr"
	"bonfire-api/internal/repository"
	"context"

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

// Create
func (s *Service) Create(ctx context.Context, p repository.OutboxEventCreateParams) (repository.OutboxEvent, error) {
	row, err := s.store.OutboxEventCreate(ctx, p)
	if err != nil {
		return repository.OutboxEvent{}, apperr.NewDBError(err)
	}
	return row, nil
}

// List
func (s *Service) List(ctx context.Context, p repository.OutboxEventListParams) ([]repository.OutboxEvent, error) {
	rows, err := s.store.OutboxEventList(ctx, repository.OutboxEventListParams{
		Limit:  p.Limit,
		Offset: p.Offset,
	})
	if err != nil {
		return nil, apperr.NewDBError(err)
	}

	return rows, nil
}

// Count
func (s *Service) Count(ctx context.Context) (int64, error) {
	count, err := s.store.OutboxEventCount(ctx)
	if err != nil {
		return 0, apperr.NewDBError(err)
	}
	return count, nil
}

// GetById
func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (repository.OutboxEvent, error) {
	var pgID pgtype.UUID
	pgID.Bytes = id
	pgID.Valid = true

	row, err := s.store.OutboxEventGetByID(ctx, pgID)
	if err != nil {
		return repository.OutboxEvent{}, apperr.NewRepositoryError(err, apperr.New(
			apperr.CodeNotFound,
			apperr.CodeNotFound.Title(),
		))

	}

	return row, nil
}

// UpdateByID
func (s *Service) UpdateByID(ctx context.Context, id uuid.UUID, p repository.OutboxEventUpdateByIDParams) (repository.OutboxEvent, error) {
	p.ID = pgtype.UUID{Bytes: id, Valid: true}

	row, err := s.store.OutboxEventUpdateByID(ctx, p)
	if err != nil {
		return repository.OutboxEvent{}, apperr.NewDBError(err)
	}
	return row, nil
}

// DeleteByID
func (s *Service) DeleteByID(ctx context.Context, id uuid.UUID) error {
	pgID := pgtype.UUID{Bytes: id, Valid: true}

	err := s.store.OutboxEventDeleteByID(ctx, pgID)
	if err != nil {
		return apperr.NewRepositoryError(err, apperr.New(
			apperr.CodeNotFound,
			apperr.CodeNotFound.Title(),
		))
	}
	return nil
}

func (s *Service) CountPending(ctx context.Context) (int64, error) {
	return s.store.OutboxEventCountPending(ctx)
}

func (s *Service) MarkProcessed(ctx context.Context, id uuid.UUID) error {
	return s.store.OutboxEventMarkProcessed(ctx, pgtype.UUID{Bytes: id, Valid: true})
}

func (s *Service) RecordFailure(ctx context.Context, p RecordFailureParams) error {
	return s.store.OutboxEventRecordFailure(ctx, repository.OutboxEventRecordFailureParams{
		ID:        pgtype.UUID{Bytes: p.ID, Valid: true},
		LastError: pgtype.Text{String: p.Error, Valid: true},
	})
}

func (s *Service) ResetAttempts(ctx context.Context, id uuid.UUID) error {
	return s.store.OutboxEventResetAttempts(ctx, pgtype.UUID{Bytes: id, Valid: true})
}

func (s *Service) MarkDeadLetter(ctx context.Context, id uuid.UUID, errMsg string) error {
	return s.store.OutboxEventMarkDeadLetter(ctx, repository.OutboxEventMarkDeadLetterParams{
		ID:        pgtype.UUID{Bytes: id, Valid: true},
		LastError: pgtype.Text{String: errMsg, Valid: true},
	})
}

func (s *Service) AcquireBatch(ctx context.Context, limit int32) ([]View, error) {
	rows, err := s.store.OutboxEventAcquireBatch(ctx, limit)
	if err != nil {
		return nil, err
	}

	views := make([]View, len(rows))
	for i, row := range rows {
		views[i] = NewView(row)
	}
	return views, nil
}

func (s *Service) DeleteOld(ctx context.Context) error {
	return s.store.OutboxEventDeleteOld(ctx)
}
