package user_profile

import (
	"bonfire-api/internal/apperr"
	"bonfire-api/internal/repository"
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
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

func (s *Service) GetByUserID(ctx context.Context, userID uuid.UUID) (repository.UserProfile, error) {
	var pgUserID pgtype.UUID
	pgUserID.Bytes = userID
	pgUserID.Valid = true

	user, err := s.store.UserProfileGet(ctx, pgUserID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return user, apperr.New(apperr.CodeNotFound, "User profile not found.")
		}
		return user, apperr.New(apperr.CodeInternal, "Database error", apperr.WithErr(err))
	}

	return user, nil
}
