package repository

import (
	"context"
	"fmt"
	"time"

	"bonfire-api/internal/cache"
)

type Cooldown struct {
	q cache.Querier
}

func NewCooldown(q cache.Querier) *Cooldown {
	return &Cooldown{q: q}
}

func cooldownKey(scope, action, identifier string) string {
	return fmt.Sprintf("cooldown:%s:%s:{%s}", scope, action, identifier)
}

func (r *Cooldown) Get(ctx context.Context, scope, action, identifier string) (bool, error) {
	key := cooldownKey(scope, action, identifier)

	var dummy bool
	err := r.q.Get(ctx, key, &dummy)
	if cache.IsNotFoundError(err) {
		return false, nil
	}
	if err != nil {
		return false, cache.NewError(err, cache.ScopeCooldown)
	}

	return true, nil
}

func (r *Cooldown) Set(ctx context.Context, scope, action, identifier string, ttl time.Duration) error {
	key := cooldownKey(scope, action, identifier)

	if err := r.q.Set(ctx, key, true, ttl); err != nil {
		return cache.NewError(err, cache.ScopeCooldown)
	}

	return nil
}
