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

type Service interface {
	CheckAvailability(ctx context.Context, email Email, username Username) (emailAvail bool, usernameAvail bool, err error)
	Get(ctx context.Context, id uuid.UUID) (*User, error)
	GetByEmail(ctx context.Context, email Email) (*User, error)
	GetByUsername(ctx context.Context, username Username) (*User, error)
	SetPreferredPresence(ctx context.Context, id uuid.UUID, p *presence.Presence) (*User, error)
	UpdateProfile(ctx context.Context, p UpdateProfileParams) (*User, error)
	VerifyAccount(ctx context.Context, id uuid.UUID) error
}

type service struct {
	repo Repository
}

func NewService(repo Repository) *service {
	return &service{
		repo: repo,
	}
}

// Get retrieves a user aggregate by its unique UUID.
func (s *service) Get(ctx context.Context, id uuid.UUID) (*User, error) {
	if id == uuid.Nil {
		return nil, apperr.NewInvalidArgument(errors.New("user ID cannot be empty"), apperr.WithMsg("Invalid user ID"))
	}

	u, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	return u, nil
}

// GetByEmail retrieves a user aggregate by a raw email string.
func (s *service) GetByEmail(ctx context.Context, rawEmail string) (*User, error) {
	email, err := NewEmail(rawEmail)
	if err != nil || !email.IsValid() {
		return nil, apperr.NewInvalidArgument(
			errors.New("invalid email address"),
			apperr.WithMsg("Invalid email address"),
		)
	}

	u, err := s.repo.GetByEmail(ctx, email)
	if err != nil {
		return nil, err
	}

	return u, nil
}

// GetByUsername retrieves a user aggregate by its value object Username.
func (s *service) GetByUsername(ctx context.Context, username Username) (*User, error) {
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
func (s *service) UpdateProfile(ctx context.Context, p UpdateProfileParams) (*User, error) {
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
func (s *service) SetPreferredPresence(ctx context.Context, id uuid.UUID, p *presence.Presence) (*User, error) {
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
func (s *service) VerifyAccount(ctx context.Context, id uuid.UUID) error {
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
func (s *service) CheckAvailability(ctx context.Context, email Email, username Username) (emailAvail, usernameAvail bool, err error) {
	if !email.IsValid() {
		return false, false, apperr.NewInvalidArgument(errors.New("invalid email address"), apperr.WithMsg("Invalid email address"))
	}
	if !username.IsValid() {
		return false, false, apperr.NewInvalidArgument(errors.New("invalid username"), apperr.WithMsg("Invalid username"))
	}

	return s.repo.CheckAvailability(ctx, email, username)
}
