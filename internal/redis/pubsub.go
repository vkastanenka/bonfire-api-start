// internal/redis/pubsub.go
package redis

import (
	"context"
	"encoding/json"

	goredis "github.com/redis/go-redis/v9"
)

func (m *manager) Publish(ctx context.Context, channel string, payload interface{}) error {
	bytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return m.client.Publish(ctx, channel, bytes).Err()
}

func (m *manager) Subscribe(ctx context.Context, channel string) *goredis.PubSub {
	return m.client.Subscribe(ctx, channel)
}
