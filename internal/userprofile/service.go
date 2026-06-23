package userprofile

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
	count, err := s.store.UserProfileCount(ctx)
	if err != nil {
		return 0, apperr.NewDBError(err, Domain)
	}
	return count, nil
}

// ==========================================
// CREATE
// ==========================================

// Create
func (s *Service) Create(ctx context.Context, p CreateParams) (View, error) {
	row, err := s.store.UserProfileCreate(ctx, repository.UserProfileCreateParams{
		UserID:      pgtype.UUID{Bytes: p.UserID, Valid: true},
		DisplayName: p.DisplayName,
	})
	if err != nil {
		return View{}, apperr.NewDBError(err, Domain)
	}
	return NewView(row), nil
}

// ==========================================
// GET
// ==========================================

// GetByUserID
func (s *Service) GetByUserID(ctx context.Context, userID uuid.UUID) (View, error) {
	row, err := s.store.UserProfileGetByUserID(ctx, pgtype.UUID{Bytes: userID, Valid: true})
	if err != nil {
		return View{}, apperr.NewDBError(err, Domain)
	}
	return NewView(row), nil
}

// ==========================================
// UPDATE
// ==========================================

// UpdateDisplayName
func (s *Service) UpdateDisplayName(ctx context.Context, p UpdateDisplayNameParams) (View, error) {
	row, err := s.store.UserProfileUpdateDisplayName(ctx, repository.UserProfileUpdateDisplayNameParams{
		UserID:      pgtype.UUID{Bytes: p.UserID, Valid: true},
		DisplayName: p.DisplayName,
	})
	if err != nil {
		return View{}, apperr.NewDBError(err, Domain)
	}
	return NewView(row), nil
}

// ==========================================
// DELETE
// ==========================================

// DeleteByUserID
func (s *Service) DeleteByUserID(ctx context.Context, userID uuid.UUID) error {
	err := s.store.UserProfileDeleteByUserID(ctx, pgtype.UUID{Bytes: userID, Valid: true})
	if err != nil {
		return apperr.NewDBError(err, Domain)
	}
	return nil
}
