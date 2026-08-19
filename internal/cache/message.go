package cache

import (
	"context"
	"encoding/json"
	"slices"
	"strconv"
	"time"

	"bonfire-api/internal/channel"
	"bonfire-api/internal/errs"
	"bonfire-api/internal/fields"
	"bonfire-api/internal/redis"

	"github.com/google/uuid"
	redisdriver "github.com/redis/go-redis/v9"
)

type MessageCache struct {
	*KeyCache[fields.ID, Message]
	client redisdriver.Cmdable
	ttl    time.Duration
}

func NewMessageCache(client redisdriver.Cmdable, ttl time.Duration) *MessageCache {
	return &MessageCache{
		KeyCache: NewKeyCache[fields.ID, Message](client, redis.ScopeMessage, MessageKey),
		client:   client,
		ttl:      ttl,
	}
}

func (c *MessageCache) IsTimelineComplete(ctx context.Context, channelID fields.ID) bool {
	val, err := c.client.Exists(ctx, ChannelLoadedKey(channelID)).Result()
	return err == nil && val > 0
}

func (c *MessageCache) SetTimelineComplete(ctx context.Context, channelID fields.ID) error {
	key := ChannelLoadedKey(channelID)
	if err := c.client.Set(ctx, key, "1", c.ttl).Err(); err != nil {
		return redis.NewError(err, redis.ScopeMessage)
	}
	return nil
}

func (c *MessageCache) Get(ctx context.Context, id fields.ID) (*channel.Message, error) {
	dto, err := c.KeyCache.Get(ctx, id)
	if err != nil || dto == nil {
		return nil, err
	}
	return dto.ToDomain()
}

func (c *MessageCache) GetBatch(
	ctx context.Context,
	ids []fields.ID,
) (map[fields.ID]*channel.Message, []fields.ID, error) {
	dtos, missing, err := c.KeyCache.GetBatch(ctx, ids)
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

	bytes, err := json.Marshal(dto)
	if err != nil {
		return errs.Internal("Failed to marshal cached json.").
			Meta("scope", redis.ScopeMessage.String()).
			Wrap(err)
	}

	pipe := c.client.Pipeline()

	// 1. Store DTO
	pipe.Set(ctx, MessageKey(msg.ID()), bytes, c.ttl)

	// 2. Add to Timeline ZSET (Score = Unix Microseconds for precision)
	zKey := ChannelMessageIDsKey(msg.ChannelID())
	score := float64(msg.CreatedAt().Time().UnixMicro())
	pipe.ZAdd(ctx, zKey, redisdriver.Z{
		Score:  score,
		Member: msg.ID().String(),
	})
	pipe.Expire(ctx, zKey, c.ttl)

	// Touch timeline completeness flag TTL if active
	loadedKey := ChannelLoadedKey(msg.ChannelID())
	pipe.Expire(ctx, loadedKey, c.ttl)

	if _, err := pipe.Exec(ctx); err != nil {
		return redis.NewError(err, redis.ScopeMessage)
	}

	return nil
}

func (c *MessageCache) SetBatch(ctx context.Context, messages []*channel.Message) error {
	if len(messages) == 0 {
		return nil
	}

	dtos := make(map[fields.ID]Message, len(messages))
	channelZMap := make(map[fields.ID][]redisdriver.Z)

	for _, msg := range messages {
		if msg == nil {
			continue
		}

		dto, err := ParseMessage(msg)
		if err != nil {
			return err
		}

		cid := msg.ChannelID()
		dtos[msg.ID()] = dto

		channelZMap[cid] = append(channelZMap[cid], redisdriver.Z{
			Score:  float64(msg.CreatedAt().Time().UnixMicro()),
			Member: msg.ID().String(),
		})
	}

	if err := c.KeyCache.SetBatch(ctx, dtos, c.ttl); err != nil {
		return err
	}

	pipe := c.client.Pipeline()
	for cid, zMembers := range channelZMap {
		zKey := ChannelMessageIDsKey(cid)
		pipe.ZAdd(ctx, zKey, zMembers...)
		pipe.Expire(ctx, zKey, c.ttl)

		loadedKey := ChannelLoadedKey(cid)
		pipe.Expire(ctx, loadedKey, c.ttl)
	}

	if _, err := pipe.Exec(ctx); err != nil {
		return redis.NewError(err, redis.ScopeMessage)
	}

	return nil
}

// ListBeforeByChannelID uses a single Redis Pipeline to check score and fetch range.
func (c *MessageCache) ListBeforeByChannelID(
	ctx context.Context,
	channelID, cursorID fields.ID,
	limit int32,
) ([]*channel.Message, error) {
	zKey := ChannelMessageIDsKey(channelID)
	loadedKey := ChannelLoadedKey(channelID)

	// Single round-trip pipeline
	pipe := c.client.Pipeline()
	scoreCmd := pipe.ZScore(ctx, zKey, cursorID.String())
	loadedCmd := pipe.Exists(ctx, loadedKey)
	_, _ = pipe.Exec(ctx)

	anchorScore, err := scoreCmd.Result()
	if err != nil {
		return nil, nil // Cursor not in ZSET -> Cache Miss
	}

	isComplete := loadedCmd.Val() > 0

	rawIDs, err := c.client.ZRevRangeByScore(ctx, zKey, &redisdriver.ZRangeBy{
		Max:    "(" + scoreToString(anchorScore),
		Min:    "-inf",
		Offset: 0,
		Count:  int64(limit),
	}).Result()
	if err != nil {
		return nil, err
	}

	if int32(len(rawIDs)) < limit && !isComplete {
		return nil, nil // Partial result on incomplete cache -> Fallback to DB
	}

	if len(rawIDs) == 0 {
		return []*channel.Message{}, nil
	}

	return c.hydrateFromIDs(ctx, rawIDs)
}

// ListAfterByChannelID fetches newer messages relative to cursorID using a single pipeline.
func (c *MessageCache) ListAfterByChannelID(
	ctx context.Context,
	channelID, cursorID fields.ID,
	limit int32,
) ([]*channel.Message, error) {
	zKey := ChannelMessageIDsKey(channelID)
	loadedKey := ChannelLoadedKey(channelID)

	pipe := c.client.Pipeline()
	scoreCmd := pipe.ZScore(ctx, zKey, cursorID.String())
	loadedCmd := pipe.Exists(ctx, loadedKey)
	_, _ = pipe.Exec(ctx)

	anchorScore, err := scoreCmd.Result()
	if err != nil {
		return nil, nil // Cursor not in ZSET -> Cache Miss
	}

	isComplete := loadedCmd.Val() > 0

	rawIDs, err := c.client.ZRangeByScore(ctx, zKey, &redisdriver.ZRangeBy{
		Min:    "(" + scoreToString(anchorScore),
		Max:    "+inf",
		Offset: 0,
		Count:  int64(limit),
	}).Result()
	if err != nil {
		return nil, err
	}

	if int32(len(rawIDs)) < limit && !isComplete {
		return nil, nil // Partial result on incomplete cache -> Fallback to DB
	}

	if len(rawIDs) == 0 {
		return []*channel.Message{}, nil
	}

	return c.hydrateFromIDs(ctx, rawIDs)
}

// ListAroundByChannelID fetches surrounding messages in a single pipelined operation.
func (c *MessageCache) ListAroundByChannelID(
	ctx context.Context,
	channelID, anchorMessageID fields.ID,
	beforeLimit, afterLimit int32,
) ([]*channel.Message, error) {
	zKey := ChannelMessageIDsKey(channelID)
	loadedKey := ChannelLoadedKey(channelID)

	// Step 1: Pipeline Anchor Score + Complete Check
	pipe := c.client.Pipeline()
	scoreCmd := pipe.ZScore(ctx, zKey, anchorMessageID.String())
	loadedCmd := pipe.Exists(ctx, loadedKey)
	_, _ = pipe.Exec(ctx)

	anchorScore, err := scoreCmd.Result()
	if err != nil {
		return nil, nil // Anchor missing from ZSET -> Cache Miss
	}

	isComplete := loadedCmd.Val() > 0

	// Step 2: Pipeline Both Range Queries
	pipeRange := c.client.Pipeline()
	beforeCmd := pipeRange.ZRevRangeByScore(ctx, zKey, &redisdriver.ZRangeBy{
		Max:    "(" + scoreToString(anchorScore),
		Min:    "-inf",
		Offset: 0,
		Count:  int64(beforeLimit),
	})
	afterCmd := pipeRange.ZRangeByScore(ctx, zKey, &redisdriver.ZRangeBy{
		Min:    scoreToString(anchorScore),
		Max:    "+inf",
		Offset: 0,
		Count:  int64(afterLimit + 1),
	})
	if _, err := pipeRange.Exec(ctx); err != nil {
		return nil, err
	}

	beforeIDs := beforeCmd.Val()
	afterIDs := afterCmd.Val()

	// Validate incomplete windows
	if (int32(len(beforeIDs)) < beforeLimit || int32(len(afterIDs)) < (afterLimit+1)) && !isComplete {
		return nil, nil // Incomplete window -> Fallback to DB
	}

	slices.Reverse(beforeIDs)
	combinedIDs := append(beforeIDs, afterIDs...)

	if len(combinedIDs) == 0 {
		return []*channel.Message{}, nil
	}

	return c.hydrateFromIDs(ctx, combinedIDs)
}

func (c *MessageCache) Delete(ctx context.Context, channelID, messageID fields.ID) error {
	pipe := c.client.Pipeline()

	zKey := ChannelMessageIDsKey(channelID)
	msgKey := MessageKey(messageID)

	pipe.ZRem(ctx, zKey, messageID.String())
	pipe.Del(ctx, msgKey)

	if _, err := pipe.Exec(ctx); err != nil {
		return redis.NewError(err, redis.ScopeMessage)
	}

	return nil
}

func scoreToString(score float64) string {
	return strconv.FormatFloat(score, 'f', -1, 64)
}

func (c *MessageCache) hydrateFromIDs(ctx context.Context, rawIDs []string) ([]*channel.Message, error) {
	msgIDs := make([]fields.ID, 0, len(rawIDs))
	for _, raw := range rawIDs {
		parsedUUID, err := uuid.Parse(raw)
		if err != nil {
			return nil, nil
		}
		id, err := fields.ParseRequiredID("message_id", parsedUUID)
		if err != nil {
			return nil, nil
		}
		msgIDs = append(msgIDs, id)
	}

	dtosMap, missing, err := c.GetBatch(ctx, msgIDs)
	if err != nil || len(missing) > 0 {
		return nil, nil // DTO missing -> Fallback to DB
	}

	ordered := make([]*channel.Message, 0, len(msgIDs))
	for _, id := range msgIDs {
		msg, ok := dtosMap[id]
		if !ok || msg == nil {
			return nil, nil
		}
		ordered = append(ordered, msg)
	}

	return ordered, nil
}
