package session

import (
	"context"
	"net/netip"
	"time"

	"bonfire-api/internal/repository"

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

type CreateParams struct {
	ID               *uuid.UUID
	UserID           uuid.UUID
	RefreshTokenHash []byte
	ExpiresAt        time.Time
	ClientIP         netip.Addr
	UserAgent        string
}

func (s *Service) Create(ctx context.Context, p CreateParams) (View, error) {
	var targetID uuid.UUID

	if p.ID != nil {
		targetID = *p.ID
	} else {
		var err error
		targetID, err = uuid.NewV7()
		if err != nil {
			return View{}, repository.NewError(err, repository.ScopeSession)
		}
	}

	row, err := s.store.SessionCreate(ctx, repository.SessionCreateParams{
		ID:               pgtype.UUID{Bytes: targetID, Valid: true},
		UserID:           pgtype.UUID{Bytes: p.UserID, Valid: true},
		RefreshTokenHash: p.RefreshTokenHash,
		ExpiresAt:        pgtype.Timestamptz{Time: p.ExpiresAt, Valid: true},
		ClientIP:         p.ClientIP,
		UserAgent:        p.UserAgent,
	})
	if err != nil {
		return View{}, repository.NewError(err, repository.ScopeSession)
	}

	return NewView(row), nil
}

func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (View, error) {
	row, err := s.store.SessionGetByID(ctx, pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		return View{}, repository.NewError(err, repository.ScopeSession)
	}
	return NewView(row), nil
}

func (s *Service) GetAuthByID(ctx context.Context, id uuid.UUID) (AuthView, error) {
	row, err := s.store.SessionGetByID(ctx, pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		return AuthView{}, repository.NewError(err, repository.ScopeSession)
	}
	return NewAuthView(row), nil
}

type UpdateRefreshTokenParams struct {
	ID               uuid.UUID
	RefreshTokenHash []byte
	ExpiresAt        time.Time
}

func (s *Service) UpdateRefreshToken(ctx context.Context, p UpdateRefreshTokenParams) (AuthView, error) {
	row, err := s.store.SessionUpdateRefreshToken(ctx, repository.SessionUpdateRefreshTokenParams{
		ID:               pgtype.UUID{Bytes: p.ID, Valid: true},
		RefreshTokenHash: p.RefreshTokenHash,
		ExpiresAt:        pgtype.Timestamptz{Time: p.ExpiresAt, Valid: true},
	})
	if err != nil {
		return AuthView{}, repository.NewError(err, repository.ScopeSession)
	}
	return NewAuthView(row), nil
}

func (s *Service) UpdateLastSeen(ctx context.Context, id uuid.UUID) (View, error) {
	row, err := s.store.SessionUpdateLastSeen(ctx, pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		return View{}, repository.NewError(err, repository.ScopeSession)
	}
	return NewView(row), nil
}

func (s *Service) Revoke(ctx context.Context, id uuid.UUID) (View, error) {
	row, err := s.store.SessionUpdateRevoked(ctx, pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		return View{}, repository.NewError(err, repository.ScopeSession)
	}
	return NewView(row), nil
}

func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	err := s.store.SessionDelete(ctx, pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		return repository.NewError(err, repository.ScopeSession)
	}
	return nil
}

type DeleteAllExceptParams struct {
	UserID    uuid.UUID
	SessionID uuid.UUID
}

func (s *Service) DeleteAllExcept(ctx context.Context, p DeleteAllExceptParams) error {
	err := s.store.SessionDeleteAllExcept(ctx, repository.SessionDeleteAllExceptParams{
		UserID: pgtype.UUID{Bytes: p.UserID, Valid: true},
		ID:     pgtype.UUID{Bytes: p.SessionID, Valid: true},
	})
	if err != nil {
		return repository.NewError(err, repository.ScopeSession)
	}
	return nil
}
