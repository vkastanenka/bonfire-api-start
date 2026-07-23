package user

import (
	"context"

	"bonfire-api/internal/errs"

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

	u, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	return u, nil
}

func (s *Service) GetByEmail(ctx context.Context, rawEmail string) (*User, error) {
	email, err := NewEmail(rawEmail)
	if err != nil || !email.IsValid() {
		return nil, errs.InvalidArgument("invalid email address")
	}

	u, err := s.repo.GetByEmail(ctx, email)
	if err != nil {
		return nil, err
	}

	return u, nil
}

func (s *Service) GetByUsername(ctx context.Context, rawUsername string) (*User, error) {
	username, err := NewUsername(rawUsername)
	if err != nil || !username.IsValid() {
		return nil, errs.InvalidArgument("invalid username")
	}

	u, err := s.repo.GetByUsername(ctx, username)
	if err != nil {
		return nil, err
	}

	return u, nil
}

type UpdateProfileParams struct {
	UserID      uuid.UUID
	DisplayName string
	AvatarURL   *string
}

func (s *Service) UpdateProfile(ctx context.Context, p UpdateProfileParams) (*User, error) {
	if p.UserID == uuid.Nil {
		return nil, errs.InvalidArgument("user ID cannot be empty")
	}

	displayName, err := NewProfileDisplayName(p.DisplayName)
	if err != nil || !displayName.IsValid() {
		return nil, errs.InvalidArgument("invalid display name")
	}

	u, err := s.repo.Get(ctx, p.UserID)
	if err != nil {
		return nil, err
	}

	u.UpdateProfile(displayName, p.AvatarURL)

	prof := u.Profile()
	if err := s.repo.UpsertProfile(ctx, u.ID(), &prof); err != nil {
		return nil, err
	}

	return u, nil
}

func (s *Service) SetPreferredPresence(ctx context.Context, id uuid.UUID, rawPresence string) (*User, error) {
	if id == uuid.Nil {
		return nil, errs.InvalidArgument("user ID cannot be empty")
	}

	p, err := NewPreferredPresence(rawPresence)
	if err != nil {
		return nil, errs.InvalidArgument("invalid preferred presence")
	}

	u, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	if err := u.SetPreferredPresence(p); err != nil {
		return nil, errs.InvalidArgument("invalid presence value").Wrap(err)
	}

	if err := s.repo.Update(ctx, u); err != nil {
		return nil, errs.Internal("failed to save user presence").Wrap(err)
	}

	return u, nil
}

func (s *Service) CheckAvailability(ctx context.Context, rawEmail, rawUsername string) (emailAvail, usernameAvail bool, err error) {
	email, err := NewEmail(rawEmail)
	if err != nil || !email.IsValid() {
		return false, false, errs.InvalidArgument("invalid email address")
	}

	username, err := NewUsername(rawUsername)
	if err != nil || !username.IsValid() {
		return false, false, errs.InvalidArgument("invalid username")
	}

	return s.repo.CheckAvailability(ctx, email, username)
}
