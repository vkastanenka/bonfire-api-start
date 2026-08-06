package user

import (
	"context"

	"github.com/google/uuid"
)

type Cache interface {
	GetAggregate(ctx context.Context, userID uuid.UUID) (*Aggregate, error)
	GetAggregateBatch(ctx context.Context, userIDs []uuid.UUID) (map[uuid.UUID]*Aggregate, []uuid.UUID, error)
	SetAggregate(ctx context.Context, agg *Aggregate) error
	SetAggregateBatch(ctx context.Context, aggs []*Aggregate) error
}

type Repository interface {
	Create(ctx context.Context, agg *Aggregate) error
	GetAggregate(ctx context.Context, id uuid.UUID) (*Aggregate, error)
	GetAggregateBatch(ctx context.Context, ids []uuid.UUID) ([]*Aggregate, error)
	GetByEmail(ctx context.Context, email Email) (*Aggregate, error)
	Update(ctx context.Context, u *User) error
	UpdateProfile(ctx context.Context, userID uuid.UUID, prof *Profile) error
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

func (s *Service) Get(ctx context.Context, rawID uuid.UUID) (*Aggregate, error) {
	id, err := NewID(rawID)
	if err != nil {
		return nil, err
	}

	u, err := s.cache.GetAggregate(ctx, id.UUID())
	if err == nil && u != nil {
		return u, nil
	}
	// Log error

	u, err = s.repo.GetAggregate(ctx, id.UUID())
	if err != nil {
		return nil, err
	}

	if cacheErr := s.cache.SetAggregate(ctx, u); cacheErr != nil {
		// Log error
	}

	return u, nil
}

// UpdateUsername
// UpdateEmail
// UpdatePhone
// UpdatePassword
// UpdatePreferredPreference
// Disable
// ScheduleDelete
// Anonymize

type UpdateProfileParams struct {
	UserID      uuid.UUID
	DisplayName string
	Bio         *string
	AvatarURL   *string
	BannerColor *string
}

func (s *Service) UpdateProfile(ctx context.Context, p UpdateProfileParams) (*Aggregate, error) {
	id, err := NewID(p.UserID)
	if err != nil {
		return nil, err
	}

	displayName, err := NewDisplayName(p.DisplayName)
	if err != nil {
		return nil, err
	}

	bio, err := NewBio(p.Bio)
	if err != nil {
		return nil, err
	}

	avatarURL, err := NewURL(p.AvatarURL)
	if err != nil {
		return nil, err
	}

	bannerColor, err := NewHexCode(p.BannerColor)
	if err != nil {
		return nil, err
	}

	agg, err := s.repo.GetAggregate(ctx, id.UUID())
	if err != nil {
		return nil, err
	}

	agg.Profile().Update(displayName, &bio, &avatarURL, &bannerColor)

	if err := s.repo.UpdateProfile(ctx, id.UUID(), agg.Profile()); err != nil {
		return nil, err
	}

	if s.cache != nil {
		if cacheErr := s.cache.SetAggregate(ctx, agg); cacheErr != nil {
			// Log error
		}
	}

	return agg, nil
}
