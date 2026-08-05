package cache

import (
	"context"
	"fmt"
	"time"

	"bonfire-api/internal/redis"
)

type ShieldStore interface {
	Get(ctx context.Context, key string, dest interface{}) error
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	Delete(ctx context.Context, key ...string) error
	Increment(ctx context.Context, key string, window time.Duration) (int64, error)
	Expire(ctx context.Context, key string, ttl time.Duration) error
}

type Shield struct {
	store ShieldStore
}

func NewShield(store ShieldStore) *Shield {
	return &Shield{store: store}
}

func cooldownKey(scope, action, identifier string) string {
	return fmt.Sprintf("shield:cooldown:%s:%s:{%s}", scope, action, identifier)
}

func lockoutKey(key string) string {
	return fmt.Sprintf("shield:lockout:{%s}", key)
}

func failureCountKey(key string) string {
	return fmt.Sprintf("shield:failures:{%s}", key)
}

func consumedTokenKey(tokenID string) string {
	return fmt.Sprintf("shield:consumed_token:{%s}", tokenID)
}

func (r *Shield) GetCooldown(ctx context.Context, scope, action, identifier string) (bool, error) {
	k := cooldownKey(scope, action, identifier)

	var dummy bool
	err := r.store.Get(ctx, k, &dummy)
	if redis.IsNotFoundError(err) {
		return false, nil
	}
	if err != nil {
		return false, redis.NewError(err, redis.ScopeCooldown)
	}

	return true, nil
}

func (r *Shield) SetCooldown(ctx context.Context, scope, action, identifier string, ttl time.Duration) error {
	k := cooldownKey(scope, action, identifier)

	if err := r.store.Set(ctx, k, true, ttl); err != nil {
		return redis.NewError(err, redis.ScopeCooldown)
	}

	return nil
}

func (r *Shield) IncrementFailures(ctx context.Context, key string, window time.Duration) (int64, error) {
	k := failureCountKey(key)

	count, err := r.store.Increment(ctx, k, window)
	if err != nil {
		return 0, redis.NewError(err, redis.ScopeCooldown)
	}

	if count == 1 {
		_ = r.store.Expire(ctx, k, window)
	}

	return count, nil
}

func (r *Shield) Lockout(ctx context.Context, key string, duration time.Duration) error {
	k := lockoutKey(key)

	if err := r.store.Set(ctx, k, true, duration); err != nil {
		return redis.NewError(err, redis.ScopeCooldown)
	}

	return nil
}

func (r *Shield) IsLocked(ctx context.Context, key string) (bool, error) {
	k := lockoutKey(key)

	var dummy bool
	err := r.store.Get(ctx, k, &dummy)
	if redis.IsNotFoundError(err) {
		return false, nil
	}
	if err != nil {
		return false, redis.NewError(err, redis.ScopeCooldown)
	}

	return true, nil
}

func (r *Shield) ResetFailures(ctx context.Context, key string) error {
	k := failureCountKey(key)

	if err := r.store.Delete(ctx, k); err != nil && !redis.IsNotFoundError(err) {
		return redis.NewError(err, redis.ScopeCooldown)
	}

	return nil
}

func (r *Shield) IsTokenConsumed(ctx context.Context, tokenID string) (bool, error) {
	k := consumedTokenKey(tokenID)

	var dummy bool
	err := r.store.Get(ctx, k, &dummy)
	if redis.IsNotFoundError(err) {
		return false, nil
	}
	if err != nil {
		return false, redis.NewError(err, redis.ScopeCooldown)
	}

	return true, nil
}

func (r *Shield) MarkTokenConsumed(ctx context.Context, tokenID string, ttl time.Duration) error {
	k := consumedTokenKey(tokenID)

	if err := r.store.Set(ctx, k, true, ttl); err != nil {
		return redis.NewError(err, redis.ScopeCooldown)
	}

	return nil
}
