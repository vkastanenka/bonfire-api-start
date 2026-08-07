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

func usersEmailUpdateCodeKey(userID fields.ID) string {
	return fmt.Sprintf("users:%s:email_update_code", userID.String())
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
