package cache

import (
	"context"
	"strconv"
	"time"

	"bonfire-api/internal/fields"
	"bonfire-api/internal/redis"

	redisdriver "github.com/redis/go-redis/v9"
)

const loadedField = "_loaded"

type ReactionCache struct {
	client redisdriver.Cmdable
	ttl    time.Duration
}

func NewReactionCache(client redisdriver.Cmdable, ttl time.Duration) *ReactionCache {
	return &ReactionCache{
		client: client,
		ttl:    ttl,
	}
}

// Get reads aggregate emoji counts for a single message.
func (c *ReactionCache) Get(
	ctx context.Context,
	messageID fields.ID,
) (map[string]int, bool, error) {
	key := MessageReactionsKey(messageID)
	data, err := c.client.HGetAll(ctx, key).Result()
	if err == redisdriver.Nil {
		return nil, false, nil
	} else if err != nil {
		return nil, false, redis.NewError(err, redis.ScopeMessage)
	}

	if len(data) == 0 || data[loadedField] != "1" {
		return nil, false, nil
	}

	counts := make(map[string]int, len(data)-1)
	for emoji, countStr := range data {
		if emoji == loadedField {
			continue
		}
		if cnt, parseErr := strconv.Atoi(countStr); parseErr == nil {
			counts[emoji] = cnt
		}
	}

	return counts, true, nil
}

// GetBatch reads aggregate emoji counts for multiple messages in one pipeline round-trip.
func (c *ReactionCache) GetBatch(
	ctx context.Context,
	messageIDs []fields.ID,
) (hits map[fields.ID]map[string]int, misses []fields.ID, err error) {
	if len(messageIDs) == 0 {
		return make(map[fields.ID]map[string]int), nil, nil
	}

	pipe := c.client.Pipeline()
	cmds := make([]*redisdriver.MapStringStringCmd, len(messageIDs))

	for i, id := range messageIDs {
		cmds[i] = pipe.HGetAll(ctx, MessageReactionsKey(id))
	}

	if _, err := pipe.Exec(ctx); err != nil && err != redisdriver.Nil {
		return nil, messageIDs, redis.NewError(err, redis.ScopeMessage)
	}

	hits = make(map[fields.ID]map[string]int, len(messageIDs))

	for i, id := range messageIDs {
		data := cmds[i].Val()

		// Missing key or no loaded sentinel = Cache Miss
		if len(data) == 0 || data[loadedField] != "1" {
			misses = append(misses, id)
			continue
		}

		counts := make(map[string]int, len(data)-1)
		for emoji, countStr := range data {
			if emoji == loadedField {
				continue
			}
			if cnt, parseErr := strconv.Atoi(countStr); parseErr == nil {
				counts[emoji] = cnt
			}
		}
		hits[id] = counts
	}

	return hits, misses, nil
}

// Set writes aggregate emoji counts for a single message.
func (c *ReactionCache) Set(
	ctx context.Context,
	messageID fields.ID,
	counts map[string]int,
) error {
	return c.SetBatch(ctx, map[fields.ID]map[string]int{messageID: counts}, []fields.ID{messageID})
}

// SetBatch backfills aggregate emoji counts for multiple messages.
func (c *ReactionCache) SetBatch(
	ctx context.Context,
	countsMap map[fields.ID]map[string]int,
	missedIDs []fields.ID,
) error {
	if len(missedIDs) == 0 {
		return nil
	}

	pipe := c.client.Pipeline()

	for _, id := range missedIDs {
		key := MessageReactionsKey(id)
		counts := countsMap[id]

		hashData := map[string]any{
			loadedField: "1", // Sentinel value guarantees presence even with 0 total reactions
		}
		for emoji, count := range counts {
			hashData[emoji] = strconv.Itoa(count)
		}

		pipe.Del(ctx, key)
		pipe.HSet(ctx, key, hashData)
		pipe.Expire(ctx, key, c.ttl)
	}

	if _, err := pipe.Exec(ctx); err != nil {
		return redis.NewError(err, redis.ScopeMessage)
	}

	return nil
}

// IncrementEmoji invalidates cache entry so read path fetches clean DB state.
func (c *ReactionCache) IncrementEmoji(ctx context.Context, messageID fields.ID, _ string) error {
	return c.Delete(ctx, messageID)
}

// DecrementEmoji invalidates cache entry so read path fetches clean DB state.
func (c *ReactionCache) DecrementEmoji(ctx context.Context, messageID fields.ID, _ string) error {
	return c.Delete(ctx, messageID)
}

// Delete invalidates the reaction cache entry for a message.
func (c *ReactionCache) Delete(ctx context.Context, messageID fields.ID) error {
	key := MessageReactionsKey(messageID)
	if err := c.client.Del(ctx, key).Err(); err != nil {
		return redis.NewError(err, redis.ScopeMessage)
	}
	return nil
}
