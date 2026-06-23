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

func (s *Service) Count(ctx context.Context) (int64, error) {
	count, err := s.store.UserSessionCount(ctx)
	if err != nil {
		return 0, apperr.NewDBError(err, Domain)
	}
	return count, nil
}

// ==========================================
// CREATE
// ==========================================

func (s *Service) Create(ctx context.Context, p CreateParams) (View, error) {
	row, err := s.store.UserSessionCreate(ctx, repository.UserSessionCreateParams{
		ID:           pgtype.UUID{Bytes: p.ID, Valid: true},
		UserID:       pgtype.UUID{Bytes: p.UserID, Valid: true},
		RefreshToken: p.RefreshToken,
		UserAgent:    p.UserAgent,
		ClientIP:     p.ClientIP,
		IsBlocked:    p.IsBlocked,
		ExpiresAt:    pgtype.Timestamptz{Time: p.ExpiresAt, Valid: true},
	})
	if err != nil {
		return View{}, apperr.NewDBError(err, Domain)
	}
	return NewView(row), nil
}

// ==========================================
// LIST
// ==========================================

func (s *Service) ListActiveByUserID(ctx context.Context, userID uuid.UUID) ([]View, error) {
	rows, err := s.store.UserSessionListActiveByUserID(ctx, pgtype.UUID{Bytes: userID, Valid: true})
	if err != nil {
		return nil, apperr.NewDBError(err, Domain)
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

func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (View, error) {
	row, err := s.store.UserSessionGetByID(ctx, pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		return View{}, apperr.NewDBError(err, Domain)
	}
	return NewView(row), nil
}

func (s *Service) GetByRefreshToken(ctx context.Context, refreshToken string) (View, error) {
	row, err := s.store.UserSessionGetByRefreshToken(ctx, refreshToken)
	if err != nil {
		return View{}, apperr.NewDBError(err, Domain)
	}
	return NewView(row), nil
}

// ==========================================
// UPDATE
// ==========================================

func (s *Service) UpdateRefreshToken(ctx context.Context, p UpdateRefreshTokenParams) (View, error) {
	row, err := s.store.UserSessionUpdateRefreshToken(ctx, repository.UserSessionUpdateRefreshTokenParams{
		ID:           pgtype.UUID{Bytes: p.ID, Valid: true},
		RefreshToken: p.RefreshToken,
		ExpiresAt:    pgtype.Timestamptz{Time: p.ExpiresAt, Valid: true},
	})
	if err != nil {
		return View{}, apperr.NewDBError(err, Domain)
	}
	return NewView(row), nil
}

func (s *Service) UpdateLastSeen(ctx context.Context, id uuid.UUID) (View, error) {
	row, err := s.store.UserSessionUpdateLastSeen(ctx, pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		return View{}, apperr.NewDBError(err, Domain)
	}
	return NewView(row), nil
}

func (s *Service) MarkBlocked(ctx context.Context, id uuid.UUID) (View, error) {
	row, err := s.store.UserSessionMarkBlocked(ctx, pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		return View{}, apperr.NewDBError(err, Domain)
	}
	return NewView(row), nil
}

// ==========================================
// DELETE
// ==========================================

func (s *Service) Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	err := s.store.UserSessionDelete(ctx, repository.UserSessionDeleteParams{
		ID:     pgtype.UUID{Bytes: id, Valid: true},
		UserID: pgtype.UUID{Bytes: userID, Valid: true},
	})
	if err != nil {
		return apperr.NewDBError(err, Domain)
	}
	return nil
}

func (s *Service) DeleteAllExcept(ctx context.Context, userID uuid.UUID, exceptID uuid.UUID) error {
	err := s.store.UserSessionDeleteAllExcept(ctx, repository.UserSessionDeleteAllExceptParams{
		UserID: pgtype.UUID{Bytes: userID, Valid: true},
		ID:     pgtype.UUID{Bytes: exceptID, Valid: true},
	})
	if err != nil {
		return apperr.NewDBError(err, Domain)
	}
	return nil
}

func (s *Service) PurgeExpired(ctx context.Context) error {
	err := s.store.UserSessionPurgeExpired(ctx)
	if err != nil {
		return apperr.NewDBError(err, Domain)
	}
	return nil
}
