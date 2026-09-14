package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"time"

	"bonfire-api/internal/channel"
	"bonfire-api/internal/redis"

	"github.com/google/uuid"
	redisdriver "github.com/redis/go-redis/v9"
)

const (
	messageTTL        = 24 * time.Hour
	channelHistoryMax = 200 // Max messages per channel ring buffer
)

type MessageCache struct {
	client redisdriver.Cmdable
}

func NewMessageCache(client redisdriver.Cmdable) *MessageCache {
	return &MessageCache{
		client: client,
	}
}

func (c *MessageCache) Get(ctx context.Context, id uuid.UUID) (*channel.Message, error) {
	return getAndUnmarshal(ctx, c.client, messageKey(id), redis.ScopeMessage, unmarshalMessage)
}

// Set writes a single message object and updates its channel message index ring buffer.
func (c *MessageCache) Set(ctx context.Context, msg *channel.Message) error {
	dto := ParseMessage(msg)
	msgBytes, err := json.Marshal(dto)
	if err != nil {
		return err
	}

	pipe := c.client.Pipeline()

	// 1. Store serialized message object with a TTL
	pipe.Set(ctx, messageKey(msg.ID), msgBytes, messageTTL)

	// 2. Add message ID to the channel's ZSet scored by its UUIDv7 timestamp
	score := float64(msg.CreatedAt.UnixMilli())
	pipe.ZAdd(ctx, channelMessagesKey(msg.ChannelID), redisdriver.Z{
		Score:  score,
		Member: msg.ID.String(),
	})

	// 3. Trim channel ring buffer to keep only the newest channelHistoryMax entries
	pipe.ZRemRangeByRank(ctx, channelMessagesKey(msg.ChannelID), 0, -int64(channelHistoryMax+1))

	_, err = pipe.Exec(ctx)
	if err != nil {
		return redis.NewError(err, redis.ScopeMessage)
	}

	return nil
}

// Delete evicts a single message key and removes it from its channel message ZSet index.
func (c *MessageCache) Delete(ctx context.Context, channelID, msgID uuid.UUID) error {
	pipe := c.client.Pipeline()
	pipe.Del(ctx, messageKey(msgID))
	pipe.ZRem(ctx, channelMessagesKey(channelID), msgID.String())

	if _, err := pipe.Exec(ctx); err != nil {
		return redis.NewError(err, redis.ScopeMessage)
	}
	return nil
}

// GetRecentByChannelID retrieves the latest `limit` messages for a given channel in descending chronological order.
// Returns (messages, hit, error). Hit is false if the channel index is empty or has a missing message payload.
func (c *MessageCache) GetRecentByChannelID(
	ctx context.Context,
	channelID uuid.UUID,
	limit int,
) ([]*channel.Message, bool, error) {
	if limit <= 0 {
		limit = 50
	}

	// 1. Get message IDs ordered newest to oldest
	msgIDStrs, err := c.client.ZRevRange(ctx, channelMessagesKey(channelID), 0, int64(limit-1)).Result()
	if err != nil || len(msgIDStrs) == 0 {
		return nil, false, err
	}

	// 2. Fetch all message payloads via pipeline
	pipe := c.client.Pipeline()
	cmds := make([]*redisdriver.StringCmd, len(msgIDStrs))
	for i, idStr := range msgIDStrs {
		cmds[i] = pipe.Get(ctx, "{message:"+idStr+"}")
	}

	if _, err := pipe.Exec(ctx); err != nil && !errors.Is(err, redisdriver.Nil) {
		return nil, false, redis.NewError(err, redis.ScopeMessage)
	}

	// 3. Unmarshal and assemble the results
	messages := make([]*channel.Message, 0, len(msgIDStrs))
	for _, cmd := range cmds {
		rawJSON, err := cmd.Result()
		if err != nil {
			// A single missing message payload indicates a cache inconsistency; trigger fallback
			return nil, false, nil
		}

		var dto Message
		if err := json.Unmarshal([]byte(rawJSON), &dto); err != nil {
			return nil, false, nil
		}

		messages = append(messages, dto.ToDomain())
	}

	return messages, true, nil
}

// SetBatch stores a batch of messages for a channel into Redis in a single atomic pipeline.
// Useful for backfilling the cache after a DB history fetch.
func (c *MessageCache) SetBatch(ctx context.Context, channelID uuid.UUID, messages []*channel.Message) error {
	if len(messages) == 0 {
		return nil
	}

	pipe := c.client.Pipeline()

	zEntries := make([]redisdriver.Z, 0, len(messages))
	for _, msg := range messages {
		dto := ParseMessage(msg)
		msgBytes, err := json.Marshal(dto)
		if err != nil {
			return err
		}

		pipe.Set(ctx, messageKey(msg.ID), msgBytes, messageTTL)

		zEntries = append(zEntries, redisdriver.Z{
			Score:  float64(msg.CreatedAt.UnixMilli()),
			Member: msg.ID.String(),
		})
	}

	// Bulk write index and trim
	pipe.ZAdd(ctx, channelMessagesKey(channelID), zEntries...)
	pipe.ZRemRangeByRank(ctx, channelMessagesKey(channelID), 0, -int64(channelHistoryMax+1))

	_, err := pipe.Exec(ctx)
	if err != nil {
		return redis.NewError(err, redis.ScopeMessage)
	}

	return nil
}

// GetAroundByChannelID fetches messages surrounding a cursor message from the channel ZSet index.
// Returns (messages, hasMoreBefore, hasMoreAfter, hit, error).
func (c *MessageCache) GetAroundByChannelID(
	ctx context.Context,
	channelID, cursorID uuid.UUID,
	beforeLimit, afterLimit int,
) ([]*channel.Message, bool, bool, bool, error) {
	// 1. Get score (timestamp in ms) of the target cursor message
	score, err := c.client.ZScore(ctx, channelMessagesKey(channelID), cursorID.String()).Result()
	if err != nil {
		// Cursor message not present in index -> Cache miss
		return nil, false, false, false, nil
	}

	// Fetch extra 1 item for each side to reliably compute hasMore flags
	beforeOffset := int64(beforeLimit + 1)
	afterOffset := int64(afterLimit + 1)

	// Fetch IDs before (older or equal to target score, descending order)
	beforeIDs, err := c.client.ZRevRangeByScore(ctx, channelMessagesKey(channelID), &redisdriver.ZRangeBy{
		Min:   "-inf",
		Max:   fmt.Sprintf("%f", score),
		Count: beforeOffset,
	}).Result()
	if err != nil {
		return nil, false, false, false, redis.NewError(err, redis.ScopeMessage)
	}

	// Fetch IDs after (newer than target score, ascending order)
	afterIDs, err := c.client.ZRangeByScore(ctx, channelMessagesKey(channelID), &redisdriver.ZRangeBy{
		Min:   fmt.Sprintf("(%f", score), // exclusive lower bound
		Max:   "+inf",
		Count: afterOffset,
	}).Result()
	if err != nil {
		return nil, false, false, false, redis.NewError(err, redis.ScopeMessage)
	}

	hasMoreBefore := len(beforeIDs) > beforeLimit
	if hasMoreBefore {
		beforeIDs = beforeIDs[:beforeLimit]
	}

	hasMoreAfter := len(afterIDs) > afterLimit
	if hasMoreAfter {
		afterIDs = afterIDs[:afterLimit]
	}

	// Reverse beforeIDs so the combined array is in ascending chronological order
	slices.Reverse(beforeIDs)

	// Combine ordered IDs: [older ... target ... newer]
	allIDs := append(beforeIDs, afterIDs...)
	if len(allIDs) == 0 {
		return nil, false, false, false, nil
	}

	// 2. MGET or Pipeline GET all message payloads
	pipe := c.client.Pipeline()
	cmds := make([]*redisdriver.StringCmd, len(allIDs))
	for i, idStr := range allIDs {
		cmds[i] = pipe.Get(ctx, "{message:"+idStr+"}")
	}

	if _, err := pipe.Exec(ctx); err != nil && !errors.Is(err, redisdriver.Nil) {
		return nil, false, false, false, redis.NewError(err, redis.ScopeMessage)
	}

	messages := make([]*channel.Message, 0, len(allIDs))
	for _, cmd := range cmds {
		rawJSON, err := cmd.Result()
		if err != nil {
			// Missing payload -> Inconsistent cache, fallback to DB
			return nil, false, false, false, nil
		}

		var dto Message
		if err := json.Unmarshal([]byte(rawJSON), &dto); err != nil {
			return nil, false, false, false, nil
		}

		messages = append(messages, dto.ToDomain())
	}

	return messages, hasMoreBefore, hasMoreAfter, true, nil
}

// GetBeforeByChannelID fetches messages preceding a cursor message from the channel ZSet index.
// Returns (messages, hasMoreBefore, hit, error).
func (c *MessageCache) GetBeforeByChannelID(
	ctx context.Context,
	channelID, cursorID uuid.UUID,
	limit int,
) ([]*channel.Message, bool, bool, error) {
	// Get timestamp score of the target cursor message
	score, err := c.client.ZScore(ctx, channelMessagesKey(channelID), cursorID.String()).Result()
	if err != nil {
		// Cursor message not in ring buffer -> Cache miss
		return nil, false, false, nil
	}

	// Exclusive upper bound: messages strictly older than the cursor
	ids, err := c.client.ZRevRangeByScore(ctx, channelMessagesKey(channelID), &redisdriver.ZRangeBy{
		Min:   "-inf",
		Max:   fmt.Sprintf("(%f", score), // exclusive
		Count: int64(limit + 1),
	}).Result()
	if err != nil {
		return nil, false, false, redis.NewError(err, redis.ScopeMessage)
	}

	hasMoreBefore := len(ids) > limit
	if hasMoreBefore {
		ids = ids[:limit]
	}

	if len(ids) == 0 {
		return []*channel.Message{}, false, true, nil
	}

	messages, err := c.fetchAndUnmarshalBatch(ctx, ids)
	if err != nil || messages == nil {
		return nil, false, false, err
	}

	// Reverse so returned array matches DB order (ascending chronological)
	slices.Reverse(messages)

	return messages, hasMoreBefore, true, nil
}

// GetAfterByChannelID fetches messages following a cursor message from the channel ZSet index.
// Returns (messages, hasMoreAfter, hit, error).
func (c *MessageCache) GetAfterByChannelID(
	ctx context.Context,
	channelID, cursorID uuid.UUID,
	limit int,
) ([]*channel.Message, bool, bool, error) {
	// Get timestamp score of the target cursor message
	score, err := c.client.ZScore(ctx, channelMessagesKey(channelID), cursorID.String()).Result()
	if err != nil {
		// Cursor message not in ring buffer -> Cache miss
		return nil, false, false, nil
	}

	// Exclusive lower bound: messages strictly newer than the cursor
	ids, err := c.client.ZRangeByScore(ctx, channelMessagesKey(channelID), &redisdriver.ZRangeBy{
		Min:   fmt.Sprintf("(%f", score), // exclusive
		Max:   "+inf",
		Count: int64(limit + 1),
	}).Result()
	if err != nil {
		return nil, false, false, redis.NewError(err, redis.ScopeMessage)
	}

	hasMoreAfter := len(ids) > limit
	if hasMoreAfter {
		ids = ids[:limit]
	}

	if len(ids) == 0 {
		return []*channel.Message{}, false, true, nil
	}

	messages, err := c.fetchAndUnmarshalBatch(ctx, ids)
	if err != nil || messages == nil {
		return nil, false, false, err
	}

	return messages, hasMoreAfter, true, nil
}

// Helper to keep payload fetching DRY across GetRecent, GetAround, GetBefore, and GetAfter
func (c *MessageCache) fetchAndUnmarshalBatch(ctx context.Context, ids []string) ([]*channel.Message, error) {
	pipe := c.client.Pipeline()
	cmds := make([]*redisdriver.StringCmd, len(ids))
	for i, idStr := range ids {
		cmds[i] = pipe.Get(ctx, "{message:"+idStr+"}")
	}

	if _, err := pipe.Exec(ctx); err != nil && !errors.Is(err, redisdriver.Nil) {
		return nil, redis.NewError(err, redis.ScopeMessage)
	}

	messages := make([]*channel.Message, 0, len(ids))
	for _, cmd := range cmds {
		rawJSON, err := cmd.Result()
		if err != nil {
			return nil, nil // Payload missing -> Trigger cache fallback
		}

		var dto Message
		if err := json.Unmarshal([]byte(rawJSON), &dto); err != nil {
			return nil, nil
		}

		messages = append(messages, dto.ToDomain())
	}

	return messages, nil
}
