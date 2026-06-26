package session

import (
	"context"
	"net/netip"
	"time"

	"bonfire-api/internal/apperr"
	"bonfire-api/internal/repository"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// --- Session Service ---

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

// --- Session Service Count ---

func (s *Service) Count(ctx context.Context) (int64, error) {
	count, err := s.store.SessionCount(ctx)
	if err != nil {
		return 0, apperr.NewDBError(err)
	}
	return count, nil
}

// ==========================================
// CREATE
// ==========================================

// --- Session Service Create ---

type CreateParams struct {
	ID           uuid.UUID
	UserID       uuid.UUID
	RefreshToken string
	UserAgent    string
	ClientIP     netip.Addr
	IsBlocked    bool
	ExpiresAt    time.Time
}

func (s *Service) Create(ctx context.Context, p CreateParams) (View, error) {
	row, err := s.store.SessionCreate(ctx, repository.SessionCreateParams{
		ID:           pgtype.UUID{Bytes: p.ID, Valid: true},
		UserID:       pgtype.UUID{Bytes: p.UserID, Valid: true},
		RefreshToken: p.RefreshToken,
		UserAgent:    p.UserAgent,
		ClientIP:     p.ClientIP,
		IsBlocked:    p.IsBlocked,
		ExpiresAt:    pgtype.Timestamptz{Time: p.ExpiresAt, Valid: true},
	})
	if err != nil {
		return View{}, apperr.NewDBError(err)
	}
	return NewView(row), nil
}

// ==========================================
// LIST
// ==========================================

type ListParams struct {
	UserID uuid.UUID
	Status string
}

func (s *Service) List(ctx context.Context, p ListParams) ([]View, error) {
	var rows []repository.Session
	var err error

	if p.UserID == uuid.Nil {
		return []View{}, nil
	}

	dbUUID := pgtype.UUID{Bytes: p.UserID, Valid: true}

	switch p.Status {
	case "active":
		rows, err = s.store.SessionListActiveByUserID(ctx, dbUUID)
	case "blocked":
		rows, err = s.store.SessionListBlockedByUserID(ctx, dbUUID)
	case "expired":
		rows, err = s.store.SessionListExpiredByUserID(ctx, dbUUID)
	default:
		rows, err = s.store.SessionListByUserID(ctx, dbUUID)
	}

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

func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (View, error) {
	row, err := s.store.SessionGetByID(ctx, pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		return View{}, apperr.NewDBError(err)
	}
	return NewView(row), nil
}

func (s *Service) GetByRefreshToken(ctx context.Context, refreshToken string) (View, error) {
	row, err := s.store.SessionGetByRefreshToken(ctx, refreshToken)
	if err != nil {
		return View{}, apperr.NewDBError(err)
	}
	return NewView(row), nil
}

// ==========================================
// UPDATE
// ==========================================

type UpdateRefreshTokenParams struct {
	ID           uuid.UUID
	RefreshToken string
	ExpiresAt    time.Time
}

func (s *Service) UpdateRefreshToken(ctx context.Context, p UpdateRefreshTokenParams) (View, error) {
	row, err := s.store.SessionUpdateRefreshToken(ctx, repository.SessionUpdateRefreshTokenParams{
		ID:           pgtype.UUID{Bytes: p.ID, Valid: true},
		RefreshToken: p.RefreshToken,
		ExpiresAt:    pgtype.Timestamptz{Time: p.ExpiresAt, Valid: true},
	})
	if err != nil {
		return View{}, apperr.NewDBError(err)
	}
	return NewView(row), nil
}

func (s *Service) UpdateLastSeen(ctx context.Context, id uuid.UUID) (View, error) {
	row, err := s.store.SessionUpdateLastSeen(ctx, pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		return View{}, apperr.NewDBError(err)
	}
	return NewView(row), nil
}

func (s *Service) MarkBlocked(ctx context.Context, id uuid.UUID) (View, error) {
	row, err := s.store.SessionMarkBlocked(ctx, pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		return View{}, apperr.NewDBError(err)
	}
	return NewView(row), nil
}

// ==========================================
// DELETE
// ==========================================

func (s *Service) Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	err := s.store.SessionDelete(ctx, repository.SessionDeleteParams{
		ID:     pgtype.UUID{Bytes: id, Valid: true},
		UserID: pgtype.UUID{Bytes: userID, Valid: true},
	})
	if err != nil {
		return apperr.NewDBError(err)
	}
	return nil
}

func (s *Service) DeleteAllExcept(ctx context.Context, userID uuid.UUID, exceptID uuid.UUID) error {
	err := s.store.SessionDeleteAllExcept(ctx, repository.SessionDeleteAllExceptParams{
		UserID: pgtype.UUID{Bytes: userID, Valid: true},
		ID:     pgtype.UUID{Bytes: exceptID, Valid: true},
	})
	if err != nil {
		return apperr.NewDBError(err)
	}
	return nil
}

func (s *Service) PurgeExpired(ctx context.Context) error {
	err := s.store.SessionPurgeExpired(ctx)
	if err != nil {
		return apperr.NewDBError(err)
	}
	return nil
}
