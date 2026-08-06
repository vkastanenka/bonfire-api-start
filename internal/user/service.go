package user

import (
	"context"

	"bonfire-api/internal/errs"
	"bonfire-api/internal/presence"

	"github.com/google/uuid"
)

type Cache interface {
	DeleteAggregate(ctx context.Context, id uuid.UUID) error
	GetAggregate(ctx context.Context, id uuid.UUID) (*User, error)
	SetAggregate(ctx context.Context, u *User) error
}

type Repository interface {
	Create(ctx context.Context, u *User) error
	GetAggregate(ctx context.Context, id uuid.UUID) (*User, error)
	GetByEmail(ctx context.Context, email Email) (*User, error)
	GetByUsername(ctx context.Context, username Username) (*User, error)
	CheckAvailability(ctx context.Context, email Email, username Username) (bool, bool, error)
	Update(ctx context.Context, u *User) error
	UpsertProfile(ctx context.Context, userID uuid.UUID, prof *Profile) error
}

type Service struct {
	cache Cache
	repo  Repository
}

func NewService(cache Cache, repo Repository) *Service {
	return &Service{
		cache: cache,
		repo:  repo,
	}
}

func (s *Service) Get(ctx context.Context, id uuid.UUID) (*User, error) {
	if id == uuid.Nil {
		return nil, errs.InvalidArgument("user ID cannot be empty")
	}

	if s.cache != nil {
		u, err := s.cache.GetAggregate(ctx, id)
		if err == nil && u != nil {
			return u, nil
		}
		// Log cache write error without failing the HTTP request
	}

	u, err := s.repo.GetAggregate(ctx, id)
	if err != nil {
		return nil, err
	}

	if s.cache != nil {
		if cacheErr := s.cache.SetAggregate(ctx, u); cacheErr != nil {
			// Log cache write error without failing the HTTP request
		}
	}

	return u, nil
}

func (s *Service) GetByEmail(ctx context.Context, email Email) (*User, error) {
	// 0. Validate

	// 1. Query DB directly (B-Tree indexed lookup is ~1ms)
	u, err := s.repo.GetByEmail(ctx, email)
	if err != nil {
		return nil, err
	}

	// 2. Warm the primary User Aggregate cache (user:<id>:aggregate)
	// This ensures the upcoming GET /users/@me call hits Redis directly!
	if s.cache != nil {
		if cacheErr := s.cache.SetAggregate(ctx, u); cacheErr != nil {
			// Log cache write error without failing login
		}
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
		return nil, errs.InvalidArgument("user ID cannot be empty")
	}

	u, err := s.repo.GetAggregate(ctx, p.UserID)
	if err != nil {
		return nil, err
	}

	u.UpdateProfile(p.DisplayName, p.AvatarURL)

	prof := u.Profile()
	if err := s.repo.UpsertProfile(ctx, u.ID(), &prof); err != nil {
		return nil, err
	}

	// Update cache directly with the mutated aggregate
	if s.cache != nil {
		if cacheErr := s.cache.SetAggregate(ctx, u); cacheErr != nil {
			// Fail open: log cache write failure, don't break the user request
		}
	}

	return u, nil
}

func (s *Service) SetPreferredPresence(ctx context.Context, id uuid.UUID, presence *presence.Presence) (*User, error) {
	if id == uuid.Nil {
		return nil, errs.InvalidArgument("user ID cannot be empty")
	}

	u, err := s.repo.GetAggregate(ctx, id)
	if err != nil {
		return nil, err
	}

	if err := u.SetPreferredPresence(presence); err != nil {
		return nil, errs.InvalidArgument("invalid presence transition").Wrap(err)
	}

	if err := s.repo.Update(ctx, u); err != nil {
		return nil, errs.Internal("failed to save user presence").Wrap(err)
	}

	// Sync the updated user aggregate to Redis
	if s.cache != nil {
		if cacheErr := s.cache.SetAggregate(ctx, u); cacheErr != nil {
			// Fail open: log error
		}
	}

	return u, nil
}

func (s *Service) CheckAvailability(ctx context.Context, email Email, username Username) (emailAvail, usernameAvail bool, err error) {
	return s.repo.CheckAvailability(ctx, email, username)
}
