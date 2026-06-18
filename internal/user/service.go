package user

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

type UserService struct {
	store Store
}

func NewUserService(store Store) *UserService {
	return &UserService{store: store}
}

func (s *UserService) GetUserByID(ctx context.Context, userID uuid.UUID) (repository.User, error) {
	var pgUserID pgtype.UUID
	pgUserID.Bytes = userID
	pgUserID.Valid = true

	user, err := s.store.UserGet(ctx, pgUserID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return user, apperr.New(apperr.CodeNotFound, "User not found")
		}
		return user, apperr.New(apperr.CodeInternal, "Database error", apperr.WithErr(err))
	}

	return user, nil
}

func (s *UserService) GetUserByEmail(ctx context.Context, email string) (repository.User, error) {
	user, err := s.store.UserGetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return user, apperr.New(apperr.CodeNotFound, "User with this email not found")
		}
		return user, apperr.New(apperr.CodeInternal, "Database error", apperr.WithErr(err))
	}
	return user, nil
}

func (s *UserService) DeleteUserByEmail(ctx context.Context, email string) (repository.User, error) {
	user, err := s.store.UserDeleteByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return user, apperr.New(apperr.CodeNotFound, "User not found, nothing to delete")
		}
		return user, apperr.New(apperr.CodeInternal, "Failed to delete user", apperr.WithErr(err))
	}
	return user, nil
}
