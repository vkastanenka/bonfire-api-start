package repository

import (
	"context"
	"fmt"
	"time"

	"bonfire-api/internal/cache"
	"bonfire-api/internal/user"

	"github.com/google/uuid"
)

const presenceTTL = 5 * time.Minute

type Presence struct {
	cache cache.Manager
}

func NewPresence(cache cache.Manager) *Presence {
	return &Presence{cache: cache}
}

var _ user.PresenceRepository = (*Presence)(nil)

func (r *Presence) SetPresence(ctx context.Context, userID uuid.UUID, p user.Presence) error {
	key := presenceKey(userID)
	if err := r.cache.Set(ctx, key, p.String(), presenceTTL); err != nil {
		return cache.NewError(err, cache.ScopePresence)
	}
	return nil
}

func (r *Presence) GetPresence(ctx context.Context, userID uuid.UUID) (user.Presence, error) {
	var val string
	err := r.cache.Get(ctx, presenceKey(userID), &val)
	if cache.IsNotFoundError(err) {
		return user.PresenceOffline, nil
	}
	if err != nil {
		return user.PresenceUnknown, cache.NewError(err, cache.ScopePresence)
	}

	return user.NewPresence(val)
}

func (r *Presence) GetPresenceBulk(ctx context.Context, userIDs []uuid.UUID) (map[uuid.UUID]user.Presence, error) {
	results := make(map[uuid.UUID]user.Presence, len(userIDs))
	if len(userIDs) == 0 {
		return results, nil
	}

	// keys := make([]string, len(userIDs))
	// for i, id := range userIDs {
	// 	keys[i] = presenceKey(id)
	// }

	// values, err := r.cache.MGet(ctx, keys...)
	// if err != nil {
	// 	return nil, cache.NewError(err, cache.ScopePresence)
	// }

	// for i, id := range userIDs {
	// 	if values[i] == nil {
	// 		results[id] = user.PresenceOffline
	// 		continue
	// 	}

	// 	if valStr, ok := values[i].(string); ok {
	// 		results[id] = user.NewPresence(valStr)
	// 	} else {
	// 		results[id] = user.PresenceOffline
	// 	}
	// }

	return results, nil
}

func presenceKey(userID uuid.UUID) string {
	return fmt.Sprintf("user:{%s}:presence", userID.String())
}
