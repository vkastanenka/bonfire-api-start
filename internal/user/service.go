package user

import (
	"context"
	"errors"
	"fmt"
	"time"

	"bonfire-api/internal/apperr"
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

// Get retrieves a user aggregate by its unique UUID.
func (s *Service) Get(ctx context.Context, id uuid.UUID) (*User, error) {
	if id == uuid.Nil {
		return nil, apperr.NewInvalidArgument(errors.New("user ID cannot be empty"), apperr.WithMsg("Invalid user ID"))
	}

	u, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	return u, nil
}

// GetByEmail retrieves a user aggregate by its value object Email.
func (s *Service) GetByEmail(ctx context.Context, email Email) (*User, error) {
	if !email.IsValid() {
		return nil, apperr.NewInvalidArgument(errors.New("invalid email address"), apperr.WithMsg("Invalid email address"))
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
		return nil, apperr.NewInvalidArgument(errors.New("invalid username"), apperr.WithMsg("Invalid username"))
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
		return nil, apperr.NewInvalidArgument(errors.New("user ID cannot be empty"), apperr.WithMsg("Invalid user ID"))
	}

	if !p.DisplayName.IsValid() {
		return nil, apperr.NewInvalidArgument(errors.New("invalid display name"), apperr.WithMsg("Invalid display name"))
	}

	u, err := s.repo.Get(ctx, p.UserID)
	if err != nil {
		return nil, err
	}

	u.UpdateProfile(p.DisplayName, p.AvatarURL)

	if err := s.repo.SaveProfile(ctx, u); err != nil {
		return nil, fmt.Errorf("failed to save profile: %w", err)
	}

	return u, nil
}

// SetPreferredPresence mutates a user's default presence and updates the user record.
func (s *Service) SetPreferredPresence(ctx context.Context, id uuid.UUID, p *presence.Presence) (*User, error) {
	if id == uuid.Nil {
		return nil, apperr.NewInvalidArgument(errors.New("user ID cannot be empty"), apperr.WithMsg("Invalid user ID"))
	}

	u, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	if err := u.SetPreferredPresence(p); err != nil {
		return nil, apperr.NewInvalidArgument(err, apperr.WithMsg("Invalid presence value"))
	}

	if err := s.repo.Save(ctx, u); err != nil {
		return nil, fmt.Errorf("failed to save user presence: %w", err)
	}

	return u, nil
}

// VerifyAccount verifies a user's account idempotently and updates the user record.
func (s *Service) VerifyAccount(ctx context.Context, id uuid.UUID) error {
	if id == uuid.Nil {
		return apperr.NewInvalidArgument(errors.New("user ID cannot be empty"), apperr.WithMsg("Invalid user ID"))
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
		return fmt.Errorf("failed to verify account: %w", err)
	}

	return nil
}

// CheckAvailability queries whether an email address and username are available for registration.
func (s *Service) CheckAvailability(ctx context.Context, email Email, username Username) (emailAvail, usernameAvail bool, err error) {
	if !email.IsValid() {
		return false, false, apperr.NewInvalidArgument(errors.New("invalid email address"), apperr.WithMsg("Invalid email address"))
	}
	if !username.IsValid() {
		return false, false, apperr.NewInvalidArgument(errors.New("invalid username"), apperr.WithMsg("Invalid username"))
	}

	return s.repo.CheckAvailability(ctx, email, username)
}
