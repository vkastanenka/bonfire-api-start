package user

import (
	"bonfire-api/internal/crypto"
	"bonfire-api/internal/errs"
	"bonfire-api/internal/fields"
	"bonfire-api/internal/outbox"
	"bonfire-api/internal/pkg/ptr"
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type Repository interface {
	Create(ctx context.Context, u *User) (*User, error)
	Get(ctx context.Context, id fields.ID) (*User, error)
	GetByEmail(ctx context.Context, email Email) (*User, error)
	GetDeleteScheduledBatch(ctx context.Context, currentTime fields.Timestamp, batchLimit int32) ([]*User, error)
	Update(ctx context.Context, u *User) (*User, error)
	UpdateBatch(ctx context.Context, usersJson []byte) ([]*User, error)
}

type CachedRepository interface {
	Create(ctx context.Context, u *User) (*User, error)
	Get(ctx context.Context, id fields.ID) (*User, error)
	Update(ctx context.Context, u *User) (*User, error)
	UpdateBatch(ctx context.Context, usersJson []byte) ([]*User, error)
}

type OutboxRepository interface {
	Publish(ctx context.Context, variant string, payload any) (*outbox.Event, error)
}

type Service struct {
	r  Repository
	cr CachedRepository
	o  OutboxRepository
}

func NewService(
	r Repository,
	cr CachedRepository,
	o OutboxRepository,
) *Service {
	return &Service{
		r:  r,
		cr: cr,
		o:  o,
	}
}

func (s *Service) Get(ctx context.Context, rawID uuid.UUID) (*User, error) {
	id, err := fields.ParseRequiredID("id", rawID)
	if err != nil {
		return nil, err
	}

	u, err := s.cr.Get(ctx, id)
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

	u.UpdateEmail(newEmail, fields.NewTimestampFromTime(time.Now()))

	uu, err := s.cr.Update(ctx, u)
	if err != nil {
		return nil, err
	}

	return uu, nil
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

	u.UpdateUsername(newUsername, fields.NewTimestampFromTime(time.Now()))

	uu, err := s.cr.Update(ctx, u)
	if err != nil {
		return nil, err
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

	_, err = s.cr.Update(ctx, u)
	if err != nil {
		return err
	}

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

	u, err := s.cr.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	if err := u.EnsureActive(); err != nil {
		return nil, err
	}

	u.UpdatePreferredPresence(preferredPresence, until, fields.NewTimestampFromTime(time.Now()))

	uu, err := s.cr.Update(ctx, u)
	if err != nil {
		return nil, err
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

	u, err := s.cr.Get(ctx, id)
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

	uu, err := s.cr.Update(ctx, u)
	if err != nil {
		return nil, err
	}

	return uu, nil
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

	_, err = s.cr.Update(ctx, u)
	if err != nil {
		return err
	}

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

	u.ScheduleDelete(
		fields.NewTimestampFromTime(time.Now().Add(ScheduleDeleteGracePeriod)),
		fields.NewTimestampFromTime(time.Now()),
	)

	_, err = s.cr.Update(ctx, u)
	if err != nil {
		return err
	}

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

	for _, u := range users {
		u.Anonymize(now)
	}

	usersJSON, err := json.Marshal(users)
	if err != nil {
		return errs.Internal("Failed to marshal anonymized users batch.").Wrap(err)
	}

	if _, err := s.cr.UpdateBatch(ctx, usersJSON); err != nil {
		return err
	}

	return nil
}

func (s *Service) authenticateAndFetch(ctx context.Context, id fields.ID, password Password) (*User, error) {
	u, err := s.r.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	if err := crypto.ComparePassword(u.PasswordHash().String(), password.String()); err != nil {
		return nil, ErrInvalidPassword(err)
	}

	if err := u.EnsureActive(); err != nil {
		return nil, err
	}

	return u, nil
}

func ErrInvalidPassword(err error) *errs.Error {
	return errs.Unauthenticated("Invalid password.").
		FieldViolation("password", "Invalid password.", "INVALID_PASSWORD").
		Wrap(err)
}
