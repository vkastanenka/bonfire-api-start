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

// ==========================================
// META
// ==========================================

// Count
func (s *Service) Count(ctx context.Context) (int64, error) {
	count, err := s.store.OutboxEventCount(ctx)
	if err != nil {
		return 0, apperr.NewDBError(err)
	}
	return count, nil
}

// ==========================================
// CREATE
// ==========================================

// Create
func (s *Service) Create(ctx context.Context, p CreateParams) (View, error) {
	row, err := s.store.OutboxEventCreate(ctx, repository.OutboxEventCreateParams{
		EventType: p.EventType,
		Payload:   p.Payload,
	})
	if err != nil {
		return View{}, apperr.NewDBError(err)
	}
	return NewView(row), nil
}

// ==========================================
// LIST
// ==========================================

// List
func (s *Service) List(ctx context.Context, p ListParams) ([]View, error) {
	var pgCursor pgtype.UUID
	if p.Cursor != nil {
		pgCursor = pgtype.UUID{Bytes: *p.Cursor, Valid: true}
	}

	rows, err := s.store.OutboxEventList(ctx, repository.OutboxEventListParams{
		Column1: pgCursor,
		Limit:   p.Limit,
	})
	if err != nil {
		return nil, apperr.NewDBError(err)
	}

	views := make([]View, len(rows))
	for i, row := range rows {
		views[i] = NewView(repository.OutboxEvent(row))
	}
	return views, nil
}

// AcquireBatch
func (s *Service) AcquireBatch(ctx context.Context, limit int32) ([]View, error) {
	rows, err := s.store.OutboxEventAcquireBatch(ctx, limit)
	if err != nil {
		return nil, apperr.NewDBError(err)
	}

	views := make([]View, len(rows))
	for i, row := range rows {
		views[i] = NewView(row)
	}
	return views, nil
}

// ==========================================
// GET
// ==========================================

// GetByID
func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (View, error) {
	pgID := pgtype.UUID{Bytes: id, Valid: true}

	row, err := s.store.OutboxEventGetByID(ctx, pgID)
	if err != nil {
		return View{}, apperr.NewRepositoryError(err, apperr.New(
			apperr.CodeNotFound,
			apperr.CodeNotFound.Title(),
		))
	}

	// Assuming sqlc generated a matching struct or reused OutboxEvent
	return NewView(repository.OutboxEvent(row)), nil
}

// ==========================================
// UPDATE
// ==========================================

// MarkProcessed
func (s *Service) MarkProcessed(ctx context.Context, id uuid.UUID) error {
	pgID := pgtype.UUID{Bytes: id, Valid: true}
	if err := s.store.OutboxEventMarkProcessed(ctx, pgID); err != nil {
		return apperr.NewDBError(err)
	}
	return nil
}

// RecordFailure
func (s *Service) RecordFailure(ctx context.Context, p RecordFailureParams) error {
	err := s.store.OutboxEventRecordFailure(ctx, repository.OutboxEventRecordFailureParams{
		ID:        pgtype.UUID{Bytes: p.ID, Valid: true},
		LastError: pgtype.Text{String: p.Error, Valid: true},
	})
	if err != nil {
		return apperr.NewDBError(err)
	}
	return nil
}

// ResetAttempts
func (s *Service) ResetAttempts(ctx context.Context, id uuid.UUID) error {
	pgID := pgtype.UUID{Bytes: id, Valid: true}
	if err := s.store.OutboxEventResetAttempts(ctx, pgID); err != nil {
		return apperr.NewDBError(err)
	}
	return nil
}

// MarkDeadLetter
func (s *Service) MarkDeadLetter(ctx context.Context, id uuid.UUID, errMsg string) error {
	err := s.store.OutboxEventMarkDeadLetter(ctx, repository.OutboxEventMarkDeadLetterParams{
		ID:        pgtype.UUID{Bytes: id, Valid: true},
		LastError: pgtype.Text{String: errMsg, Valid: true},
	})
	if err != nil {
		return apperr.NewDBError(err)
	}
	return nil
}

// ==========================================
// DELETE
// ==========================================

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

func (s *Service) PurgeProcessed(ctx context.Context) error {
	if err := s.store.OutboxEventPurgeProcessed(ctx); err != nil {
		return apperr.NewDBError(err)
	}
	return nil
}
