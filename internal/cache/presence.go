package cache

import (
	"context"
	"time"

	"bonfire-api/internal/presence"
	"bonfire-api/internal/redis"

	"github.com/google/uuid"
)

type Presence struct {
	store *redis.Store
	ttl   time.Duration
}

func NewPresence(store *redis.Store, ttl time.Duration) *Presence {
	return &Presence{
		store: store.WithScope(redis.ScopePresence),
		ttl:   ttl,
	}
}

func key(userID uuid.UUID) string {
	return "user:" + userID.String() + ":presence"
}

func (s *Presence) GetPresence(ctx context.Context, userID uuid.UUID) (presence.Presence, error) {
	k := key(userID)

	var raw string
	err := s.store.Get(ctx, k, &raw)
	if redis.IsCacheMiss(err) {
		return presence.PresenceOffline, nil
	}
	if err != nil {
		return presence.PresenceUnknown, err
	}

	p, err := presence.Parse(raw)
	if err != nil {
		return presence.PresenceOffline, nil
	}

	return p, nil
}

func (s *Presence) GetPresenceBatch(ctx context.Context, userIDs []uuid.UUID) (map[uuid.UUID]presence.Presence, error) {
	if len(userIDs) == 0 {
		return make(map[uuid.UUID]presence.Presence), nil
	}

	keys := make([]string, len(userIDs))
	for i, id := range userIDs {
		keys[i] = key(id)
	}

	vals, err := s.store.MGet(ctx, keys...)
	if err != nil {
		return nil, err
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
			if p, err := presence.Parse(v); err == nil {
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

func (s *Presence) SetPresence(ctx context.Context, userID uuid.UUID, p presence.Presence) error {
	k := key(userID)
	return s.store.Set(ctx, k, p.String(), s.ttl)
}
