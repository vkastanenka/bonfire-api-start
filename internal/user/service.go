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

// type CreateUserParams struct {
// 	Email       Email
// 	Username    Username
// 	Password    Password // Raw password value object
// 	DisplayName ProfileDisplayName
// }

// // Create handles initial user creation and persistence.
// func (s *Service) Create(ctx context.Context, p CreateUserParams, passwordHasher PasswordHasher) (*User, error) {
// 	// 1. Check uniqueness/availability first
// 	emailAvail, usernameAvail, err := s.repo.CheckAvailability(ctx, p.Email, p.Username)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to check availability: %w", err)
// 	}

// 	if !emailAvail {
// 		return nil, apperr.NewAlreadyExists(errors.New("email already taken"), apperr.WithMsg("Email is already registered"))
// 	}
// 	if !usernameAvail {
// 		return nil, apperr.NewAlreadyExists(errors.New("username already taken"), apperr.WithMsg("Username is already taken"))
// 	}

// 	// 2. Hash the raw password (via an injected hasher interface)
// 	hashedPassword, err := passwordHasher.Hash(p.Password.String())
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to hash password: %w", err)
// 	}

// 	// 3. Instantiate domain aggregate
// 	u, err := NewUser(p.Email, p.Username, hashedPassword, p.DisplayName)
// 	if err != nil {
// 		return nil, err
// 	}

// 	// 4. Persist aggregate root
// 	if err := s.repo.Create(ctx, u); err != nil {
// 		return nil, fmt.Errorf("failed to persist user: %w", err)
// 	}

// 	return u, nil
// }

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

func (s *Service) GetByUsername(ctx context.Context, username Username) (*User, error) {
	if !username.IsValid() {
		return nil, apperr.NewInvalidArgument(errors.New("username cannot be empty"), apperr.WithMsg("Invalid username"))
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

func (s *Service) UpdateProfile(ctx context.Context, p UpdateProfileParams) (*User, error) {
	if p.UserID == uuid.Nil {
		return nil, apperr.NewInvalidArgument(errors.New("user ID cannot be empty"), apperr.WithMsg("Invalid user ID"))
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

func (s *Service) VerifyAccount(ctx context.Context, id uuid.UUID) error {
	if id == uuid.Nil {
		return apperr.NewInvalidArgument(errors.New("user ID cannot be empty"), apperr.WithMsg("Invalid user ID"))
	}

	u, err := s.repo.Get(ctx, id)
	if err != nil {
		return err
	}

	if u.IsVerified() {
		return nil // Idempotent: already verified
	}

	u.Verify(time.Now().UTC())

	if err := s.repo.Save(ctx, u); err != nil {
		return fmt.Errorf("failed to verify account: %w", err)
	}

	return nil
}

func (s *Service) CheckAvailability(ctx context.Context, email Email, username Username) (emailAvail, usernameAvail bool, err error) {
	return s.repo.CheckAvailability(ctx, email, username)
}
