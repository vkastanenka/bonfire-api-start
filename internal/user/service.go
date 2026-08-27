package user

import (
	"context"

	"bonfire-api/internal/crypto"
	"bonfire-api/internal/fields"
	"bonfire-api/internal/pkg/ptr"

	"github.com/google/uuid"
)

type Service struct {
	repo       Repository
	cache      Cache
	outboxRepo OutboxRepository
	tx         TX
}

func NewService(
	repo Repository,
	cache Cache,
	outboxRepo OutboxRepository,
	tx TX,
) *Service {
	return &Service{
		repo:       repo,
		cache:      cache,
		outboxRepo: outboxRepo,
		tx:         tx,
	}
}

func (s *Service) Get(ctx context.Context, userID uuid.UUID) (*User, error) {
	id, err := fields.ParseRequiredID("id", userID)
	if err != nil {
		return nil, err
	}

	user, err := s.fetchValid(ctx, id)
	if err != nil {
		return nil, err
	}

	return user, nil
}

func (s *Service) GetView(ctx context.Context, userID uuid.UUID) (UserView, error) {
	user, err := s.Get(ctx, userID)
	if err != nil {
		return UserView{}, err
	}

	userPresence, _ := s.cache.GetPresence(ctx, user.ID())

	return ToUserView(user, userPresence, fields.Now()), nil
}

type UpdateEmailParams struct {
	UserID   uuid.UUID
	NewEmail string
	Password string
}

func (s *Service) UpdateEmail(ctx context.Context, p UpdateEmailParams) (*User, error) {
	newEmail, err := ParseEmail("email", p.NewEmail)
	if err != nil {
		return nil, err
	}

	id, u, err := s.parseAndAuthenticate(ctx, p.UserID, p.Password)
	if err != nil {
		return nil, err
	}

	if u.email.Equals(newEmail.Text) {
		return u, nil
	}

	return s.update(ctx, func(txCtx context.Context, now fields.Timestamp) (*User, string, any, error) {
		updatedUser, err := s.repo.UpdateEmail(txCtx, id, newEmail, now)
		return updatedUser, EventUpdateEmail, EventUpdateEmailPayload{}, err
	})
}

type UpdateUsernameParams struct {
	UserID      uuid.UUID
	NewUsername string
	Password    string
}

func (s *Service) UpdateUsername(ctx context.Context, p UpdateUsernameParams) (*User, error) {
	newUsername, err := ParseUsername("username", p.NewUsername)
	if err != nil {
		return nil, err
	}

	id, u, err := s.parseAndAuthenticate(ctx, p.UserID, p.Password)
	if err != nil {
		return nil, err
	}

	if u.username.Equals(newUsername.Text) {
		return u, nil
	}

	return s.update(ctx, func(txCtx context.Context, now fields.Timestamp) (*User, string, any, error) {
		updatedUser, err := s.repo.UpdateUsername(txCtx, id, newUsername, now)
		return updatedUser, EventUpdateUsername, EventUpdateUsernamePayload{
			UserID:      id.String(),
			NewUsername: newUsername.String(),
		}, err
	})
}

type UpdatePasswordParams struct {
	UserID             uuid.UUID
	CurrentPassword    string
	NewPassword        string
	NewPasswordConfirm string
}

func (s *Service) UpdatePassword(ctx context.Context, p UpdatePasswordParams) error {
	currentPassword, err := ParsePassword("current_password", p.CurrentPassword)
	if err != nil {
		return err
	}

	newPassword, err := ParsePassword("new_password", p.NewPassword)
	if err != nil {
		return err
	}

	newPasswordConfirm, err := ParsePassword("new_password_confirm", p.NewPasswordConfirm)
	if err != nil {
		return err
	}

	if !newPassword.Equals(newPasswordConfirm.Text) {
		return ErrPasswordMismatch("new_password_confirm")
	}

	id, _, err := s.parseAndAuthenticate(ctx, p.UserID, currentPassword.String())
	if err != nil {
		return err
	}

	passwordHash, err := crypto.HashPassword(newPassword.String())
	if err != nil {
		return ErrPasswordHashFailed(err)
	}

	newPasswordHash, err := ParsePasswordHash("new_password_hash", passwordHash)
	if err != nil {
		return err
	}

	_, err = s.update(ctx, func(txCtx context.Context, now fields.Timestamp) (*User, string, any, error) {
		updatedUser, err := s.repo.UpdatePasswordHash(txCtx, id, newPasswordHash, now)
		if err != nil {
			return nil, "", nil, err
		}

		payload := EventUpdatePasswordPayload{
			UserID: updatedUser.ID().String(),
			Email:  updatedUser.Email().String(),
		}

		return updatedUser, EventUpdatePassword, payload, nil
	})

	return nil
}

type UpdatePreferredPresenceParams struct {
	UserID   uuid.UUID
	Presence *string
	Duration *string
}

func (s *Service) UpdatePreferredPresence(ctx context.Context, p UpdatePreferredPresenceParams) (*User, error) {
	id, err := fields.ParseRequiredID("id", p.UserID)
	if err != nil {
		return nil, err
	}

	preferredPresence, err := ParsePreferredPresence("preferred_presence", ptr.From(p.Presence))
	if err != nil {
		return nil, err
	}

	duration, err := ParsePreferredPresenceDurationString(ptr.From(p.Duration))
	if err != nil {
		return nil, err
	}

	now := fields.Now()
	until, err := duration.CalculateUntil(now)
	if err != nil {
		return nil, err
	}

	u, err := s.fetchValid(ctx, id)
	if err != nil {
		return nil, err
	}

	if u.PreferredPresence().Equals(preferredPresence) && u.PreferredPresenceUntil().Equals(until) {
		return u, nil
	}

	return s.update(ctx, func(txCtx context.Context, now fields.Timestamp) (*User, string, any, error) {
		updatedUser, err := s.repo.UpdatePresence(txCtx, id, preferredPresence, until, now)
		if err != nil {
			return nil, "", nil, err
		}

		payload := EventUpdatePreferredPresencePayload{}

		return updatedUser, EventUpdatePreferredPresence, payload, nil
	})
}

type UpdateProfileParams struct {
	UserID      uuid.UUID
	DisplayName string
	Bio         *string
	AvatarURL   *string
	BannerColor *string
}

func (s *Service) UpdateProfile(ctx context.Context, p UpdateProfileParams) (*User, error) {
	id, err := fields.ParseRequiredID("id", p.UserID)
	if err != nil {
		return nil, err
	}

	displayName, err := ParseDisplayName("display_name", p.DisplayName)
	if err != nil {
		return nil, err
	}

	bio, err := ParseBio("bio", ptr.From(p.Bio))
	if err != nil {
		return nil, err
	}

	avatarURL, err := fields.ParseURL("avatar_url", ptr.From(p.AvatarURL))
	if err != nil {
		return nil, err
	}

	bannerColor, err := fields.ParseHexColor("banner_color", ptr.From(p.BannerColor))
	if err != nil {
		return nil, err
	}

	u, err := s.fetchValid(ctx, id)
	if err != nil {
		return nil, err
	}

	if u.DisplayName().Equals(displayName.Text) &&
		u.Bio().Equals(bio.Text) &&
		u.AvatarURL().Equals(avatarURL) &&
		u.BannerColor().Equals(bannerColor.Text) {
		return u, nil
	}

	return s.update(ctx, func(txCtx context.Context, now fields.Timestamp) (*User, string, any, error) {
		updatedUser, err := s.repo.UpdateProfile(txCtx, id, displayName, bio, avatarURL, bannerColor, now)
		if err != nil {
			return nil, "", nil, err
		}

		payload := EventUpdateProfilePayload{
			UserID:      updatedUser.ID().String(),
			DisplayName: updatedUser.DisplayName().String(),
			Bio:         updatedUser.Bio().StringPtr(),
			AvatarURL:   updatedUser.AvatarURL().StringPtr(),
			BannerColor: updatedUser.BannerColor().StringPtr(),
		}

		return updatedUser, EventUpdateProfile, payload, nil
	})
}

type DisableParams struct {
	UserID   uuid.UUID
	Password string
}

func (s *Service) Disable(ctx context.Context, p DisableParams) error {
	id, u, err := s.parseAndAuthenticate(ctx, p.UserID, p.Password)
	if err != nil {
		return err
	}
	if u.IsDisabled() {
		return nil
	}

	_, err = s.update(ctx, func(txCtx context.Context, now fields.Timestamp) (*User, string, any, error) {
		updatedUser, err := s.repo.SetDisabled(txCtx, id, now, now)
		if err != nil {
			return nil, "", nil, err
		}

		payload := EventDisablePayload{
			UserID: updatedUser.ID().String(),
		}

		return updatedUser, EventDisable, payload, nil
	})

	return err
}

type ScheduleDeleteParams struct {
	UserID   uuid.UUID
	Password string
}

func (s *Service) ScheduleDelete(ctx context.Context, p ScheduleDeleteParams) error {
	id, u, err := s.parseAndAuthenticate(ctx, p.UserID, p.Password)
	if err != nil {
		return err
	}

	if u.IsScheduledForDeletion() {
		return nil
	}

	_, err = s.update(ctx, func(txCtx context.Context, now fields.Timestamp) (*User, string, any, error) {
		scheduledAt := fields.NewTimestamp(now.Time().Add(ScheduleDeleteGracePeriod))

		updatedUser, err := s.repo.SetDeleteSchedule(txCtx, id, scheduledAt, now, now)
		if err != nil {
			return nil, "", nil, err
		}

		payload := EventScheduleDeletePayload{
			UserID:      updatedUser.ID().String(),
			Email:       updatedUser.Email().String(),
			ScheduledAt: scheduledAt.String(),
		}

		return updatedUser, EventScheduleDelete, payload, nil
	})

	return err
}

func (s *Service) AnonymizeBatch(ctx context.Context) error {
	now := fields.Now()

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

	return s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		updatedUsers, err := s.repo.UpdateBatch(txCtx, users)
		if err != nil {
			return err
		}

		userIDs := make([]string, len(updatedUsers))
		for i, u := range updatedUsers {
			userIDs[i] = u.ID().String()
		}

		// payload := EventAnonymizeBatchPayload{
		// 	UserIDs: userIDs,
		// 	Count:   len(userIDs),
		// }

		// if _, err := s.outboxRepo.Publish(txCtx, EventAnonymizeBatch, payload); err != nil {
		// 	return err
		// }

		return nil
	})
}

func (s *Service) fetchValid(ctx context.Context, actorID fields.ID) (*User, error) {
	u, err := s.repo.Get(ctx, actorID)
	if err != nil {
		return nil, err
	}

	err = u.EnsureActive()
	if err != nil {
		return nil, err
	}

	return u, nil
}

func (s *Service) fetchAndAuthenticate(ctx context.Context, actorID fields.ID, password Password) (*User, error) {
	u, err := s.fetchValid(ctx, actorID)
	if err != nil {
		return nil, err
	}

	if err := crypto.ComparePassword(u.PasswordHash().String(), password.String()); err != nil {
		return nil, ErrInvalidPassword("password").Wrap(err)
	}

	return u, nil
}

func (s *Service) parseAndAuthenticate(
	ctx context.Context,
	rawUserID uuid.UUID,
	passwordRaw string,
) (fields.ID, *User, error) {
	id, err := fields.ParseRequiredID("id", rawUserID)
	if err != nil {
		return fields.ID{}, nil, err
	}

	password, err := ParsePassword("password", passwordRaw)
	if err != nil {
		return fields.ID{}, nil, err
	}

	u, err := s.fetchAndAuthenticate(ctx, id, password)
	if err != nil {
		return fields.ID{}, nil, err
	}

	return id, u, nil
}

func (s *Service) update(
	ctx context.Context,
	updateFn func(txCtx context.Context, now fields.Timestamp) (*User, string, any, error),
) (*User, error) {
	now := fields.Now()
	var updatedUser *User

	err := s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		var err error
		var eventType string
		var eventPayload any

		updatedUser, eventType, eventPayload, err = updateFn(txCtx, now)
		if err != nil {
			return err
		}

		return s.outboxRepo.Publish(txCtx, eventType, eventPayload)
	})
	if err != nil {
		return nil, err
	}

	return updatedUser, nil
}
