package store

import (
	"context"
	"fmt"
	"time"

	"bonfire-api/internal/cache"
)

type Shield struct {
	q cache.Querier
}

func NewShield(q cache.Querier) *Shield {
	return &Shield{q: q}
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
	err := r.q.Get(ctx, k, &dummy)
	if cache.IsNotFoundError(err) {
		return false, nil
	}
	if err != nil {
		return false, cache.NewError(err, cache.ScopeCooldown)
	}

	return true, nil
}

func (r *Shield) SetCooldown(ctx context.Context, scope, action, identifier string, ttl time.Duration) error {
	k := cooldownKey(scope, action, identifier)

	if err := r.q.Set(ctx, k, true, ttl); err != nil {
		return cache.NewError(err, cache.ScopeCooldown)
	}

	return nil
}

func (r *Shield) IncrementFailures(ctx context.Context, key string, window time.Duration) (int64, error) {
	k := failureCountKey(key)

	count, err := r.q.Increment(ctx, k, window)
	if err != nil {
		return 0, cache.NewError(err, cache.ScopeCooldown)
	}

	if count == 1 {
		_ = r.q.Expire(ctx, k, window)
	}

	return count, nil
}

func (r *Shield) Lockout(ctx context.Context, key string, duration time.Duration) error {
	k := lockoutKey(key)

	if err := r.q.Set(ctx, k, true, duration); err != nil {
		return cache.NewError(err, cache.ScopeCooldown)
	}

	return nil
}

func (r *Shield) IsLocked(ctx context.Context, key string) (bool, error) {
	k := lockoutKey(key)

	var dummy bool
	err := r.q.Get(ctx, k, &dummy)
	if cache.IsNotFoundError(err) {
		return false, nil
	}
	if err != nil {
		return false, cache.NewError(err, cache.ScopeCooldown)
	}

	return true, nil
}

func (r *Shield) ResetFailures(ctx context.Context, key string) error {
	k := failureCountKey(key)

	if err := r.q.Delete(ctx, k); err != nil && !cache.IsNotFoundError(err) {
		return cache.NewError(err, cache.ScopeCooldown)
	}

	return nil
}

// IsTokenConsumed checks whether a single-use token (by JTI / ID) has already been used.
func (r *Shield) IsTokenConsumed(ctx context.Context, tokenID string) (bool, error) {
	k := consumedTokenKey(tokenID)

	var dummy bool
	err := r.q.Get(ctx, k, &dummy)
	if cache.IsNotFoundError(err) {
		return false, nil
	}
	if err != nil {
		return false, cache.NewError(err, cache.ScopeCooldown)
	}

	return true, nil
}

// MarkTokenConsumed marks a single-use token as consumed for the remaining TTL of the token.
func (r *Shield) MarkTokenConsumed(ctx context.Context, tokenID string, ttl time.Duration) error {
	k := consumedTokenKey(tokenID)

	if err := r.q.Set(ctx, k, true, ttl); err != nil {
		return cache.NewError(err, cache.ScopeCooldown)
	}

	return nil
}
