package user

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"bonfire-api/internal/crypto"
	"bonfire-api/internal/errs"
	"bonfire-api/internal/fields"
	"bonfire-api/internal/outbox"
	"bonfire-api/internal/pkg/ptr"

	"github.com/google/uuid"
)

type Cache interface {
	Delete(ctx context.Context, userID fields.ID) error
	DeleteBatch(ctx context.Context, userIDs []fields.ID) error
}

type Repository interface {
	Get(ctx context.Context, id fields.ID) (*User, error)
	GetCached(ctx context.Context, id fields.ID) (*User, error)
	GetDeleteScheduledBatch(ctx context.Context, currentTime fields.Timestamp, batchLimit int32) ([]*User, error)
	Update(ctx context.Context, u *User) (*User, error)
	UpdateBatch(ctx context.Context, usersJson []byte) ([]*User, error)
}

type OutboxRepository interface {
	Publish(ctx context.Context, variant string, payload any) (*outbox.Event, error)
	PublishBatch(ctx context.Context, items []outbox.BatchItem) ([]*outbox.Event, error)
}

type TX interface {
	ExecTx(ctx context.Context, fn func(txCtx context.Context) error) error
}

type Service struct {
	c  Cache
	r  Repository
	o  OutboxRepository
	tx TX
}

func NewService(
	c Cache,
	r Repository,
	o OutboxRepository,
	tx TX,
) *Service {
	return &Service{
		c:  c,
		r:  r,
		o:  o,
		tx: tx,
	}
}

func (s *Service) Get(ctx context.Context, rawID uuid.UUID) (*User, error) {
	id, err := fields.ParseRequiredID("id", rawID)
	if err != nil {
		return nil, err
	}

	u, err := s.r.GetCached(ctx, id)
	if err != nil {
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
	id, err := fields.ParseRequiredID("id", p.UserID)
	if err != nil {
		return nil, err
	}

	newEmail, err := ParseEmail("email", p.NewEmail)
	if err != nil {
		return nil, err
	}

	password, err := ParsePassword("password", p.Password)
	if err != nil {
		return nil, err
	}

	u, err := s.authenticateAndFetch(ctx, id, password)
	if err != nil {
		return nil, err
	}

	if u.email.Equals(newEmail) {
		return u, nil
	}

	oldEmail := u.email.String()
	u.UpdateEmail(newEmail, fields.NewTimestampFromTime(time.Now()))

	var updatedUser *User
	err = s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		var err error
		updatedUser, err = s.r.Update(txCtx, u)
		if err != nil {
			return err
		}

		payload := EventUpdateEmailPayload{
			UserID:   updatedUser.ID().String(),
			OldEmail: oldEmail,
			NewEmail: updatedUser.Email().String(),
		}

		if _, err := s.o.Publish(txCtx, EventUpdateEmail, payload); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	s.invalidateCache(ctx, updatedUser.ID(), "update email")

	return updatedUser, nil
}

type UpdateUsernameParams struct {
	UserID      uuid.UUID
	NewUsername string
	Password    string
}

func (s *Service) UpdateUsername(ctx context.Context, p UpdateUsernameParams) (*User, error) {
	id, err := fields.ParseRequiredID("id", p.UserID)
	if err != nil {
		return nil, err
	}

	newUsername, err := ParseUsername("username", p.NewUsername)
	if err != nil {
		return nil, err
	}

	password, err := ParsePassword("password", p.Password)
	if err != nil {
		return nil, err
	}

	u, err := s.authenticateAndFetch(ctx, id, password)
	if err != nil {
		return nil, err
	}

	if u.username.Equals(newUsername) {
		return u, nil
	}

	oldUsername := u.username.String()
	u.UpdateUsername(newUsername, fields.NewTimestampFromTime(time.Now()))

	var updatedUser *User
	err = s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		var err error
		updatedUser, err = s.r.Update(txCtx, u)
		if err != nil {
			return err
		}

		payload := EventUpdateUsernamePayload{
			UserID:      updatedUser.ID().String(),
			OldUsername: oldUsername,
			NewUsername: updatedUser.Username().String(),
		}

		if _, err := s.o.Publish(txCtx, EventUpdateUsername, payload); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	s.invalidateCache(ctx, updatedUser.ID(), "update username")

	return updatedUser, nil
}

type UpdatePasswordParams struct {
	UserID             uuid.UUID
	CurrentPassword    string
	NewPassword        string
	NewPasswordConfirm string
}

func (s *Service) UpdatePassword(ctx context.Context, p UpdatePasswordParams) error {
	id, err := fields.ParseRequiredID("id", p.UserID)
	if err != nil {
		return err
	}

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

	if !newPassword.Equals(newPasswordConfirm) {
		return errs.InvalidArgument("Passwords must match.").
			FieldViolation("new_password_confirm", "Passwords do not match.", "PASSWORD_MISMATCH")
	}

	u, err := s.authenticateAndFetch(ctx, id, currentPassword)
	if err != nil {
		return err
	}

	passwordHash, err := crypto.HashPassword(newPassword.String())
	if err != nil {
		return errs.Internal("Failed to hash password.").Wrap(err)
	}

	newPasswordHash, err := ParsePasswordHash("new_password_hash", passwordHash)
	if err != nil {
		return err
	}

	u.UpdatePasswordHash(newPasswordHash, fields.NewTimestampFromTime(time.Now()))

	err = s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		updatedUser, err := s.r.Update(txCtx, u)
		if err != nil {
			return err
		}

		payload := EventUpdatePasswordPayload{
			UserID: updatedUser.ID().String(),
			Email:  updatedUser.Email().String(),
		}

		if _, err := s.o.Publish(txCtx, EventUpdatePassword, payload); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return err
	}

	s.invalidateCache(ctx, u.ID(), "update password")

	return nil
}

type UpdatePreferredPresenceParams struct {
	UserID   uuid.UUID
	Presence *string
	Until    *string
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

	until, err := fields.ParseTimestamp("preferred_presence_until", ptr.From(p.Until))
	if err != nil {
		return nil, err
	}

	u, err := s.r.GetCached(ctx, id)
	if err != nil {
		return nil, err
	}

	if err := u.EnsureActive(); err != nil {
		return nil, err
	}

	u.UpdatePreferredPresence(preferredPresence, until, fields.NewTimestampFromTime(time.Now()))

	var updatedUser *User
	err = s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		var err error
		updatedUser, err = s.r.Update(txCtx, u)
		if err != nil {
			return err
		}

		payload := EventUpdatePreferredPresencePayload{
			UserID:            updatedUser.ID().String(),
			PreferredPresence: updatedUser.PreferredPresence().StringPtr(),
			Until:             updatedUser.PreferredPresenceUntil().StringPtr(),
		}

		if _, err := s.o.Publish(txCtx, EventUpdatePreferredPresence, payload); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	s.invalidateCache(ctx, updatedUser.ID(), "update preferred presence")

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

	u, err := s.r.GetCached(ctx, id)
	if err != nil {
		return nil, err
	}

	if err := u.EnsureActive(); err != nil {
		return nil, err
	}

	u.UpdateProfile(
		displayName,
		bio,
		avatarURL,
		bannerColor,
		fields.NewTimestampFromTime(time.Now()),
	)

	var updatedUser *User
	err = s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		var err error
		updatedUser, err = s.r.Update(txCtx, u)
		if err != nil {
			return err
		}

		payload := EventUpdateProfilePayload{
			UserID:      updatedUser.ID().String(),
			DisplayName: updatedUser.DisplayName().String(),
			Bio:         updatedUser.Bio().StringPtr(),
			AvatarURL:   updatedUser.AvatarURL().StringPtr(),
			BannerColor: updatedUser.BannerColor().StringPtr(),
		}

		if _, err := s.o.Publish(txCtx, EventUpdateProfile, payload); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	s.invalidateCache(ctx, updatedUser.ID(), "update profile")

	return updatedUser, nil
}

type DisableParams struct {
	UserID   uuid.UUID
	Password string
}

func (s *Service) Disable(ctx context.Context, p DisableParams) error {
	id, err := fields.ParseRequiredID("id", p.UserID)
	if err != nil {
		return err
	}

	password, err := ParsePassword("password", p.Password)
	if err != nil {
		return err
	}

	u, err := s.authenticateAndFetch(ctx, id, password)
	if err != nil {
		return err
	}

	u.Disable(fields.NewTimestampFromTime(time.Now()))

	err = s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		updatedUser, err := s.r.Update(txCtx, u)
		if err != nil {
			return err
		}

		payload := EventDisablePayload{
			UserID: updatedUser.ID().String(),
		}

		if _, err := s.o.Publish(txCtx, EventDisable, payload); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return err
	}

	s.invalidateCache(ctx, u.ID(), "disable user")

	return nil
}

const ScheduleDeleteGracePeriod = 30 * 24 * time.Hour

type ScheduleDeleteParams struct {
	UserID   uuid.UUID
	Password string
}

func (s *Service) ScheduleDelete(ctx context.Context, p ScheduleDeleteParams) error {
	id, err := fields.ParseRequiredID("id", p.UserID)
	if err != nil {
		return err
	}

	password, err := ParsePassword("password", p.Password)
	if err != nil {
		return err
	}

	u, err := s.authenticateAndFetch(ctx, id, password)
	if err != nil {
		return err
	}

	t := time.Now()
	now := fields.NewTimestampFromTime(t)
	scheduledAt := fields.NewTimestampFromTime(t.Add(ScheduleDeleteGracePeriod))

	u.ScheduleDelete(scheduledAt, now)

	err = s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		updatedUser, err := s.r.Update(txCtx, u)
		if err != nil {
			return err
		}

		payload := EventScheduleDeletePayload{
			UserID:      updatedUser.ID().String(),
			Email:       updatedUser.Email().String(),
			ScheduledAt: scheduledAt.String(),
		}

		if _, err := s.o.Publish(txCtx, EventScheduleDelete, payload); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return err
	}

	s.invalidateCache(ctx, u.ID(), "schedule delete")

	return nil
}

const AnonymizeBatchSize = 100

func (s *Service) AnonymizeBatch(ctx context.Context) error {
	now := fields.NewTimestampFromTime(time.Now())
	users, err := s.r.GetDeleteScheduledBatch(ctx, now, AnonymizeBatchSize)
	if err != nil {
		return err
	}

	if len(users) == 0 {
		return nil
	}

	userIDs := make([]fields.ID, len(users))
	batchItems := make([]outbox.BatchItem, len(users))

	for i, u := range users {
		u.Anonymize(now)
		userIDs[i] = u.ID()

		batchItems[i] = outbox.BatchItem{
			Variant: EventAnonymized,
			Payload: EventAnonymizedPayload{
				UserID: u.ID().String(),
			},
		}
	}

	usersJSON, err := json.Marshal(users)
	if err != nil {
		return errs.Internal("Failed to marshal anonymized users batch.").Wrap(err)
	}

	err = s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		if _, err := s.r.UpdateBatch(txCtx, usersJSON); err != nil {
			return err
		}

		if _, err := s.o.PublishBatch(txCtx, batchItems); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return err
	}

	if err := s.c.DeleteBatch(ctx, userIDs); err != nil {
		slog.WarnContext(ctx, "failed to invalidate batch user cache after anonymization",
			"count", len(userIDs),
			"error", err,
		)
	}

	return nil
}

func (s *Service) invalidateCache(ctx context.Context, id fields.ID, action string) {
	if err := s.c.Delete(ctx, id); err != nil {
		slog.WarnContext(ctx, "failed to invalidate user cache",
			"user_id", id.String(),
			"action", action,
			"error", err,
		)
	}
}

func (s *Service) authenticateAndFetch(ctx context.Context, id fields.ID, password Password) (*User, error) {
	u, err := s.r.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	if err := crypto.ComparePassword(u.PasswordHash().String(), password.String()); err != nil {
		return nil, errs.Unauthenticated("Invalid password.").
			FieldViolation("password", "Invalid password.", "INVALID_PASSWORD").
			Wrap(err)
	}

	if err := u.EnsureActive(); err != nil {
		return nil, err
	}

	return u, nil
}
