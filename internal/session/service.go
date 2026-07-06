package session

import (
	"context"
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
	ID           *uuid.UUID
	UserID       uuid.UUID
	RefreshToken string
	ExpiresAt    time.Time
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
		ID:           pgtype.UUID{Bytes: targetID, Valid: true},
		UserID:       pgtype.UUID{Bytes: p.UserID, Valid: true},
		RefreshToken: p.RefreshToken,
		ExpiresAt:    pgtype.Timestamptz{Time: p.ExpiresAt, Valid: true},
	})
	if err != nil {
		return View{}, repository.NewError(err, repository.ScopeSession)
	}

	return NewView(row), nil
}
