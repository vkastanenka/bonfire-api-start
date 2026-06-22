package outbox

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
		return 0, apperr.NewDBError(err, OutboxEvent)
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
		return View{}, apperr.NewDBError(err, OutboxEvent)
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
		return nil, apperr.NewDBError(err, OutboxEvent)
	}

	views := make([]View, len(rows))
	for i, row := range rows {
		views[i] = NewView(row)
	}
	return views, nil
}

// AcquireBatch
func (s *Service) AcquireBatch(ctx context.Context, limit int32) ([]View, error) {
	rows, err := s.store.OutboxEventAcquireBatch(ctx, limit)
	if err != nil {
		return nil, apperr.NewDBError(err, OutboxEvent)
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
	row, err := s.store.OutboxEventGetByID(ctx, pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		return View{}, apperr.NewDBError(err, OutboxEvent)
	}
	return NewView(row), nil
}

// ==========================================
// UPDATE
// ==========================================

// MarkProcessed
func (s *Service) MarkProcessed(ctx context.Context, id uuid.UUID) (View, error) {
	row, err := s.store.OutboxEventMarkProcessed(ctx, pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		return View{}, apperr.NewDBError(err, OutboxEvent)
	}
	return NewView(row), nil
}

// RecordFailure
func (s *Service) RecordFailure(ctx context.Context, p RecordFailureParams) (View, error) {
	row, err := s.store.OutboxEventRecordFailure(ctx, repository.OutboxEventRecordFailureParams{
		ID:        pgtype.UUID{Bytes: p.ID, Valid: true},
		LastError: pgtype.Text{String: p.Error, Valid: true},
	})
	if err != nil {
		return View{}, apperr.NewDBError(err, OutboxEvent)
	}
	return NewView(row), nil
}

// ResetAttempts
func (s *Service) ResetAttempts(ctx context.Context, id uuid.UUID) (View, error) {
	row, err := s.store.OutboxEventResetAttempts(ctx, pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		return View{}, apperr.NewDBError(err, OutboxEvent)
	}
	return NewView(row), nil
}

// MarkDeadLetter
func (s *Service) MarkDeadLetter(ctx context.Context, p MarkDeadLetterParams) (View, error) {
	row, err := s.store.OutboxEventMarkDeadLetter(ctx, repository.OutboxEventMarkDeadLetterParams{
		ID:        pgtype.UUID{Bytes: p.ID, Valid: true},
		LastError: pgtype.Text{String: p.Error, Valid: true},
	})
	if err != nil {
		return View{}, apperr.NewDBError(err, OutboxEvent)
	}
	return NewView(row), nil
}

// ==========================================
// DELETE
// ==========================================

func (s *Service) DeleteByID(ctx context.Context, id uuid.UUID) error {
	pgID := pgtype.UUID{Bytes: id, Valid: true}
	err := s.store.OutboxEventDeleteByID(ctx, pgID)
	if err != nil {
		return apperr.NewDBError(err, OutboxEvent)
	}
	return nil
}

func (s *Service) PurgeProcessed(ctx context.Context) error {
	if err := s.store.OutboxEventPurgeProcessed(ctx); err != nil {
		return apperr.NewDBError(err, OutboxEvent)
	}
	return nil
}
