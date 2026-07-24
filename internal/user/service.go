package user

import (
	"context"

	"bonfire-api/internal/errs"
	"bonfire-api/internal/presence"

	"github.com/google/uuid"
)

type Repository interface {
	Create(ctx context.Context, u *User) error
	Get(ctx context.Context, id uuid.UUID) (*User, error)
	GetByEmail(ctx context.Context, email Email) (*User, error)
	GetByUsername(ctx context.Context, username Username) (*User, error)
	CheckAvailability(ctx context.Context, email Email, username Username) (bool, bool, error)
	Update(ctx context.Context, u *User) error
	UpsertProfile(ctx context.Context, userID uuid.UUID, prof *Profile) error
}

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{
		repo: repo,
	}
}

func (s *Service) Get(ctx context.Context, id uuid.UUID) (*User, error) {
	if id == uuid.Nil {
		return nil, errs.InvalidArgument("user ID cannot be empty")
	}

	return s.repo.Get(ctx, id)
}

func (s *Service) GetByEmail(ctx context.Context, email Email) (*User, error) {
	return s.repo.GetByEmail(ctx, email)
}

func (s *Service) GetByUsername(ctx context.Context, username Username) (*User, error) {
	return s.repo.GetByUsername(ctx, username)
}

type UpdateProfileParams struct {
	UserID      uuid.UUID
	DisplayName ProfileDisplayName
	AvatarURL   *string
}

func (s *Service) UpdateProfile(ctx context.Context, p UpdateProfileParams) (*User, error) {
	if p.UserID == uuid.Nil {
		return nil, errs.InvalidArgument("user ID cannot be empty")
	}

	u, err := s.repo.Get(ctx, p.UserID)
	if err != nil {
		return nil, err
	}

	u.UpdateProfile(p.DisplayName, p.AvatarURL)

	prof := u.Profile()
	if err := s.repo.UpsertProfile(ctx, u.ID(), &prof); err != nil {
		return nil, err
	}

	return u, nil
}

func (s *Service) SetPreferredPresence(ctx context.Context, id uuid.UUID, presence *presence.Presence) (*User, error) {
	if id == uuid.Nil {
		return nil, errs.InvalidArgument("user ID cannot be empty")
	}

	u, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	if err := u.SetPreferredPresence(presence); err != nil {
		return nil, errs.InvalidArgument("invalid presence transition").Wrap(err)
	}

	if err := s.repo.Update(ctx, u); err != nil {
		return nil, errs.Internal("failed to save user presence").Wrap(err)
	}

	return u, nil
}

func (s *Service) CheckAvailability(ctx context.Context, email Email, username Username) (emailAvail, usernameAvail bool, err error) {
	return s.repo.CheckAvailability(ctx, email, username)
}
