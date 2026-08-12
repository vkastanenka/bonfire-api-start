package cache

import (
	"context"
	"fmt"
	"time"

	"bonfire-api/internal/fields"
	"bonfire-api/internal/redis"

	"github.com/google/uuid"
)

func FriendsListKey(userID fields.ID) string {
	return fmt.Sprintf("friends:%s", userID.String())
}

type FriendRelation struct {
	PeerID    uuid.UUID `json:"peer_id"`
	ActorID   uuid.UUID `json:"actor_id"`
	ChannelID uuid.UUID `json:"channel_id"`
}

type FriendCache struct {
	engine *KeyCache[fields.ID, []FriendRelation]
	store  *redis.Store
}

func NewFriendCache(store *redis.Store, ttl time.Duration) *FriendCache {
	scopedStore := store.WithScope(redis.ScopeFriend)
	engine := NewKeyCache[fields.ID, []FriendRelation](
		scopedStore,
		ttl,
		FriendsListKey,
	)
	return &FriendCache{
		engine: engine,
		store:  scopedStore,
	}
}

// Get fetches a user's friend list from Redis.
func (c *FriendCache) Get(ctx context.Context, userID fields.ID) ([]FriendRelation, error) {
	cached, err := c.engine.Get(ctx, userID)
	if err != nil {
		return nil, c.store.Err(err)
	}
	if cached == nil {
		return nil, nil
	}

	return *cached, nil
}

// Set stores the complete friend list for a single user.
func (c *FriendCache) Set(ctx context.Context, userID fields.ID, relations []FriendRelation) error {
	if !userID.IsValid() {
		return nil
	}
	if err := c.engine.Set(ctx, userID, &relations); err != nil {
		return c.store.Err(err)
	}
	return nil
}

// Invalidate clears the cached friend lists for one or more users when a relationship mutates.
func (c *FriendCache) Delete(ctx context.Context, userIDs ...fields.ID) error {
	validIDs := make([]fields.ID, 0, len(userIDs))
	for _, id := range userIDs {
		if id.IsValid() {
			validIDs = append(validIDs, id)
		}
	}

	if len(validIDs) == 0 {
		return nil
	}

	if err := c.engine.DeleteBatch(ctx, validIDs); err != nil {
		return c.store.Err(err)
	}

	return nil
}
