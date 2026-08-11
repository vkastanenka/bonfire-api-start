package cache

import (
	"context"
	"fmt"
	"time"

	"bonfire-api/internal/fields"
	"bonfire-api/internal/pkg/ptr"
	"bonfire-api/internal/redis"
)

// Key generation helpers following the standard namespace format.
func UserEmailUpdateCodeKey(userID fields.ID) string {
	return fmt.Sprintf("users:%s:email_update_code", userID.String())
}

func UserPhoneUpdateCodeKey(userID fields.ID) string {
	return fmt.Sprintf("users:%s:phone_update_code", userID.String())
}

// PhoneUpdatePayload represents the JSON DTO cached for phone update verification.
type PhoneUpdatePayload struct {
	Code  string `json:"code"`
	Phone string `json:"phone"`
}

type VerificationCodeCache struct {
	emailCodeEngine *KeyCache[fields.ID, string]
	phoneCodeEngine *KeyCache[fields.ID, PhoneUpdatePayload]
}

func NewVerificationCodeCache(store *redis.Store, ttl time.Duration) *VerificationCodeCache {
	userScopeStore := store.WithScope(redis.ScopeVerificationCode)

	return &VerificationCodeCache{
		emailCodeEngine: NewKeyCache[fields.ID, string](
			userScopeStore,
			ttl,
			UserEmailUpdateCodeKey,
		),
		phoneCodeEngine: NewKeyCache[fields.ID, PhoneUpdatePayload](
			userScopeStore,
			ttl,
			UserPhoneUpdateCodeKey,
		),
	}
}

// ============================================================================
// Email Update Verification Code
// ============================================================================

func (c *VerificationCodeCache) SetUserEmailUpdateCode(
	ctx context.Context,
	userID fields.ID,
	code fields.VerificationCode,
) error {
	if !userID.IsValid() {
		return nil
	}
	return c.emailCodeEngine.Set(ctx, userID, code.StringPtr())
}

func (c *VerificationCodeCache) GetUserEmailUpdateCode(
	ctx context.Context,
	userID fields.ID,
) (fields.VerificationCode, error) {
	if !userID.IsValid() {
		return fields.VerificationCode{}, nil
	}

	rawCode, err := c.emailCodeEngine.Get(ctx, userID)
	if err != nil || rawCode == nil {
		return fields.VerificationCode{}, err
	}

	return fields.ParseVerificationCode("verification_code", *rawCode)
}

func (c *VerificationCodeCache) DeleteUserEmailUpdateCode(
	ctx context.Context,
	userID fields.ID,
) error {
	if !userID.IsValid() {
		return nil
	}
	return c.emailCodeEngine.Delete(ctx, userID)
}

// ============================================================================
// Phone Update Verification Code
// ============================================================================

func (c *VerificationCodeCache) SetUserPhoneUpdateCode(
	ctx context.Context,
	userID fields.ID,
	code fields.VerificationCode,
	phone string,
) error {
	if !userID.IsValid() {
		return nil
	}

	payload := PhoneUpdatePayload{
		Code:  code.String(),
		Phone: phone,
	}

	return c.phoneCodeEngine.Set(ctx, userID, ptr.To(payload))
}

func (c *VerificationCodeCache) GetUserPhoneUpdateCode(
	ctx context.Context,
	userID fields.ID,
) (fields.VerificationCode, string, error) {
	if !userID.IsValid() {
		return fields.VerificationCode{}, "", nil
	}

	payload, err := c.phoneCodeEngine.Get(ctx, userID)
	if err != nil || payload == nil {
		return fields.VerificationCode{}, "", err
	}

	vCode, err := fields.ParseVerificationCode("verification_code", payload.Code)
	if err != nil {
		return fields.VerificationCode{}, "", err
	}

	return vCode, payload.Phone, nil
}

func (c *VerificationCodeCache) DeleteUserPhoneUpdateCode(
	ctx context.Context,
	userID fields.ID,
) error {
	if !userID.IsValid() {
		return nil
	}
	return c.phoneCodeEngine.Delete(ctx, userID)
}
