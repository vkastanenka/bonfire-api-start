package gateway

import (
	"context"
	"encoding/json"
	"fmt"

	pkgredis "bonfire-api/internal/redis"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type Broadcaster struct {
	redisClient *redis.Client
}

func NewBroadcaster(rdb *redis.Client) *Broadcaster {
	return &Broadcaster{redisClient: rdb}
}

// PublishToUser finds all gateway nodes hosting active sessions for targetUser and publishes the frame.
func (b *Broadcaster) PublishToUser(ctx context.Context, targetUser uuid.UUID, eventType string, data any) error {
	raw, err := json.Marshal(data)
	if err != nil {
		return err
	}

	presenceKey := fmt.Sprintf("gateway:user_nodes:%s", targetUser)
	nodeIDs, err := b.redisClient.SMembers(ctx, presenceKey).Result()
	if err != nil || len(nodeIDs) == 0 {
		return nil // User is offline
	}

	payload := NodeEventPayload{
		Type:       eventType,
		TargetUser: targetUser,
		Data:       raw,
	}

	for _, nodeID := range nodeIDs {
		channel := fmt.Sprintf("gateway:%s:events", nodeID)
		if err := pkgredis.Publish(ctx, b.redisClient, channel, payload); err != nil {
			return err
		}
	}

	return nil
}
