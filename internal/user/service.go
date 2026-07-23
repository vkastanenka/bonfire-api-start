package user

import (
	"context"
	"time"

	"bonfire-api/internal/errs"
	"bonfire-api/internal/presence"

	"github.com/google/uuid"
)

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

// GetByEmail retrieves a user aggregate by a raw email string.
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

// GetByUsername retrieves a user aggregate by its value object Username.
func (s *Service) GetByUsername(ctx context.Context, username Username) (*User, error) {
	if !username.IsValid() {
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
	DisplayName ProfileDisplayName
	AvatarURL   *string
}

// UpdateProfile mutates a user's profile and persists the updated profile record.
func (s *Service) UpdateProfile(ctx context.Context, p UpdateProfileParams) (*User, error) {
	if p.UserID == uuid.Nil {
		return nil, errs.InvalidArgument("user ID cannot be empty")
	}

	if !p.DisplayName.IsValid() {
		return nil, errs.InvalidArgument("invalid display name")
	}

	u, err := s.repo.Get(ctx, p.UserID)
	if err != nil {
		return nil, err
	}

	u.UpdateProfile(p.DisplayName, p.AvatarURL)

	if err := s.repo.SaveProfile(ctx, u); err != nil {
		return nil, errs.Internal("failed to save profile").Wrap(err)
	}

	return u, nil
}

// SetPreferredPresence mutates a user's default presence and updates the user record.
func (s *Service) SetPreferredPresence(ctx context.Context, id uuid.UUID, p *presence.Presence) (*User, error) {
	if id == uuid.Nil {
		return nil, errs.InvalidArgument("user ID cannot be empty")
	}

	u, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	if err := u.SetPreferredPresence(p); err != nil {
		return nil, errs.InvalidArgument("invalid presence value").Wrap(err)
	}

	if err := s.repo.Save(ctx, u); err != nil {
		return nil, errs.Internal("failed to save user presence").Wrap(err)
	}

	return u, nil
}

// VerifyAccount verifies a user's account idempotently and updates the user record.
func (s *Service) VerifyAccount(ctx context.Context, id uuid.UUID) error {
	if id == uuid.Nil {
		return errs.InvalidArgument("user ID cannot be empty")
	}

	u, err := s.repo.Get(ctx, id)
	if err != nil {
		return err
	}

	if u.IsVerified() {
		return nil // Idempotent
	}

	u.Verify(time.Now().UTC())

	if err := s.repo.Save(ctx, u); err != nil {
		return errs.Internal("failed to verify account").Wrap(err)
	}

	return nil
}

// CheckAvailability queries whether an email address and username are available for registration.
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
