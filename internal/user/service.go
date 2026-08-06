package user

import (
	"context"

	"github.com/google/uuid"
)

type Cache interface {
	AggregateGet(ctx context.Context, userID uuid.UUID) (*Aggregate, error)
	AggregateGetBatch(ctx context.Context, userIDs []uuid.UUID) (map[uuid.UUID]*Aggregate, []uuid.UUID, error)
	AggregateSet(ctx context.Context, agg *Aggregate) error
	AggregateSetBatch(ctx context.Context, aggs []*Aggregate) error
}

type Repository interface {
	AggregateGet(ctx context.Context, id uuid.UUID) (*Aggregate, error)
	AggregateGetBatch(ctx context.Context, ids []uuid.UUID) ([]*Aggregate, error)
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

	u, err := s.cache.AggregateGet(ctx, id.UUID())
	if err == nil && u != nil {
		return u, nil
	}
	if err != nil {
		// Log cache miss or error
	}

	u, err = s.repo.AggregateGet(ctx, id.UUID())
	if err != nil {
		return nil, err
	}

	if cacheErr := s.cache.AggregateSet(ctx, u); cacheErr != nil {
		// Log cache write error
	}

	return u, nil
}

func (s *Service) RequestUpdateEmail(ctx context.Context) error {
	// TODO: Outbox verification email
	return nil
}

type UpdateUsernameParams struct {
	UserID   uuid.UUID
	Username string
	Password string
}

func (s *Service) UpdateUsername(ctx context.Context, p UpdateUsernameParams) (*Aggregate, error) {
	id, err := NewID(p.UserID)
	if err != nil {
		return nil, err
	}

	username, err := NewUsername(p.Username)
	if err != nil {
		return nil, err
	}

	// TODO: Check password validity

	agg, err := s.repo.AggregateGet(ctx, id.UUID())
	if err != nil {
		return nil, err
	}

	agg.User().UpdateUsername(username)

	if err := s.repo.Update(ctx, agg.User()); err != nil {
		return nil, err
	}

	if s.cache != nil {
		if cacheErr := s.cache.SetAggregate(ctx, agg); cacheErr != nil {
			// Log error
		}
	}

	return nil, nil
}

type UpdateEmailParams struct {
	UserID   uuid.UUID
	Email    string
	Password string
}

func (s *Service) UpdateEmail(ctx context.Context) error {
	return nil
}

func (s *Service) UpdatePhone(ctx context.Context) error {
	return nil
}

type UpdatePasswordParams struct {
	CurrentPassword    string
	NewPassword        string
	NewPasswordConfirm string
}

func (s *Service) UpdatePassword(ctx context.Context, p UpdatePasswordParams) error {
	return nil
}

type UpdatePreferredPresenceParams struct {
	Presence  *string
	ExpiresAt *string
}

func (s *Service) UpdatePreferredPreference(ctx context.Context, p UpdatePreferredPresenceParams) (*Aggregate, error) {
	preferredPresence, err := NewPreferredPresence(p.Presence)
	if err != nil {
		return nil, err
	}

	expiresAt, err := NewTimestamp(p.ExpiresAt)
	if err != nil {
		return nil, err
	}

	agg, err := s.repo.Update(ctx, id.UUID())
	if err != nil {
		return nil, err
	}

	return nil, nil
}

// Disable
// ScheduleDelete
// AnonymizeBatch

type UpdateProfileParams struct {
	UserID      uuid.UUID
	DisplayName string
	Bio         *string
	AvatarURL   *string
	BannerColor *string
}

func (s *Service) UpdateProfile(ctx context.Context, p UpdateProfileParams) (*Aggregate, error) {
	// 1. Validate inputs via value objects first (keeps bad data out of the db)
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

	// 2. Delegate the update directly to the repository
	// (Repo executes the UPDATE and returns the fully hydrated *Aggregate)
	agg, err := s.repo.UpdateProfile(ctx, id.UUID(), UpdateProfileRepoParams{
		DisplayName: displayName,
		Bio:         bio,
		AvatarURL:   avatarURL,
		BannerColor: bannerColor,
	})
	if err != nil {
		return nil, err
	}

	// 3. Sync the cache immediately with the returned aggregate
	if s.cache != nil {
		if cacheErr := s.cache.SetAggregate(ctx, agg); cacheErr != nil {
			// s.logger.WarnContext(ctx, "failed to update user cache after profile update", "error", cacheErr)
		}
	}

	return agg, nil
}
