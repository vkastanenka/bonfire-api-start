package deletion

import (
	"context"
	"time"

	"bonfire-api/internal/repository"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// --- deletion service ---

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

// --- deletion service Count ---

func (s *Service) Count(ctx context.Context) (int64, error) {
	count, err := s.store.DeleteRequestCount(ctx)
	if err != nil {
		return 0, repository.NewError(err, repository.DomainDeleteRequest)
	}
	return count, nil
}

// ==========================================
// CREATE
// ==========================================

// --- deletion service Create ---

func (s *Service) Create(ctx context.Context, userID uuid.UUID) (View, error) {
	row, err := s.store.DeleteRequestCreate(ctx, repository.DeleteRequestCreateParams{
		UserID:      pgtype.UUID{Bytes: userID, Valid: true},
		ScheduledAt: pgtype.Timestamptz{Time: time.Now().Add(GracePeriod), Valid: true},
	})
	if err != nil {
		return View{}, repository.NewError(err, repository.DomainDeleteRequest)
	}
	return NewView(row), nil
}

// ==========================================
// LIST
// ==========================================

// --- deletion service ListDue ---

func (s *Service) ListDue(ctx context.Context) ([]View, error) {
	rows, err := s.store.DeleteRequestListDue(ctx)
	if err != nil {
		return nil, repository.NewError(err, repository.DomainDeleteRequest)
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

// --- deletion service GetByUserID ---

func (s *Service) GetByUserID(ctx context.Context, userID uuid.UUID) (View, error) {
	row, err := s.store.DeleteRequestGetByUserID(ctx, pgtype.UUID{Bytes: userID, Valid: true})
	if err != nil {
		return View{}, repository.NewError(err, repository.DomainDeleteRequest)
	}
	return NewView(row), nil
}

// ==========================================
// DELETE
// ==========================================

// --- deletion service DeleteByUserID ---

func (s *Service) DeleteByUserID(ctx context.Context, id uuid.UUID) error {
	err := s.store.DeleteRequestDeleteByUserID(ctx, pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		return repository.NewError(err, repository.DomainDeleteRequest)
	}
	return nil
}
