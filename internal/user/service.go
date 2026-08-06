package user

import (
	"bonfire-api/internal/token"
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
)

type Cache interface {
	Get(ctx context.Context, id ID) (*User, error)
	GetBatch(ctx context.Context, ids []ID) (map[ID]*User, []ID, error)
	Set(ctx context.Context, user *User) error
	SetBatch(ctx context.Context, users []*User) error
}

type Repository interface {
	Create(ctx context.Context, u *User) error
	Get(ctx context.Context, id ID) (*User, error)
	GetByEmail(ctx context.Context, email Email) (*User, error)
	ListDeleteScheduled(ctx context.Context, currentTime time.Time, limit int32) ([]*User, error)
	Update(ctx context.Context, u *User) (*User, error)
	UpdateBatch(ctx context.Context, users []*User) ([]*User, error)
}

type TokenProvider interface {
	GeneratePair(uid, sid uuid.UUID) (token.Pair, error)
	GeneratePasswordReset(userID uuid.UUID) (string, time.Time, error)
	GenerateEmailVerify(userID uuid.UUID) (string, time.Time, error)
	VerifyPasswordReset(tokenStr string) (*token.Claims, error)
	VerifyEmailVerify(tokenStr string) (*token.Claims, error)
	VerifyRefresh(tokenStr string) (*token.Claims, error)
}

type Service struct {
	cache  Cache
	repo   Repository
	tokens TokenProvider
}

func NewService(cache Cache, repo Repository, tokens TokenProvider) *Service {
	return &Service{
		cache:  cache,
		repo:   repo,
		tokens: tokens,
	}
}

func (s *Service) Get(ctx context.Context, rawID uuid.UUID) (*User, error) {
	id, err := NewID(rawID)
	if err != nil {
		return nil, err
	}

	u, err := s.cache.Get(ctx, id)
	if err == nil && u != nil {
		return u, nil
	}
	if err != nil {
		// Log cache err
	}

	u, err = s.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	go func(userToCache *User) {
		asyncCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
		defer cancel()

		if cacheErr := s.cache.Set(asyncCtx, userToCache); cacheErr != nil {
			// Log cache err
		}
	}(u)

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

func (s *Service) UpdateUsername(ctx context.Context, p UpdateUsernameParams) (*User, error) {
	id, err := NewID(p.UserID)
	if err != nil {
		return nil, err
	}

	username, err := NewUsername(p.Username)
	if err != nil {
		return nil, err
	}

	// TODO: Check password validity

	agg, err := s.repo.UserGet(ctx, id.UUID())
	if err != nil {
		return nil, err
	}

	agg.User().UpdateUsername(username)

	if err := s.repo.Update(ctx, agg.User()); err != nil {
		return nil, err
	}

	if cacheErr := s.cache.Set(ctx, u); cacheErr != nil {
		slog.WarnContext(ctx, "failed to populate user cache",
			"user_id", id.String(),
			"error", cacheErr,
		)
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

func (s *Service) UpdatePreferredPreference(ctx context.Context, p UpdatePreferredPresenceParams) (*User, error) {
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

func (s *Service) UpdateProfile(ctx context.Context, p UpdateProfileParams) (*User, error) {
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
	// (Repo executes the UPDATE and returns the fully hydrated *User)
	agg, err := s.repo.UpdateProfile(ctx, id.UUID(), UpdateProfileRepoParams{
		DisplayName: displayName,
		Bio:         bio,
		AvatarURL:   avatarURL,
		BannerColor: bannerColor,
	})
	if err != nil {
		return nil, err
	}

	if cacheErr := s.cache.Set(ctx, u); cacheErr != nil {
		slog.WarnContext(ctx, "failed to populate user cache",
			"user_id", id.String(),
			"error", cacheErr,
		)
	}

	return agg, nil
}
