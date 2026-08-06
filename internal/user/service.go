package user

import (
	"bonfire-api/internal/crypto"
	"bonfire-api/internal/errs"
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

type Cache interface {
	Delete(ctx context.Context, id ID) error
	DeleteBatch(ctx context.Context, ids []ID) error
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

type UpdateUsernameParams struct {
	UserID      uuid.UUID
	NewUsername string
	Password    string
}

func (s *Service) UpdateUsername(ctx context.Context, p UpdateUsernameParams) (*User, error) {
	id, err := NewID(p.UserID)
	if err != nil {
		return nil, err
	}

	newUsername, err := NewUsername(p.NewUsername)
	if err != nil {
		return nil, err
	}

	password, err := NewPassword(p.Password)
	if err != nil {
		return nil, err
	}

	u, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	if err = crypto.ComparePassword(u.PasswordHash().String(), password.String()); err != nil {
		return nil, errs.Unauthenticated("Invalid password.").
			FieldViolation("password", "Invalid password.", "INVALID_CREDENTIALS").
			Wrap(errors.New("invalid credentials"))
	}

	u.UpdateUsername(newUsername)

	uu, err := s.repo.Update(ctx, u)
	if err != nil {
		return nil, err
	}

	if cacheErr := s.cache.Delete(ctx, id); cacheErr != nil {
		// Log err
	}

	return uu, nil
}

type UpdateEmailParams struct {
	UserID   uuid.UUID
	NewEmail string
	Password string
}

func (s *Service) UpdateEmail(ctx context.Context, p UpdateEmailParams) (*User, error) {
	id, err := NewID(p.UserID)
	if err != nil {
		return nil, err
	}

	newEmail, err := NewEmail(p.NewEmail)
	if err != nil {
		return nil, err
	}

	password, err := NewPassword(p.Password)
	if err != nil {
		return nil, err
	}

	u, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	if err = crypto.ComparePassword(u.PasswordHash().String(), password.String()); err != nil {
		return nil, errs.Unauthenticated("Invalid password.").
			FieldViolation("password", "Invalid password.", "INVALID_CREDENTIALS").
			Wrap(errors.New("invalid credentials"))
	}

	u.UpdateEmail(newEmail)

	uu, err := s.repo.Update(ctx, u)
	if err != nil {
		return nil, err
	}

	if cacheErr := s.cache.Delete(ctx, id); cacheErr != nil {
		// Log err
	}

	return uu, nil
}

type UpdatePasswordParams struct {
	UserID             uuid.UUID
	CurrentPassword    string
	NewPassword        string
	NewPasswordConfirm string
}

func (s *Service) UpdatePassword(ctx context.Context, p UpdatePasswordParams) error {
	id, err := NewID(p.UserID)
	if err != nil {
		return err
	}

	currentPassword, err := NewPassword(p.CurrentPassword)
	if err != nil {
		return err
	}

	newPassword, err := NewPassword(p.NewPassword)
	if err != nil {
		return err
	}

	newPasswordConfirm, err := NewPassword(p.NewPasswordConfirm)
	if err != nil {
		return err
	}

	if !newPassword.Equals(newPasswordConfirm) {
		return errs.InvalidArgument("Passwords must match.")
	}

	u, err := s.repo.Get(ctx, id)
	if err != nil {
		return err
	}

	if err = crypto.ComparePassword(u.PasswordHash().String(), currentPassword.String()); err != nil {
		return errs.Unauthenticated("Invalid password.").
			FieldViolation("password", "Invalid password.", "INVALID_CREDENTIALS").
			Wrap(errors.New("invalid credentials"))
	}

	passwordHash, err := crypto.HashPassword(newPassword.String())
	if err != nil {
		return errs.Internal("failed to hash password").Wrap(err)
	}

	newPasswordHash, err := NewPassword(passwordHash)
	if err != nil {
		return err
	}

	u.UpdatePassword(newPasswordHash)

	_, err = s.repo.Update(ctx, u)
	if err != nil {
		return err
	}

	if cacheErr := s.cache.Delete(ctx, id); cacheErr != nil {
		// Log err
	}

	return nil
}

func (s *Service) UpdatePhone(ctx context.Context) error {
	return nil
}

type UpdatePreferredPresenceParams struct {
	UserID   uuid.UUID
	Presence *string
	Until    *string
}

func (s *Service) UpdatePreferredPreference(ctx context.Context, p UpdatePreferredPresenceParams) (*User, error) {
	id, err := NewID(p.UserID)
	if err != nil {
		return nil, err
	}

	preferredPresence, err := NewPreferredPresence(p.Presence)
	if err != nil {
		return nil, err
	}

	until, err := NewTimestamp(p.Until)
	if err != nil {
		return nil, err
	}

	u, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	u.SetPreferredPresence(preferredPresence, until)

	uu, err := s.repo.Update(ctx, u)
	if err != nil {
		return nil, err
	}

	if cacheErr := s.cache.Delete(ctx, id); cacheErr != nil {
		// Log err
	}

	return uu, nil
}

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

	bannerColor, err := NewHexColor(p.BannerColor)
	if err != nil {
		return nil, err
	}

	u, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	u.UpdateProfile(displayName, bio, avatarURL, bannerColor)

	uu, err := s.repo.Update(ctx, u)
	if err != nil {
		return nil, err
	}

	if cacheErr := s.cache.Delete(ctx, id); cacheErr != nil {
		// Log err
	}

	return uu, nil
}

func (s *Service) Disable(ctx context.Context, rawID uuid.UUID) error {
	id, err := NewID(rawID)
	if err != nil {
		return err
	}

	u, err := s.repo.Get(ctx, id)
	if err != nil {
		return err
	}

	u.Disable()

	_, err = s.repo.Update(ctx, u)
	if err != nil {
		return err
	}

	if cacheErr := s.cache.Delete(ctx, id); cacheErr != nil {
		// Log err
	}

	return nil
}

func (s *Service) ScheduleDelete(ctx context.Context, rawID uuid.UUID, rawDeleteAt time.Time) error {
	id, err := NewID(rawID)
	if err != nil {
		return err
	}

	deleteAt := NewTimestampFromTime(&rawDeleteAt)

	u, err := s.repo.Get(ctx, id)
	if err != nil {
		return err
	}

	u.ScheduleDelete(*deleteAt.Time())

	_, err = s.repo.Update(ctx, u)
	if err != nil {
		return err
	}

	if cacheErr := s.cache.Delete(ctx, id); cacheErr != nil {
		// Log err
	}

	return nil
}

func (s *Service) RequestUpdateEmailCode(ctx context.Context) error {
	// TODO: Outbox verification email
	return nil
}

func (s *Service) VerifyUpdateEmailCode(ctx context.Context) error {
	return nil
}
