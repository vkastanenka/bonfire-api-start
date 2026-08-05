package cache

import (
	"context"
	"fmt"
	"time"

	"bonfire-api/internal/presence"
	"bonfire-api/internal/redis"

	"github.com/google/uuid"
)

type PresenceStore interface {
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	Get(ctx context.Context, key string, dest interface{}) error
	MGet(ctx context.Context, keys ...string) ([]interface{}, error)
}

type Presence struct {
	store PresenceStore
	ttl   time.Duration
}

func NewPresence(store PresenceStore, ttl time.Duration) *Presence {
	return &Presence{
		store: store,
		ttl:   ttl,
	}
}

func presenceKey(userID uuid.UUID) string {
	return fmt.Sprintf("presence:%s", userID.String())
}

func (s *Presence) SetPresence(ctx context.Context, userID uuid.UUID, p presence.Presence) error {
	key := presenceKey(userID)
	if err := s.store.Set(ctx, key, p.String(), s.ttl); err != nil {
		return redis.NewError(err, redis.ScopePresence)
	}
	return nil
}

func (s *Presence) GetPresence(ctx context.Context, userID uuid.UUID) (presence.Presence, error) {
	key := presenceKey(userID)

	var raw string
	err := s.store.Get(ctx, key, &raw)
	if redis.IsNotFoundError(err) {
		return presence.PresenceOffline, nil
	}
	if err != nil {
		return presence.PresenceUnknown, redis.NewError(err, redis.ScopePresence)
	}

	p, err := presence.New(raw)
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
		keys[i] = presenceKey(id)
	}

	vals, err := s.store.MGet(ctx, keys...)
	if err != nil {
		return nil, redis.NewError(err, redis.ScopePresence)
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
