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
	client redisdriver.Cmdable
	scope  redis.Scope
	ttl    time.Duration
}

func NewMessageCache(
	client redisdriver.Cmdable,
	scope redis.Scope,
	ttl time.Duration,
) *MessageCache {
	return &MessageCache{
		client: client,
		scope:  scope,
		ttl:    ttl,
	}
}

// Get fetches a single message payload by ID.
func (c *MessageCache) Get(ctx context.Context, id fields.ID) (*channel.Message, error) {
	data, err := c.client.Get(ctx, MessageKey(id)).Bytes()
	if redis.IsCacheMiss(err) {
		return nil, nil
	}
	if err != nil {
		return nil, redis.NewError(err, c.scope)
	}

	var dto Message
	if err := json.Unmarshal(data, &dto); err != nil {
		return nil, errs.Internal("Failed to unmarshal cached message.").
			Meta("scope", c.scope.String()).
			Wrap(err)
	}

	return dto.ToDomain()
}

// GetBatch retrieves multiple message payloads using MGET in chunks.
func (c *MessageCache) GetBatch(
	ctx context.Context,
	ids []fields.ID,
) (map[fields.ID]*channel.Message, []fields.ID, error) {
	found := make(map[fields.ID]*channel.Message, len(ids))
	var missing []fields.ID

	for i := 0; i < len(ids); i += KeyMaxBatchSize {
		if err := ctx.Err(); err != nil {
			return nil, nil, err
		}

		end := min(i+KeyMaxBatchSize, len(ids))
		chunk := ids[i:end]

		redisKeys := make([]string, len(chunk))
		for j, id := range chunk {
			redisKeys[j] = MessageKey(id)
		}

		vals, err := c.client.MGet(ctx, redisKeys...).Result()
		if err != nil {
			return nil, nil, redis.NewError(err, c.scope)
		}

		for j, raw := range vals {
			id := chunk[j]
			data, ok := parseRawBytes(raw)
			if !ok {
				missing = append(missing, id)
				continue
			}

			var dto Message
			if err := json.Unmarshal(data, &dto); err != nil {
				return nil, nil, errs.Internal("Failed to unmarshal cached message batch item.").
					Meta("scope", c.scope.String()).
					Wrap(err)
			}

			msg, err := dto.ToDomain()
			if err != nil {
				missing = append(missing, id)
				continue
			}

			found[id] = msg
		}
	}

	if len(missing) > 0 {
		missing = deduplicateMissing(missing, found)
	}

	return found, missing, nil
}

// Set writes a single message payload and updates the channel index if warm.
func (c *MessageCache) Set(ctx context.Context, msg *channel.Message) error {
	if msg == nil {
		return nil
	}

	return c.SetBatch(ctx, []*channel.Message{msg})
}

// SetBatch safely sets message payloads and appends them to warm ZSET channel indexes.
func (c *MessageCache) SetBatch(ctx context.Context, messages []*channel.Message) error {
	validMessages := make([]*channel.Message, 0, len(messages))
	for _, m := range messages {
		if m != nil {
			validMessages = append(validMessages, m)
		}
	}

	if len(validMessages) == 0 {
		return nil
	}

	for i := 0; i < len(validMessages); i += KeyMaxBatchSize {
		if err := ctx.Err(); err != nil {
			return err
		}

		end := min(i+KeyMaxBatchSize, len(validMessages))
		chunk := validMessages[i:end]

		channelZMap := make(map[fields.ID][]redisdriver.Z)
		pipe := c.client.Pipeline()

		for _, msg := range chunk {
			dto, err := ParseMessage(msg)
			if err != nil {
				return err
			}

			bytes, err := json.Marshal(dto)
			if err != nil {
				return errs.Internal("Failed to marshal message json.").
					Meta("scope", c.scope.String()).
					Wrap(err)
			}

			pipe.Set(ctx, MessageKey(msg.ID()), bytes, c.ttl)

			cid := msg.ChannelID()
			channelZMap[cid] = append(channelZMap[cid], redisdriver.Z{
				Score:  float64(msg.CreatedAt().Time().UnixMicro()),
				Member: msg.ID().String(),
			})
		}

		for cid, zMembers := range channelZMap {
			zKey := ChannelMessageIDsKey(cid)
			pipe.ZAddXX(ctx, zKey, zMembers...)
			pipe.Expire(ctx, zKey, c.ttl)
			pipe.Del(ctx, ChannelLoadedKey(cid))
		}

		if _, err := pipe.Exec(ctx); err != nil {
			return redis.NewError(err, c.scope)
		}
	}

	return nil
}

// ListBeforeByChannelID retrieves messages older than cursorID.
func (c *MessageCache) ListBeforeByChannelID(
	ctx context.Context,
	channelID, cursorID fields.ID,
	limit int32,
) ([]*channel.Message, error) {
	zKey := ChannelMessageIDsKey(channelID)
	loadedKey := ChannelLoadedKey(channelID)

	anchorScore, err := c.client.ZScore(ctx, zKey, cursorID.String()).Result()
	if err != nil {
		return nil, nil // Cursor missing from index -> Cache Miss
	}

	pipe := c.client.Pipeline()
	loadedCmd := pipe.Exists(ctx, loadedKey)
	rangeCmd := pipe.ZRevRangeByScore(ctx, zKey, &redisdriver.ZRangeBy{
		Max:    "(" + scoreToString(anchorScore),
		Min:    "-inf",
		Offset: 0,
		Count:  int64(limit),
	})
	if _, err := pipe.Exec(ctx); err != nil {
		return nil, redis.NewError(err, c.scope)
	}

	rawIDs := rangeCmd.Val()
	isComplete := loadedCmd.Val() > 0

	if int32(len(rawIDs)) < limit && !isComplete {
		return nil, nil // Incomplete index range -> Cache Miss
	}

	if len(rawIDs) == 0 {
		return []*channel.Message{}, nil
	}

	return c.hydrateFromIDs(ctx, rawIDs)
}

// ListAfterByChannelID retrieves messages newer than cursorID.
func (c *MessageCache) ListAfterByChannelID(
	ctx context.Context,
	channelID, cursorID fields.ID,
	limit int32,
) ([]*channel.Message, error) {
	zKey := ChannelMessageIDsKey(channelID)
	loadedKey := ChannelLoadedKey(channelID)

	anchorScore, err := c.client.ZScore(ctx, zKey, cursorID.String()).Result()
	if err != nil {
		return nil, nil // Cursor missing from index -> Cache Miss
	}

	pipe := c.client.Pipeline()
	loadedCmd := pipe.Exists(ctx, loadedKey)
	rangeCmd := pipe.ZRangeByScore(ctx, zKey, &redisdriver.ZRangeBy{
		Min:    "(" + scoreToString(anchorScore),
		Max:    "+inf",
		Offset: 0,
		Count:  int64(limit),
	})
	if _, err := pipe.Exec(ctx); err != nil {
		return nil, redis.NewError(err, c.scope)
	}

	rawIDs := rangeCmd.Val()
	isComplete := loadedCmd.Val() > 0

	if int32(len(rawIDs)) < limit && !isComplete {
		return nil, nil // Incomplete index range -> Cache Miss
	}

	if len(rawIDs) == 0 {
		return []*channel.Message{}, nil
	}

	return c.hydrateFromIDs(ctx, rawIDs)
}

// ListAroundByChannelID fetches messages surrounding an anchor ID (both older and newer).
func (c *MessageCache) ListAroundByChannelID(
	ctx context.Context,
	channelID, anchorMessageID fields.ID,
	beforeLimit, afterLimit int32,
) ([]*channel.Message, error) {
	zKey := ChannelMessageIDsKey(channelID)
	loadedKey := ChannelLoadedKey(channelID)

	pipeAnchor := c.client.Pipeline()
	scoreCmd := pipeAnchor.ZScore(ctx, zKey, anchorMessageID.String())
	loadedCmd := pipeAnchor.Exists(ctx, loadedKey)
	if _, err := pipeAnchor.Exec(ctx); err != nil && err != redisdriver.Nil {
		return nil, redis.NewError(err, c.scope)
	}

	anchorScore, err := scoreCmd.Result()
	if err != nil {
		return nil, nil // Anchor missing -> Cache Miss
	}

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
		return nil, redis.NewError(err, c.scope)
	}

	beforeIDs := beforeCmd.Val()
	afterIDs := afterCmd.Val()
	isComplete := loadedCmd.Val() > 0

	if int32(len(beforeIDs)) < beforeLimit && !isComplete {
		return nil, nil // Incomplete index range -> Cache Miss
	}

	// Merge into ascending sequence
	slices.Reverse(beforeIDs)
	combinedIDs := make([]string, 0, len(beforeIDs)+len(afterIDs))
	combinedIDs = append(combinedIDs, beforeIDs...)

	for _, id := range afterIDs {
		if len(combinedIDs) == 0 || combinedIDs[len(combinedIDs)-1] != id {
			combinedIDs = append(combinedIDs, id)
		}
	}

	if len(combinedIDs) == 0 {
		return []*channel.Message{}, nil
	}

	return c.hydrateFromIDs(ctx, combinedIDs)
}

// Delete removes a message payload and removes it from the channel index.
func (c *MessageCache) Delete(ctx context.Context, channelID, messageID fields.ID) error {
	pipe := c.client.Pipeline()
	pipe.ZRem(ctx, ChannelMessageIDsKey(channelID), messageID.String())
	pipe.Del(ctx, MessageKey(messageID))

	if _, err := pipe.Exec(ctx); err != nil {
		return redis.NewError(err, c.scope)
	}

	return nil
}

// DeleteBatch removes multiple message payloads and index entries.
func (c *MessageCache) DeleteBatch(ctx context.Context, channelID fields.ID, keys []fields.ID) error {
	if len(keys) == 0 {
		return nil
	}

	for i := 0; i < len(keys); i += KeyMaxBatchSize {
		if err := ctx.Err(); err != nil {
			return err
		}

		end := min(i+KeyMaxBatchSize, len(keys))
		chunk := keys[i:end]

		pipe := c.client.Pipeline()
		zKey := ChannelMessageIDsKey(channelID)

		members := make([]any, len(chunk))
		stringKeys := make([]string, len(chunk))
		for j, key := range chunk {
			members[j] = key.String()
			stringKeys[j] = MessageKey(key)
		}

		pipe.ZRem(ctx, zKey, members...)
		pipe.Del(ctx, stringKeys...)

		if _, err := pipe.Exec(ctx); err != nil {
			return redis.NewError(err, c.scope)
		}
	}

	return nil
}

// Helpers

func (c *MessageCache) hydrateFromIDs(ctx context.Context, rawIDs []string) ([]*channel.Message, error) {
	msgIDs := make([]fields.ID, 0, len(rawIDs))
	for _, raw := range rawIDs {
		parsedUUID, err := uuid.Parse(raw)
		if err != nil {
			return nil, nil // Corrupted cache entry -> Fallback to DB
		}
		id, err := fields.ParseRequiredID("message_id", parsedUUID)
		if err != nil {
			return nil, nil // Corrupted cache entry -> Fallback to DB
		}
		msgIDs = append(msgIDs, id)
	}

	dtosMap, missing, err := c.GetBatch(ctx, msgIDs)
	if err != nil || len(missing) > 0 {
		return nil, nil // Payload miss -> Fallback to DB
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

func parseRawBytes(raw any) ([]byte, bool) {
	switch v := raw.(type) {
	case string:
		if v == "" {
			return nil, false
		}
		return []byte(v), true
	case []byte:
		if len(v) == 0 {
			return nil, false
		}
		return v, true
	default:
		return nil, false
	}
}

func deduplicateMissing(missing []fields.ID, found map[fields.ID]*channel.Message) []fields.ID {
	seen := make(map[fields.ID]struct{}, len(missing))
	result := make([]fields.ID, 0, len(missing))

	for _, id := range missing {
		if _, inFound := found[id]; inFound {
			continue
		}
		if _, inSeen := seen[id]; !inSeen {
			seen[id] = struct{}{}
			result = append(result, id)
		}
	}
	return result
}

func scoreToString(score float64) string {
	return strconv.FormatFloat(score, 'f', -1, 64)
}
