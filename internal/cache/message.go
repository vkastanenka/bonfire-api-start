package cache

import (
	"context"
	"fmt"
	"time"

	"bonfire-api/internal/channel"
	"bonfire-api/internal/fields"
	"bonfire-api/internal/redis"

	"github.com/google/uuid"
	redisdriver "github.com/redis/go-redis/v9"
)

type MessageCache struct {
	*ScopeCache[fields.ID, Message]
	client redisdriver.Cmdable
	ttl    time.Duration
}

func NewMessageCache(client redisdriver.Cmdable, ttl time.Duration) *MessageCache {
	return &MessageCache{
		ScopeCache: NewScopeCache[fields.ID, Message](client, redis.ScopeMessage, messageKey),
		client:     client,
		ttl:        ttl,
	}
}

func (c *MessageCache) Get(ctx context.Context, id fields.ID) (*channel.Message, error) {
	dto, err := c.ScopeCache.Get(ctx, id)
	if err != nil || dto == nil {
		return nil, err
	}

	return dto.ToDomain()
}

func (c *MessageCache) GetBatch(
	ctx context.Context,
	ids []fields.ID,
) (map[fields.ID]*channel.Message, []fields.ID, error) {
	dtos, missing, err := c.ScopeCache.GetBatch(ctx, ids)
	if err != nil {
		return nil, nil, err
	}

	found := make(map[fields.ID]*channel.Message, len(dtos))
	for id, dto := range dtos {
		if dto == nil {
			missing = append(missing, id)
			continue
		}

		msg, err := dto.ToDomain()
		if err != nil {
			missing = append(missing, id)
			continue
		}
		found[id] = msg
	}

	return found, missing, nil
}

func (c *MessageCache) Set(ctx context.Context, msg *channel.Message) error {
	if msg == nil {
		return nil
	}

	dto, err := ParseMessage(msg)
	if err != nil {
		return err
	}

	return c.ScopeCache.Set(ctx, msg.ID(), dto, c.ttl)
}

func (c *MessageCache) SetBatch(ctx context.Context, messages []*channel.Message) error {
	dtos := make(map[fields.ID]Message, len(messages))
	for _, msg := range messages {
		if msg == nil {
			continue
		}

		dto, err := ParseMessage(msg)
		if err != nil {
			return err
		}
		dtos[msg.ID()] = dto
	}

	return c.ScopeCache.SetBatch(ctx, dtos, c.ttl)
}

// ListAround fetches message IDs from the Redis ZSET around an anchor timestamp,
// splits them into before/after windows, and bulk-fetches the actual message DTOs.
// Returns (messages, hitFeedMiss, error).
func (c *MessageCache) ListAround(
	ctx context.Context,
	channelID fields.ID,
	anchorTimestamp time.Time,
	beforeLimit, afterLimit int,
) ([]*channel.Message, bool, error) {
	setKey := channelMessagesKey(channelID)

	opCtx, cancel := context.WithTimeout(ctx, ScopeBatchTimeout)
	defer cancel()

	anchorScore := float64(anchorTimestamp.UnixNano())

	// Check if feed ZSET exists
	exists, err := c.client.Exists(opCtx, setKey).Result()
	if err != nil || exists == 0 {
		return nil, true, redis.NewError(err, redis.ScopeMessage)
	}

	// 1. Fetch messages strictly BEFORE anchor (ZREVRANGEBYSCORE descending from anchor)
	// We use limit (0, beforeLimit)
	beforeIDsStr, err := c.client.ZRevRangeByScore(opCtx, setKey, &redisdriver.ZRangeBy{
		Min:    "-inf",
		Max:    fmt.Sprintf("(%f", anchorScore), // exclusive of anchor score
		Count:  int64(beforeLimit),
		Offset: 0,
	}).Result()
	if err != nil {
		return nil, true, redis.NewError(err, redis.ScopeMessage)
	}

	// 2. Fetch anchor and messages AFTER anchor (ZRANGEBYSCORE ascending from anchor)
	afterIDsStr, err := c.client.ZRangeByScore(opCtx, setKey, &redisdriver.ZRangeBy{
		Min:    fmt.Sprintf("%f", anchorScore), // inclusive of anchor score
		Max:    "+inf",
		Count:  int64(afterLimit + 1), // +1 in case anchor is included
		Offset: 0,
	}).Result()
	if err != nil {
		return nil, true, redis.NewError(err, redis.ScopeMessage)
	}

	// Combine and deduplicate unique message IDs while preserving order
	seen := make(map[fields.ID]bool)
	var targetIDs []fields.ID

	// Before IDs are returned in reverse-chronological order; reverse them to chronological order
	for i := len(beforeIDsStr) - 1; i >= 0; i-- {
		parsedUUID, parseErr := uuid.Parse(beforeIDsStr[i])
		if parseErr != nil {
			return nil, true, nil // Malformed cache entry -> treat as feed miss
		}
		id, parseErr := fields.ParseRequiredID("id", parsedUUID)
		if parseErr != nil {
			return nil, true, nil
		}
		if !seen[id] {
			seen[id] = true
			targetIDs = append(targetIDs, id)
		}
	}

	for _, rawID := range afterIDsStr {
		parsedUUID, parseErr := uuid.Parse(rawID)
		if parseErr != nil {
			return nil, true, nil
		}
		id, parseErr := fields.ParseRequiredID("id", parsedUUID)
		if parseErr != nil {
			return nil, true, nil
		}
		if !seen[id] {
			seen[id] = true
			targetIDs = append(targetIDs, id)
		}
	}

	if len(targetIDs) == 0 {
		return nil, true, nil
	}

	// 3. Batch fetch individual message DTOs
	foundMap, missingIDs, err := c.GetBatch(ctx, targetIDs)
	if err != nil {
		return nil, false, err
	}

	// If any individual message keys expired or are missing, trigger feed/cache miss fallback
	if len(missingIDs) > 0 {
		return nil, true, nil
	}

	// Reconstruct final slice ordered chronologically based on targetIDs sequence
	messages := make([]*channel.Message, 0, len(targetIDs))
	for _, id := range targetIDs {
		if msg, ok := foundMap[id]; ok && msg != nil {
			messages = append(messages, msg)
		}
	}

	return messages, false, nil
}

// SetBatchWithFeed updates both individual message DTOs and registers them into the channel ZSET feed index.
func (c *MessageCache) SetBatchChannelIDs(ctx context.Context, channelID fields.ID, messages []*channel.Message) error {
	if len(messages) == 0 {
		return nil
	}

	// 1. Save individual message DTOs via standard batch setter
	if err := c.SetBatch(ctx, messages); err != nil {
		return err
	}

	opCtx, cancel := context.WithTimeout(ctx, ScopeBatchTimeout)
	defer cancel()

	pipe := c.client.Pipeline()
	setKey := channelMessagesKey(channelID)

	zMembers := make([]redisdriver.Z, len(messages))
	for i, msg := range messages {
		if msg == nil {
			continue
		}
		score := float64(msg.CreatedAt().Time().UnixNano())
		zMembers[i] = redisdriver.Z{
			Score:  score,
			Member: msg.ID().String(),
		}
	}

	pipe.ZAdd(opCtx, setKey, zMembers...)
	pipe.Expire(opCtx, setKey, c.ttl)

	if _, err := pipe.Exec(opCtx); err != nil {
		return redis.NewError(err, redis.ScopeMessage)
	}

	return nil
}
