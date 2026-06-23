package user

import (
	"bonfire-api/internal/apperr"
	"bonfire-api/internal/repository"
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// ==========================================
// META
// ==========================================

// CountDeleteRequests
func (s *Service) CountDeleteRequests(ctx context.Context) (int64, error) {
	count, err := s.store.UserDeleteRequestCount(ctx)
	if err != nil {
		return 0, apperr.NewDBError(err, DomainDeleteRequest)
	}
	return count, nil
}

// ==========================================
// CREATE
// ==========================================

func (s *Service) CreateDeleteRequest(ctx context.Context, p CreateDeleteRequestParams) (DeleteRequestView, error) {
	row, err := s.store.UserDeleteRequestCreate(ctx, repository.UserDeleteRequestCreateParams{
		UserID:      pgtype.UUID{Bytes: p.UserID, Valid: true},
		ScheduledAt: pgtype.Timestamptz{Time: p.ScheduledAt, Valid: true},
	})
	if err != nil {
		return DeleteRequestView{}, apperr.NewDBError(err, DomainDeleteRequest)
	}
	return NewDeleteRequestView(row), nil
}

// ==========================================
// LIST
// ==========================================

func (s *Service) ListDeleteRequestDue(ctx context.Context) ([]DeleteRequestView, error) {
	rows, err := s.store.UserDeleteRequestListDue(ctx)
	if err != nil {
		return nil, apperr.NewDBError(err, DomainDeleteRequest)
	}

	views := make([]DeleteRequestView, len(rows))
	for i, row := range rows {
		views[i] = NewDeleteRequestView(row)
	}

	return views, nil
}

// ==========================================
// GET
// ==========================================

func (s *Service) GetDeleteRequestByUserID(ctx context.Context, userID uuid.UUID) (DeleteRequestView, error) {
	row, err := s.store.UserDeleteRequestGetByUserID(ctx, pgtype.UUID{Bytes: userID, Valid: true})
	if err != nil {
		return DeleteRequestView{}, apperr.NewDBError(err, DomainDeleteRequest)
	}
	return NewDeleteRequestView(row), nil
}

// ==========================================
// DELETE
// ==========================================

func (s *Service) DeleteDeleteRequestByUserID(ctx context.Context, userID uuid.UUID) error {
	err := s.store.UserDeleteRequestDeleteByUserID(ctx, pgtype.UUID{Bytes: userID, Valid: true})
	if err != nil {
		return apperr.NewDBError(err, DomainDeleteRequest)
	}
	return nil
}
