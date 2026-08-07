package user

import (
	"bonfire-api/internal/crypto"
	"bonfire-api/internal/errs"
	"bonfire-api/internal/fields"
	"bonfire-api/internal/outbox"
	"bonfire-api/internal/pkg/ptr"
	"bonfire-api/internal/redis"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
)

type Cache interface {
	Delete(ctx context.Context, id fields.ID) error
	DeleteBatch(ctx context.Context, ids []fields.ID) error
	Get(ctx context.Context, id fields.ID) (*User, error)
	GetBatch(ctx context.Context, ids []fields.ID) (map[fields.ID]*User, []fields.ID, error)
	Set(ctx context.Context, user *User) error
	SetBatch(ctx context.Context, users []*User) error
}

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

type VerificationCodeCache interface {
	DeleteUserEmailUpdateCode(ctx context.Context, userID fields.ID) error
	DeleteUserPhoneUpdateCode(ctx context.Context, userID fields.ID) error
	GetUserEmailUpdateCode(ctx context.Context, userID fields.ID) (fields.VerificationCode, error)
	GetUserPhoneUpdateCode(ctx context.Context, userID fields.ID) (fields.VerificationCode, string, error)
	SetUserEmailUpdateCode(ctx context.Context, userID fields.ID, code fields.VerificationCode) error
	SetUserPhoneUpdateCode(ctx context.Context, userID fields.ID, code fields.VerificationCode, phone string) error
}

type Service struct {
	cache      Cache
	repo       Repository
	cRepo      CachedRepository
	outbox     OutboxRepository
	vCodeCache VerificationCodeCache
}

func NewService(
	cache Cache,
	repo Repository,
	cachedRepo CachedRepository,
	outbox OutboxRepository,
	vCodeCache VerificationCodeCache,
) *Service {
	return &Service{
		cache:      cache,
		repo:       repo,
		cRepo:      cachedRepo,
		outbox:     outbox,
		vCodeCache: vCodeCache,
	}
}

func (s *Service) Get(ctx context.Context, rawID uuid.UUID) (*User, error) {
	id, err := fields.NewID(rawID)
	if err != nil {
		return nil, err
	}

	u, err := s.cRepo.Get(ctx, id)
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
	id, err := fields.NewID(p.UserID)
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

	u, err := s.cRepo.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	if err = crypto.ComparePassword(u.PasswordHash().String(), password.String()); err != nil {
		return nil, errs.Unauthenticated("Invalid password.").
			FieldViolation("password", "Invalid password.", "INVALID_CREDENTIALS").
			Wrap(errors.New("invalid credentials"))
	}

	u.UpdateEmail(newEmail, fields.NewTimestampFromTime(time.Now()))

	uu, err := s.cRepo.Update(ctx, u)
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
	id, err := fields.NewID(p.UserID)
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

	u, err := s.cRepo.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	if err = crypto.ComparePassword(u.PasswordHash().String(), password.String()); err != nil {
		return nil, errs.Unauthenticated("Invalid password.").
			FieldViolation("password", "Invalid password.", "INVALID_CREDENTIALS").
			Wrap(errors.New("invalid credentials"))
	}

	u.UpdateUsername(newUsername, fields.NewTimestampFromTime(time.Now()))

	uu, err := s.cRepo.Update(ctx, u)
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
	id, err := fields.NewID(p.UserID)
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

	u, err := s.cRepo.Get(ctx, id)
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

	newPasswordHash, err := NewPasswordHash(passwordHash)
	if err != nil {
		return err
	}

	u.UpdatePasswordHash(newPasswordHash, fields.NewTimestampFromTime(time.Now()))

	_, err = s.cRepo.Update(ctx, u)
	if err != nil {
		return err
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
	id, err := fields.NewID(p.UserID)
	if err != nil {
		return nil, err
	}

	preferredPresence, err := NewPreferredPresence(ptr.From(p.Presence))
	if err != nil {
		return nil, err
	}

	until, err := fields.NewTimestamp(ptr.From(p.Until))
	if err != nil {
		return nil, err
	}

	u, err := s.cRepo.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	u.UpdatePreferredPresence(preferredPresence, until, fields.NewTimestampFromTime(time.Now()))

	uu, err := s.cRepo.Update(ctx, u)
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
	id, err := fields.NewID(p.UserID)
	if err != nil {
		return nil, err
	}

	displayName, err := NewDisplayName(p.DisplayName)
	if err != nil {
		return nil, err
	}

	bio, err := NewBio(ptr.From(p.Bio))
	if err != nil {
		return nil, err
	}

	avatarURL, err := fields.NewURL(ptr.From(p.AvatarURL))
	if err != nil {
		return nil, err
	}

	bannerColor, err := fields.NewHexColor(ptr.From(p.BannerColor))
	if err != nil {
		return nil, err
	}

	u, err := s.cRepo.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	u.UpdateProfile(
		displayName,
		bio,
		avatarURL,
		bannerColor,
		fields.NewTimestampFromTime(time.Now()),
	)

	uu, err := s.cRepo.Update(ctx, u)
	if err != nil {
		return nil, err
	}

	return uu, nil
}

func (s *Service) Disable(ctx context.Context, rawID uuid.UUID, rawPassword string) error {
	id, err := fields.NewID(rawID)
	if err != nil {
		return err
	}

	password, err := NewPassword(rawPassword)
	if err != nil {
		return err
	}

	u, err := s.cRepo.Get(ctx, id)
	if err != nil {
		return err
	}

	if err := u.EnsureActive(); err != nil {
		return err
	}

	if err = crypto.ComparePassword(u.PasswordHash().String(), password.String()); err != nil {
		return errs.Unauthenticated("Invalid password.").
			FieldViolation("password", "Invalid password.", "INVALID_CREDENTIALS").
			Wrap(errors.New("invalid credentials"))
	}

	u.Disable(fields.NewTimestampFromTime(time.Now()))

	_, err = s.cRepo.Update(ctx, u)
	if err != nil {
		return err
	}

	return nil
}

func (s *Service) ScheduleDelete(ctx context.Context, rawID uuid.UUID, rawDeleteAt time.Time, rawPassword string) error {
	id, err := fields.NewID(rawID)
	if err != nil {
		return err
	}

	password, err := NewPassword(rawPassword)
	if err != nil {
		return err
	}

	deleteAt := fields.NewTimestampFromTime(rawDeleteAt)

	u, err := s.cRepo.Get(ctx, id)
	if err != nil {
		return err
	}

	if err := u.EnsureActive(); err != nil {
		return err
	}

	if err = crypto.ComparePassword(u.PasswordHash().String(), password.String()); err != nil {
		return errs.Unauthenticated("Invalid password.").
			FieldViolation("password", "Invalid password.", "INVALID_CREDENTIALS").
			Wrap(errors.New("invalid credentials"))
	}

	u.ScheduleDelete(deleteAt, fields.NewTimestampFromTime(time.Now()))

	_, err = s.cRepo.Update(ctx, u)
	if err != nil {
		return err
	}

	return nil
}

func (s *Service) RequestUpdateEmailCode(ctx context.Context, rawID uuid.UUID, rawEmail string) error {
	id, err := fields.NewID(rawID)
	if err != nil {
		return err
	}

	email, err := NewEmail(rawEmail)
	if err != nil {
		return err
	}

	rawCode, err := crypto.GenerateVerificationCode(6)
	if err != nil {
		return errs.Internal("failed to generate verification code").Wrap(err)
	}

	code, err := fields.NewVerificationCode(rawCode)
	if err != nil {
		return err
	}

	if err := s.vCodeCache.SetUserEmailUpdateCode(ctx, id, code); err != nil {
		return err
	}

	_, err = s.outbox.Publish(ctx, EventRequestUpdateEmailCode, RequestUpdateEmailCodePayload{
		Code:  code.String(),
		Email: email.String(),
	})
	if err != nil {
		return err
	}

	return nil
}

func (s *Service) VerifyUpdateEmailCode(ctx context.Context, rawID uuid.UUID, rawCandidateCode string) error {
	id, err := fields.NewID(rawID)
	if err != nil {
		return err
	}

	candidateCode, err := fields.NewVerificationCode(rawCandidateCode)
	if err != nil {
		return err
	}

	cachedCode, err := s.vCodeCache.GetUserEmailUpdateCode(ctx, id)
	if err != nil {
		return err
	}

	if !cachedCode.Equals(candidateCode) {
		return errs.InvalidArgument("invalid code").Wrap(err)
	}

	if err := s.vCodeCache.DeleteUserEmailUpdateCode(ctx, id); err != nil {
		slog.WarnContext(ctx, "failed to delete verification code after successful verification",
			"user_id", id.String(),
			"error", err,
		)
	}

	return nil
}

func (s *Service) AnonymizeBatch(ctx context.Context, batchSize int32) error {
	if batchSize <= 0 {
		batchSize = 100
	}

	now := fields.NewTimestampFromTime(time.Now())
	users, err := s.repo.GetDeleteScheduledBatch(ctx, now, batchSize)
	if err != nil {
		return fmt.Errorf("failed to fetch users scheduled for deletion: %w", err)
	}

	if len(users) == 0 {
		return nil
	}

	for _, u := range users {
		u.Anonymize(now)
	}

	usersJSON, err := json.Marshal(users)
	if err != nil {
		return fmt.Errorf("failed to marshal anonymized users batch: %w", err)
	}

	if _, err := s.cRepo.UpdateBatch(ctx, usersJSON); err != nil {
		return fmt.Errorf("failed to batch update anonymized users: %w", err)
	}

	return nil
}

func (s *Service) RequestUpdatePhoneCode(ctx context.Context, rawID uuid.UUID, rawPhone string) error {
	id, err := fields.NewID(rawID)
	if err != nil {
		return err
	}

	phone, err := NewPhone(rawPhone)
	if err != nil {
		return err
	}

	rawCode, err := crypto.GenerateVerificationCode(6)
	if err != nil {
		return fmt.Errorf("failed to generate phone code: %w", err)
	}

	code, err := fields.NewVerificationCode(rawCode)
	if err != nil {
		return err
	}

	if err := s.vCodeCache.SetUserPhoneUpdateCode(ctx, id, code, phone.String()); err != nil {
		return err
	}

	_, err = s.outbox.Publish(ctx, EventRequestUpdatePhoneCode, RequestUpdatePhoneCodePayload{
		Code:  code.String(),
		Phone: phone.String(),
	})
	if err != nil {
		return fmt.Errorf("failed to publish outbox event: %w", err)
	}

	return nil
}

func (s *Service) VerifyUpdatePhoneCode(ctx context.Context, rawID uuid.UUID, rawCandidateCode string) (*User, error) {
	id, err := fields.NewID(rawID)
	if err != nil {
		return nil, err
	}

	candidateCode, err := fields.NewVerificationCode(rawCandidateCode)
	if err != nil {
		return nil, err
	}

	cachedCode, rawPhone, err := s.vCodeCache.GetUserPhoneUpdateCode(ctx, id)
	if err != nil {
		if errors.Is(err, redis.ErrCacheMiss) {
			return nil, errs.InvalidArgument("Verification code has expired or is invalid.").
				FieldViolation("code", "Verification code has expired or is invalid.", "INVALID_CODE")
		}
		return nil, err
	}

	if !candidateCode.Equals(cachedCode) {
		return nil, errs.InvalidArgument("Invalid verification code.").
			FieldViolation("code", "Invalid verification code.", "INVALID_CODE")
	}

	newPhone, err := NewPhone(rawPhone)
	if err != nil {
		return nil, err
	}

	u, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	if err := u.UpdatePhone(newPhone, fields.NewTimestampFromTime(time.Now())); err != nil {
		return nil, err
	}

	updatedUser, err := s.repo.Update(ctx, u)
	if err != nil {
		return nil, err
	}

	_ = s.vCodeCache.DeleteUserPhoneUpdateCode(ctx, id)

	return updatedUser, nil
}
