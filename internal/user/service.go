package user

import (
	"context"
	"time"

	"bonfire-api/internal/crypto"
	"bonfire-api/internal/presence"

	"github.com/google/uuid"
)

type Service struct {
	cache      Cache
	repo       Repository
	cachedRepo CachedRepository
	outboxRepo OutboxRepository
	tx         TX
}

func NewService(
	cache Cache,
	repo Repository,
	cachedRepo CachedRepository,
	outboxRepo OutboxRepository,
	tx TX,
) *Service {
	return &Service{
		cache:      cache,
		repo:       repo,
		cachedRepo: cachedRepo,
		outboxRepo: outboxRepo,
		tx:         tx,
	}
}

func (s *Service) Get(ctx context.Context, userID uuid.UUID) (*User, error) {
	u, err := s.cachedRepo.Get(ctx, userID)
	if err != nil {
		return nil, err
	}

	if err := u.EnsureActive(); err != nil {
		_ = s.cache.Delete(ctx, userID)
		return nil, err
	}

	return u, nil
}

type UpdateEmailParams struct {
	UserID   uuid.UUID
	NewEmail string
	Password string
}

func (s *Service) UpdateEmail(ctx context.Context, p UpdateEmailParams) (*User, error) {
	u, err := s.fetchAndAuthenticate(ctx, p.UserID, p.Password)
	if err != nil {
		return nil, err
	}

	if u.Email == p.NewEmail {
		return u, nil
	}

	now := time.Now()

	updatedUser, err := s.repo.UpdateEmail(ctx, p.UserID, p.NewEmail, now)
	if err != nil {
		return nil, err
	}

	_ = s.cache.Delete(ctx, updatedUser.ID)

	return updatedUser, nil
}

type UpdateUsernameParams struct {
	UserID      uuid.UUID
	NewUsername string
	Password    string
}

func (s *Service) UpdateUsername(ctx context.Context, p UpdateUsernameParams) (*User, error) {
	u, err := s.fetchAndAuthenticate(ctx, p.UserID, p.Password)
	if err != nil {
		return nil, err
	}

	if u.Username == p.NewUsername {
		return u, nil
	}

	var updatedUser *User
	now := time.Now()

	err = s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		var txErr error
		updatedUser, txErr = s.repo.UpdateUsername(txCtx, u.ID, p.NewUsername, now)
		if txErr != nil {
			return txErr
		}

		payload := EventUpdateUsernamePayload{
			UserID:    updatedUser.ID,
			Username:  updatedUser.Username,
			UpdatedAt: updatedUser.UpdatedAt,
		}

		return s.outboxRepo.Publish(txCtx, EventUpdateUsername, payload, now)
	})
	if err != nil {
		return nil, err
	}

	_ = s.cache.Delete(ctx, updatedUser.ID)

	return updatedUser, nil
}

type UpdatePasswordParams struct {
	UserID             uuid.UUID
	CurrentPassword    string
	NewPassword        string
	NewPasswordConfirm string
}

func (s *Service) UpdatePassword(ctx context.Context, p UpdatePasswordParams) error {
	if p.NewPassword != p.NewPasswordConfirm {
		return ErrPasswordMismatch()
	}

	u, err := s.fetchAndAuthenticate(ctx, p.UserID, p.CurrentPassword)
	if err != nil {
		return err
	}

	newPasswordHash, err := crypto.HashPassword(p.NewPassword)
	if err != nil {
		return ErrPasswordHashFailed().Wrap(err)
	}

	now := time.Now()

	_, err = s.repo.UpdatePasswordHash(ctx, u.ID, newPasswordHash, now)
	if err != nil {
		return err
	}

	_ = s.cache.Delete(ctx, u.ID)

	return nil
}

type UpdatePreferredPresenceParams struct {
	UserID   uuid.UUID
	Presence *presence.Presence
	Duration *PreferredPresenceDuration
}

func (s *Service) UpdatePreferredPresence(ctx context.Context, p UpdatePreferredPresenceParams) (*User, error) {
	validateNow := time.Now()

	until, err := p.Duration.CalculateUntil(validateNow)
	if err != nil {
		return nil, err
	}

	u, err := s.Get(ctx, p.UserID)
	if err != nil {
		return nil, err
	}

	if u.PreferredPresence == p.Presence && u.PreferredPresenceUntil == until {
		return u, nil
	}

	var updatedUser *User
	now := time.Now()

	err = s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		var txErr error
		updatedUser, txErr = s.repo.UpdatePresence(txCtx, u.ID, p.Presence, until, now)
		if txErr != nil {
			return txErr
		}

		eff := presence.PresenceOnline
		if p := updatedUser.EffectivePresence(now); p != nil && p.IsValid() {
			eff = *p
		}

		payload := EventUpdatePresencePayload{
			UserID:    updatedUser.ID,
			Presence:  eff,
			UpdatedAt: updatedUser.UpdatedAt,
		}

		return s.outboxRepo.Publish(txCtx, EventUpdatePresence, payload, now)
	})
	if err != nil {
		return nil, err
	}

	_ = s.cache.Delete(ctx, updatedUser.ID)

	return updatedUser, nil
}

type UpdateProfileParams struct {
	UserID      uuid.UUID
	DisplayName string
	Bio         *string
	AvatarURL   *string
	BannerColor *string
}

func (s *Service) UpdateProfile(ctx context.Context, p UpdateProfileParams) (*User, error) {
	u, err := s.Get(ctx, p.UserID)
	if err != nil {
		return nil, err
	}

	if u.DisplayName == p.DisplayName &&
		u.Bio == p.Bio &&
		u.AvatarURL == p.AvatarURL &&
		u.BannerColor == p.BannerColor {
		return u, nil
	}

	var updatedUser *User
	now := time.Now()

	err = s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		var txErr error
		updatedUser, txErr = s.repo.UpdateProfile(txCtx, u.ID, p.DisplayName, p.Bio, p.AvatarURL, p.BannerColor, now)
		if txErr != nil {
			return txErr
		}

		payload := EventUpdateProfilePayload{
			UserID:      updatedUser.ID,
			DisplayName: updatedUser.DisplayName,
			Bio:         updatedUser.Bio,
			AvatarURL:   updatedUser.AvatarURL,
			BannerColor: updatedUser.BannerColor,
			UpdatedAt:   updatedUser.UpdatedAt,
		}

		return s.outboxRepo.Publish(txCtx, EventUpdateProfile, payload, now)
	})
	if err != nil {
		return nil, err
	}

	_ = s.cache.Delete(ctx, updatedUser.ID)

	return updatedUser, nil
}

type DisableParams struct {
	UserID   uuid.UUID
	Password string
}

func (s *Service) Disable(ctx context.Context, p DisableParams) error {
	u, err := s.fetchAndAuthenticate(ctx, p.UserID, p.Password)
	if err != nil {
		return err
	}
	if u.IsDisabled() {
		return nil
	}

	now := time.Now()

	err = s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		updatedUser, err := s.repo.SetDisabled(txCtx, u.ID, &now, now)
		if err != nil {
			return err
		}

		payload := EventDisablePayload{
			UserID:    updatedUser.ID,
			UpdatedAt: updatedUser.UpdatedAt,
		}

		return s.outboxRepo.Publish(txCtx, EventDisable, payload, now)
	})
	if err != nil {
		return err
	}

	_ = s.cache.Delete(ctx, u.ID)

	return nil
}

type ScheduleDeleteParams struct {
	UserID   uuid.UUID
	Password string
}

func (s *Service) ScheduleDelete(ctx context.Context, p ScheduleDeleteParams) error {
	u, err := s.fetchAndAuthenticate(ctx, p.UserID, p.Password)
	if err != nil {
		return err
	}

	if u.IsScheduledForDeletion() {
		return nil
	}

	now := time.Now()
	scheduledAt := now.Add(ScheduleDeleteGracePeriod)

	err = s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		updatedUser, err := s.repo.SetDeleteSchedule(txCtx, u.ID, &scheduledAt, &now, now)
		if err != nil {
			return err
		}

		payload := EventDisablePayload{
			UserID:    updatedUser.ID,
			UpdatedAt: updatedUser.UpdatedAt,
		}

		return s.outboxRepo.Publish(txCtx, EventDisable, payload, now)
	})
	if err != nil {
		return err
	}

	_ = s.cache.Delete(ctx, u.ID)

	return nil
}

func (s *Service) AnonymizeBatch(ctx context.Context) error {
	now := time.Now()

	users, err := s.repo.ListDeleteScheduled(ctx, now, AnonymizeBatchSize)
	if err != nil {
		return err
	}

	if len(users) == 0 {
		return nil
	}

	for _, u := range users {
		u.Anonymize(now)
	}

	_, err = s.repo.UpdateBatch(ctx, users)
	if err != nil {
		return err
	}

	invalidIDs := make([]uuid.UUID, len(users))
	for i, u := range users {
		invalidIDs[i] = u.ID
	}
	_ = s.cache.DeleteBatch(ctx, invalidIDs)

	return nil
}

func (s *Service) fetchAndAuthenticate(ctx context.Context, actorID uuid.UUID, password string) (*User, error) {
	u, err := s.fetchValid(ctx, actorID)
	if err != nil {
		return nil, err
	}

	if err := crypto.ComparePassword(u.PasswordHash, password); err != nil {
		return nil, ErrInvalidPassword().Wrap(err)
	}

	return u, nil
}

func (s *Service) fetchValid(ctx context.Context, actorID uuid.UUID) (*User, error) {
	u, err := s.repo.Get(ctx, actorID)
	if err != nil {
		return nil, err
	}

	if err := u.EnsureActive(); err != nil {
		return nil, err
	}

	return u, nil
}

func (s *Service) fetchBatchValid(ctx context.Context, userIDs []uuid.UUID) (map[uuid.UUID]*User, error) {
	if len(userIDs) == 0 {
		return make(map[uuid.UUID]*User), nil
	}

	usersMap, err := s.repo.GetBatch(ctx, userIDs)
	if err != nil {
		return nil, err
	}

	validUsers := make(map[uuid.UUID]*User, len(usersMap))
	for id, u := range usersMap {
		if u == nil || u.EnsureActive() != nil {
			continue
		}
		validUsers[id] = u
	}

	return validUsers, nil
}
