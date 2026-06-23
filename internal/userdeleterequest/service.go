package userdeleterequest

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

func (s *Service) Count(ctx context.Context) (int64, error) {
	count, err := s.store.UserDeleteRequestCount(ctx)
	if err != nil {
		return 0, apperr.NewDBError(err, UserDeleteRequest)
	}
	return count, nil
}

// ==========================================
// CREATE
// ==========================================

func (s *Service) Create(ctx context.Context, p CreateParams) (View, error) {
	row, err := s.store.UserDeleteRequestCreate(ctx, repository.UserDeleteRequestCreateParams{
		UserID:      pgtype.UUID{Bytes: p.UserID, Valid: true},
		ScheduledAt: pgtype.Timestamptz{Time: p.ScheduledAt, Valid: true},
	})
	if err != nil {
		return View{}, apperr.NewDBError(err, UserDeleteRequest)
	}
	return NewView(row), nil
}

// ==========================================
// LIST
// ==========================================

func (s *Service) ListDue(ctx context.Context) ([]View, error) {
	rows, err := s.store.UserDeleteRequestListDue(ctx)
	if err != nil {
		return nil, apperr.NewDBError(err, UserDeleteRequest)
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

func (s *Service) GetByUserID(ctx context.Context, userID uuid.UUID) (View, error) {
	row, err := s.store.UserDeleteRequestGetByUserID(ctx, pgtype.UUID{Bytes: userID, Valid: true})
	if err != nil {
		return View{}, apperr.NewDBError(err, UserDeleteRequest)
	}
	return NewView(row), nil
}

// ==========================================
// DELETE
// ==========================================

func (s *Service) DeleteByUserID(ctx context.Context, userID uuid.UUID) error {
	err := s.store.UserDeleteRequestDeleteByUserID(ctx, pgtype.UUID{Bytes: userID, Valid: true})
	if err != nil {
		return apperr.NewDBError(err, UserDeleteRequest)
	}
	return nil
}
