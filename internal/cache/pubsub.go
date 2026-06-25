package cache

import (
	"context"
	"encoding/json"

	"github.com/redis/go-redis/v9"
)

// Pillar 3: Message Routing
// This automatically handles the JSON conversion of your complex WebSocket payloads before throwing them onto the Redis event bus.

func (m *redisManager) Publish(ctx context.Context, channel string, payload interface{}) error {
	bytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return m.client.Publish(ctx, channel, bytes).Err()
}

func (m *redisManager) Subscribe(ctx context.Context, channel string) *redis.PubSub {
	return m.client.Subscribe(ctx, channel)
}
