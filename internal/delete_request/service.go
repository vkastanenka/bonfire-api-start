package delete_request

import (
	"context"
	"time"

	"bonfire-api/internal/apperr"
	"bonfire-api/internal/repository"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// --- delete_request service ---

type Service struct {
	store repository.Store
}

func NewService(
	store repository.Store,
) *Service {
	return &Service{
		store: store,
	}
}

// ==========================================
// META
// ==========================================

// --- delete_request service Count ---

func (s *Service) Count(ctx context.Context) (int64, error) {
	count, err := s.store.DeleteRequestCount(ctx)
	if err != nil {
		return 0, apperr.NewDBError(err)
	}
	return count, nil
}

// ==========================================
// CREATE
// ==========================================

// --- delete_request service Create ---

func (s *Service) Create(ctx context.Context, userID uuid.UUID) (View, error) {
	row, err := s.store.DeleteRequestCreate(ctx, repository.DeleteRequestCreateParams{
		UserID:      pgtype.UUID{Bytes: userID, Valid: true},
		ScheduledAt: pgtype.Timestamptz{Time: time.Now().Add(GracePeriod), Valid: true},
	})
	if err != nil {
		return View{}, apperr.NewDBError(err)
	}
	return NewView(row), nil
}

// ==========================================
// LIST
// ==========================================

// --- delete_request service ListDue ---

func (s *Service) ListDue(ctx context.Context) ([]View, error) {
	rows, err := s.store.DeleteRequestListDue(ctx)
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

// --- delete_request service GetByUserID ---

func (s *Service) GetByUserID(ctx context.Context, userID uuid.UUID) (View, error) {
	row, err := s.store.DeleteRequestGetByUserID(ctx, pgtype.UUID{Bytes: userID, Valid: true})
	if err != nil {
		return View{}, apperr.NewDBError(err)
	}
	return NewView(row), nil
}

// ==========================================
// DELETE
// ==========================================

// --- delete_request service DeleteByUserID ---

func (s *Service) DeleteByUserID(ctx context.Context, id uuid.UUID) error {
	err := s.store.DeleteRequestDeleteByUserID(ctx, pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		return apperr.NewDBError(err)
	}
	return nil
}
