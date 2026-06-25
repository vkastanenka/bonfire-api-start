package cache

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// Pillar 2: Heartbeats
// This is heavily optimized for a chat application.
// Notice the GetBulkPresence function: in Discord, when you load a server, you need to know the online status of 1,000 members immediately.
// We use Redis MGET to fetch them all in a single lightning-fast query.

// presenceTTL is how long a user remains "Online" without sending a heartbeat.
const presenceTTL = 30 * time.Second

func (m *redisManager) Heartbeat(ctx context.Context, userID string) error {
	key := UserPresenceKey(userID)
	// Sets the value to "1" and resets the 30-second expiration timer
	return m.client.Set(ctx, key, "1", presenceTTL).Err()
}

func (m *redisManager) GetPresence(ctx context.Context, userID string) (bool, error) {
	key := UserPresenceKey(userID)
	err := m.client.Get(ctx, key).Err()
	if err == redis.Nil {
		return false, nil // Key expired, user is offline
	}
	if err != nil {
		return false, err
	}
	return true, nil // Key exists, user is online
}

func (m *redisManager) GetBulkPresence(ctx context.Context, userIDs []string) (map[string]bool, error) {
	if len(userIDs) == 0 {
		return nil, nil
	}

	keys := make([]string, len(userIDs))
	for i, id := range userIDs {
		keys[i] = UserPresenceKey(id)
	}

	// MGet executes in one network round trip
	results, err := m.client.MGet(ctx, keys...).Result()
	if err != nil {
		return nil, err
	}

	presenceMap := make(map[string]bool)
	for i, id := range userIDs {
		// If the result is nil, the user's presence key expired (offline)
		presenceMap[id] = results[i] != nil
	}

	return presenceMap, nil
}
