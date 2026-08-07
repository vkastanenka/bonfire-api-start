package user

import (
	"bonfire-api/internal/crypto"
	"bonfire-api/internal/errs"
	"bonfire-api/internal/fields"
	"bonfire-api/internal/outbox"
	"bonfire-api/internal/pkg/ptr"
	"context"
	"errors"
	"fmt"
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
	Create(ctx context.Context, u *User) (*User, error)
	Get(ctx context.Context, id ID) (*User, error)
	GetByEmail(ctx context.Context, email Email) (*User, error)
	GetDeleteScheduledBatch(ctx context.Context, currentTime Timestamp, batchLimit int32) ([]*User, error)
	Update(ctx context.Context, u *User) (*User, error)
	UpdateBatch(ctx context.Context, usersJson []byte) ([]*User, error)
}

type OutboxRepository interface {
	Publish(ctx context.Context, variant string, payload any) (*outbox.Event, error)
}

type Service struct {
	cache  Cache
	repo   Repository
	outbox OutboxRepository
}

func NewService(cache Cache, repo Repository, outbox OutboxRepository) *Service {
	return &Service{
		cache:  cache,
		repo:   repo,
		outbox: outbox,
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

	u, err := s.repo.Get(ctx, id)
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

func (s *Service) RequestUpdateEmailCode(ctx context.Context, rawID uuid.UUID) error {
	id, err := NewID(rawID)
	if err != nil {
		return err
	}

	u, err := s.repo.Get(ctx, id)
	if err != nil {
		return err
	}

	if err := u.EnsureActive(); err != nil {
		return err
	}

	code, err := crypto.GenerateVerificationCode(6)
	if err != nil {
		return fmt.Errorf("failed to generate verification code: %w", err)
	}

	// TODO: Update cache
	// cacheKey := fmt.Sprintf("email_update_code:%s", u.ID().String())
	// if err := s.cache.Set(ctx, cacheKey, code, 15*time.Minute); err != nil {
	// 	return fmt.Errorf("failed to store verification code: %w", err)
	// }

	_, err = s.outbox.Publish(ctx, EventRequestUpdateEmailCode, RequestUpdateEmailCodePayload{
		Email: u.Email().String(),
		Code:  code,
	})
	if err != nil {
		return err
	}

	return nil
}

func (s *Service) VerifyUpdateEmailCode(ctx context.Context, rawID uuid.UUID, rawVerificationCode string) error {
	// id, err := NewID(rawID)
	// if err != nil {
	// 	return err
	// }

	// verificationCode, err := NewVerificationCode(rawVerificationCode)
	// if err != nil {
	// 	return err
	// }

	// cacheKey := fmt.Sprintf("email_update_code:%s", id.String())
	// cachedCode, err := s.cache.Get(ctx, cacheKey)
	// if err != nil {
	// 	if errors.Is(err, cache.ErrNotFound) {
	// 		return ErrCodeExpiredOrInvalid
	// 	}
	// 	return err
	// }

	// if !verificationCode.Equals(cachedCode) {
	// 	return ErrCodeMismatch
	// }

	// // Delete code after single use to prevent replay attacks
	// _ = s.cache.Delete(ctx, cacheKey)

	return nil
}

func (s *Service) AnonymizeBatch(ctx context.Context, batchSize int32) error {
	if batchSize <= 0 {
		batchSize = 100
	}

	users, err := s.repo.GetDeleteScheduledBatch(ctx, time.Now().UTC(), batchSize)
	if err != nil {
		return fmt.Errorf("failed to list scheduled deletions: %w", err)
	}

	if len(users) == 0 {
		return nil
	}

	idsToDelete := make([]ID, len(users))
	for i, u := range users {
		u.Anonymize()
		idsToDelete[i] = u.ID()
	}

	_, err = s.repo.UpdateBatch(ctx, users)
	if err != nil {
		return fmt.Errorf("failed to batch update anonymized users: %w", err)
	}

	if cacheErr := s.cache.DeleteBatch(ctx, idsToDelete); cacheErr != nil {
		// Log cache error without failing the core transaction
	}

	return nil
}

// // VerifyEmail verifies a user's email address using a signed verification token.
// func (s *Service) VerifyEmail(ctx context.Context, tokenStr string) error {
// 	// 1. Guard Input
// 	if tokenStr == "" {
// 		return errs.InvalidArgument("Verification token is required.").
// 			FieldViolation("token", "Verification token is required.", "REQUIRED").
// 			Wrap(errors.New("verification token is required"))
// 	}

// 	// 2. Verify Email Token Signature & Claims
// 	claims, err := s.tokens.VerifyEmailVerify(tokenStr)
// 	if err != nil {
// 		return errs.Unauthenticated("Invalid or expired verification token.").
// 			FieldViolation("token", "Invalid or expired verification token.", "INVALID_TOKEN").
// 			Wrap(err)
// 	}

// 	// 3. Single-Use Token Check (Shield Store)
// 	// Prevents token replay attacks by checking if the token's JTI was already consumed
// 	consumed, err := s.shield.IsTokenConsumed(ctx, claims.ID)
// 	if err != nil {
// 		slog.ErrorContext(ctx, "failed to check token consumed state",
// 			"error", err,
// 			"token_id", claims.ID,
// 			"user_id", claims.UserID,
// 		)
// 	} else if consumed {
// 		return errs.Unauthenticated("Verification token has already been used.").
// 			FieldViolation("token", "Verification token has already been used.", "TOKEN_ALREADY_USED").
// 			Wrap(errors.New("verification token already used"))
// 	}

// 	// 4. Fetch User Aggregate directly from UserRepository
// 	u, err := s.users.Get(ctx, claims.UserID)
// 	if err != nil {
// 		if errs.IsNotFound(err) {
// 			return errs.Unauthenticated("Invalid or expired verification token.").
// 				FieldViolation("token", "User associated with this token no longer exists.", "USER_NOT_FOUND").
// 				Wrap(err)
// 		}
// 		return err
// 	}

// 	// 5. Idempotency Check
// 	// If already verified, mark token as consumed and exit early successfully
// 	if u.IsVerified() {
// 		s.consumeTokenNonBlocking(ctx, claims.ID, claims.ExpiresAt.Time)
// 		return nil
// 	}

// 	// 6. Mutate User Domain Aggregate State
// 	u.Verify(time.Now().UTC())

// 	// 7. TRANSACTION: Atomically persist user verification state
// 	persistCtx := context.WithoutCancel(ctx)

// 	txErr := s.tx.ExecTx(persistCtx, func(txCtx context.Context) error {
// 		return s.users.Update(txCtx, u)
// 	})

// 	if txErr != nil {
// 		return txErr
// 	}

// 	// 8. Consume Token non-blockingly for remaining TTL
// 	s.consumeTokenNonBlocking(persistCtx, claims.ID, claims.ExpiresAt.Time)

// 	return nil
// }

// // consumeTokenNonBlocking marks a single-use token as consumed until its expiration.
// func (s *Service) consumeTokenNonBlocking(ctx context.Context, tokenID string, expiresAt time.Time) {
// 	remainingTTL := time.Until(expiresAt)
// 	if remainingTTL <= 0 {
// 		return
// 	}

// 	if err := s.shield.MarkTokenConsumed(ctx, tokenID, remainingTTL); err != nil {
// 		slog.WarnContext(ctx, "failed to mark verification token as consumed",
// 			"error", err,
// 			"token_id", tokenID,
// 		)
// 	}
// }

// type RequestUpdatePhoneCodeParams struct {
// 	UserID   uuid.UUID
// 	Password string
// 	NewPhone string
// }

// func (s *Service) RequestUpdatePhoneCode(ctx context.Context, p RequestUpdatePhoneCodeParams) error {
// 	id, err := NewID(p.UserID)
// 	if err != nil {
// 		return err
// 	}

// 	newPhone, err := NewPhone(p.NewPhone)
// 	if err != nil {
// 		return err
// 	}

// 	password, err := NewPassword(p.Password)
// 	if err != nil {
// 		return err
// 	}

// 	u, err := s.repo.Get(ctx, id)
// 	if err != nil {
// 		return err
// 	}

// 	if err := u.EnsureActive(); err != nil {
// 		return err
// 	}

// 	// 1. Re-authenticate user before allowing phone change request
// 	if err := crypto.ComparePassword(u.PasswordHash().String(), password.String()); err != nil {
// 		return errs.Unauthenticated("Invalid password.").
// 			FieldViolation("password", "Invalid password.", "INVALID_CREDENTIALS").
// 			Wrap(err)
// 	}

// 	// 2. Generate 6-digit SMS verification code
// 	code, err := crypto.GenerateVerificationCode(6)
// 	if err != nil {
// 		return fmt.Errorf("failed to generate phone code: %w", err)
// 	}

// 	// 3. Cache payload binding the User ID, new Phone number, and generated Code
// 	// E.g., JSON payload: {"phone": "+15550199", "code": "837192"}
// 	cacheKey := fmt.Sprintf("phone_update_code:%s", id.String())
// 	cachePayload := PhoneUpdateCachePayload{
// 		Phone: newPhone.String(),
// 		Code:  code,
// 	}
// 	if err := s.cache.Set(ctx, cacheKey, cachePayload, 10*time.Minute); err != nil {
// 		return fmt.Errorf("failed to cache phone code payload: %w", err)
// 	}

// 	// 4. Publish SMS delivery event
// 	_, err = s.outbox.Publish(ctx, EventRequestUpdatePhoneCode, RequestUpdatePhoneCodePayload{
// 		Phone: newPhone.String(),
// 		Code:  code,
// 	})
// 	if err != nil {
// 		return fmt.Errorf("failed to publish outbox event: %w", err)
// 	}

// 	return nil
// }

// type VerifyUpdatePhoneCodeParams struct {
// 	UserID           uuid.UUID
// 	VerificationCode string
// }

// func (s *Service) VerifyUpdatePhoneCode(ctx context.Context, p VerifyUpdatePhoneCodeParams) (*User, error) {
// 	id, err := NewID(p.UserID)
// 	if err != nil {
// 		return nil, err
// 	}

// 	code, err := NewVerificationCode(p.VerificationCode)
// 	if err != nil {
// 		return nil, err
// 	}

// 	// 1. Fetch cached request payload
// 	cacheKey := fmt.Sprintf("phone_update_code:%s", id.String())
// 	cachedData, err := s.cache.GetPhoneUpdatePayload(ctx, cacheKey)
// 	if err != nil {
// 		if errors.Is(err, cache.ErrNotFound) {
// 			return nil, errs.InvalidArgument("Verification code has expired or is invalid.")
// 		}
// 		return nil, err
// 	}

// 	// 2. Validate input code against cached code
// 	if !code.Equals(cachedData.Code) {
// 		return nil, errs.InvalidArgument("Invalid verification code.")
// 	}

// 	// 3. Parse and validate the new phone number from cache payload
// 	newPhone, err := NewPhone(cachedData.Phone)
// 	if err != nil {
// 		return nil, err
// 	}

// 	// 4. Retrieve domain entity & apply update
// 	u, err := s.repo.Get(ctx, id)
// 	if err != nil {
// 		return nil, err
// 	}

// 	if err := u.UpdatePhone(newPhone); err != nil {
// 		return nil, err
// 	}

// 	// 5. Persist changes to Database
// 	updatedUser, err := s.repo.Update(ctx, u)
// 	if err != nil {
// 		return nil, err
// 	}

// 	// 6. Cleanup cache
// 	_ = s.cache.Delete(ctx, cacheKey)          // Clear the verification code
// 	_ = s.cache.Delete(ctx, updatedUser.ID())  // Invalidate stale user profile cache

// 	return updatedUser, nil
// }
