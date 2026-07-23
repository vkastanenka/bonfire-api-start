package store

import (
	"context"
	"fmt"
	"time"

	"bonfire-api/internal/cache"
	"bonfire-api/internal/presence"

	"github.com/google/uuid"
)

type Presence struct {
	q   cache.Store
	ttl time.Duration
}

var _ presence.Repository = (*Presence)(nil)

func NewPresence(q cache.Store, ttl time.Duration) *Presence {
	return &Presence{
		q:   q,
		ttl: ttl,
	}
}

func presenceKey(userID uuid.UUID) string {
	return fmt.Sprintf("presence:%s", userID.String())
}

func (r *Presence) SetPresence(ctx context.Context, userID uuid.UUID, p presence.Presence) error {
	key := presenceKey(userID)
	if err := r.q.Set(ctx, key, p.String(), r.ttl); err != nil {
		return cache.NewError(err, cache.ScopePresence)
	}
	return nil
}

func (r *Presence) GetPresence(ctx context.Context, userID uuid.UUID) (presence.Presence, error) {
	key := presenceKey(userID)

	var raw string
	err := r.q.Get(ctx, key, &raw)
	if cache.IsNotFoundError(err) {
		return presence.PresenceOffline, nil
	}
	if err != nil {
		return presence.PresenceUnknown, cache.NewError(err, cache.ScopePresence)
	}

	p, err := presence.New(raw)
	if err != nil {
		return presence.PresenceOffline, nil
	}

	return p, nil
}

func (r *Presence) GetPresenceBulk(ctx context.Context, userIDs []uuid.UUID) (map[uuid.UUID]presence.Presence, error) {
	if len(userIDs) == 0 {
		return make(map[uuid.UUID]presence.Presence), nil
	}

	keys := make([]string, len(userIDs))
	for i, id := range userIDs {
		keys[i] = presenceKey(id)
	}

	vals, err := r.q.MGet(ctx, keys...)
	if err != nil {
		return nil, cache.NewError(err, cache.ScopePresence)
	}

	result := make(map[uuid.UUID]presence.Presence, len(userIDs))
	for i, val := range vals {
		id := userIDs[i]

		if val == nil {
			result[id] = presence.PresenceOffline
			continue
		}

		switch v := val.(type) {
		case string:
			if p, err := presence.New(v); err == nil {
				result[id] = p
				continue
			}
		case []byte:
			if p, err := presence.ParseBytes(v); err == nil {
				result[id] = p
				continue
			}
		}

		result[id] = presence.PresenceOffline
	}

	return result, nil
}
