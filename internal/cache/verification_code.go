package cache

import (
	"context"
	"errors"
	"fmt"
	"time"

	"bonfire-api/internal/fields"
	"bonfire-api/internal/redis"
)

var ErrInvalidUserID = errors.New("invalid user ID")

type phoneUpdatePayload struct {
	Code  string `json:"code"`
	Phone string `json:"phone"`
}

func usersEmailUpdateCodeKey(userID fields.ID) string {
	return fmt.Sprintf("users:%s:email_update_code", userID.String())
}

func usersPhoneUpdateCodeKey(userID fields.ID) string {
	return fmt.Sprintf("users:%s:phone_update_code", userID.String())
}

type VerificationCodeCache struct {
	store      redis.Store
	defaultTTL time.Duration
}

func NewVerificationCodeCache(store redis.Store, defaultTTL time.Duration) *VerificationCodeCache {
	return &VerificationCodeCache{
		store:      store,
		defaultTTL: defaultTTL,
	}
}

// ============================================================================
// Email Update Verification Code
// ============================================================================

func (c *VerificationCodeCache) SetUserEmailUpdateCode(ctx context.Context, userID fields.ID, code fields.VerificationCode) error {
	if !userID.IsValid() {
		return redis.NewError(ErrInvalidUserID, redis.ScopeUser)
	}

	key := usersEmailUpdateCodeKey(userID)
	if err := c.store.Set(ctx, key, code.String(), c.defaultTTL); err != nil {
		return err
	}
	return nil
}

func (c *VerificationCodeCache) GetUserEmailUpdateCode(ctx context.Context, userID fields.ID) (fields.VerificationCode, error) {
	if !userID.IsValid() {
		return fields.VerificationCode{}, redis.NewError(ErrInvalidUserID, redis.ScopeUser)
	}

	var code string
	key := usersEmailUpdateCodeKey(userID)
	if err := c.store.Get(ctx, key, &code); err != nil {
		return fields.VerificationCode{}, err
	}
	return fields.NewVerificationCode(code)
}

func (c *VerificationCodeCache) DeleteUserEmailUpdateCode(ctx context.Context, userID fields.ID) error {
	if !userID.IsValid() {
		return redis.NewError(ErrInvalidUserID, redis.ScopeUser)
	}

	key := usersEmailUpdateCodeKey(userID)
	if err := c.store.Delete(ctx, key); err != nil {
		return err
	}
	return nil
}

// ============================================================================
// Phone Update Verification Code
// ============================================================================

func (c *VerificationCodeCache) SetUserPhoneUpdateCode(ctx context.Context, userID fields.ID, code fields.VerificationCode, phone string) error {
	if !userID.IsValid() {
		return redis.NewError(ErrInvalidUserID, redis.ScopeUser)
	}

	payload := phoneUpdatePayload{
		Code:  code.String(),
		Phone: phone,
	}

	key := usersPhoneUpdateCodeKey(userID)
	if err := c.store.Set(ctx, key, payload, c.defaultTTL); err != nil {
		return err
	}
	return nil
}

func (c *VerificationCodeCache) GetUserPhoneUpdateCode(ctx context.Context, userID fields.ID) (fields.VerificationCode, string, error) {
	if !userID.IsValid() {
		return fields.VerificationCode{}, "", redis.NewError(ErrInvalidUserID, redis.ScopeUser)
	}

	var payload phoneUpdatePayload
	key := usersPhoneUpdateCodeKey(userID)
	if err := c.store.Get(ctx, key, &payload); err != nil {
		return fields.VerificationCode{}, "", err
	}

	vCode, err := fields.NewVerificationCode(payload.Code)
	if err != nil {
		return fields.VerificationCode{}, "", err
	}

	return vCode, payload.Phone, nil
}

func (c *VerificationCodeCache) DeleteUserPhoneUpdateCode(ctx context.Context, userID fields.ID) error {
	if !userID.IsValid() {
		return redis.NewError(ErrInvalidUserID, redis.ScopeUser)
	}

	key := usersPhoneUpdateCodeKey(userID)
	if err := c.store.Delete(ctx, key); err != nil {
		return err
	}
	return nil
}
