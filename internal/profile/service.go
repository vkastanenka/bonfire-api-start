package profile

import (
	"context"

	"bonfire-api/internal/apperr"
	"bonfire-api/internal/repository"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// --- profile service ---

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

// --- profile service Count ---

func (s *Service) Count(ctx context.Context) (int64, error) {
	count, err := s.store.ProfileCount(ctx)
	if err != nil {
		return 0, apperr.NewDBError(err)
	}
	return count, nil
}

// ==========================================
// CREATE
// ==========================================

// --- profile service Create ---

type CreateParams struct {
	UserID      uuid.UUID
	DisplayName string
}

func (s *Service) Create(ctx context.Context, p CreateParams) (View, error) {
	row, err := s.store.ProfileCreate(ctx, repository.ProfileCreateParams{
		UserID:      pgtype.UUID{Bytes: p.UserID, Valid: true},
		DisplayName: p.DisplayName,
	})
	if err != nil {
		return View{}, apperr.NewDBError(err)
	}
	return NewView(row), nil
}

// ==========================================
// GET
// ==========================================

// --- profile service GetByUserID ---

func (s *Service) GetByUserID(ctx context.Context, userID uuid.UUID) (View, error) {
	row, err := s.store.ProfileGetByUserID(ctx, pgtype.UUID{Bytes: userID, Valid: true})
	if err != nil {
		return View{}, apperr.NewDBError(err)
	}
	return NewView(row), nil
}

// ==========================================
// UPDATE
// ==========================================

// --- profile service UpdateDisplayName ---

type UpdateDisplayNameParams struct {
	UserID      uuid.UUID
	DisplayName string
}

func (s *Service) UpdateDisplayName(ctx context.Context, p UpdateDisplayNameParams) (View, error) {
	row, err := s.store.ProfileUpdateDisplayName(ctx, repository.ProfileUpdateDisplayNameParams{
		UserID:      pgtype.UUID{Bytes: p.UserID, Valid: true},
		DisplayName: p.DisplayName,
	})
	if err != nil {
		return View{}, apperr.NewDBError(err)
	}
	return NewView(row), nil
}

// ==========================================
// DELETE
// ==========================================

// --- profile service DeleteByUserID ---

func (s *Service) DeleteByUserID(ctx context.Context, id uuid.UUID) error {
	err := s.store.ProfileDeleteByUserID(ctx, pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		return apperr.NewDBError(err)
	}
	return nil
}
