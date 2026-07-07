package user

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

type CreateParams struct {
	ID       *uuid.UUID `json:"id,omitempty"`
	Email    string     `json:"email"`
	Username string     `json:"username"`
	Password string     `json:"password"`
}

func (s *Service) Create(ctx context.Context, p CreateParams) (View, error) {
	var targetID uuid.UUID

	if p.ID != nil {
		targetID = *p.ID
	} else {
		var err error
		targetID, err = uuid.NewV7()
		if err != nil {
			return View{}, apperr.NewInternal(err, "")
		}
	}

	row, err := s.store.UserCreate(ctx, repository.UserCreateParams{
		ID:           pgtype.UUID{Bytes: targetID, Valid: true},
		Email:        p.Email,
		Username:     p.Username,
		PasswordHash: p.Password,
	})
	if err != nil {
		return View{}, repository.NewError(err, repository.ScopeUser)
	}
	return NewView(row), nil
}

func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (View, error) {
	row, err := s.store.UserGetByID(ctx, pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		return View{}, repository.NewError(err, repository.ScopeUser)
	}
	return NewView(row), nil
}

func (s *Service) GetByEmail(ctx context.Context, email string) (View, error) {
	row, err := s.store.UserGetByEmail(ctx, email)
	if err != nil {
		return View{}, repository.NewError(err, repository.ScopeUser)
	}
	return NewView(row), nil
}

func (s *Service) GetByUsername(ctx context.Context, username string) (View, error) {
	row, err := s.store.UserGetByUsername(ctx, username)
	if err != nil {
		return View{}, repository.NewError(err, repository.ScopeUser)
	}
	return NewView(row), nil
}

func (s *Service) GetAuthByID(ctx context.Context, id uuid.UUID) (AuthView, error) {
	row, err := s.store.UserGetByID(ctx, pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		return AuthView{}, repository.NewError(err, repository.ScopeUser)
	}
	return NewAuthView(row), nil
}

func (s *Service) GetAuthByEmail(ctx context.Context, email string) (AuthView, error) {
	row, err := s.store.UserGetByEmail(ctx, email)
	if err != nil {
		return AuthView{}, repository.NewError(err, repository.ScopeUser)
	}
	return NewAuthView(row), nil
}

type CreateProfileParams struct {
	UserID      uuid.UUID
	DisplayName string
}

func (s *Service) CreateProfile(ctx context.Context, p CreateProfileParams) (ProfileView, error) {
	row, err := s.store.UserProfileCreate(ctx, repository.UserProfileCreateParams{
		UserID:      pgtype.UUID{Bytes: p.UserID, Valid: true},
		DisplayName: p.DisplayName,
	})
	if err != nil {
		return ProfileView{}, repository.NewError(err, repository.ScopeUserProfile)
	}
	return NewProfileView(row), nil
}
