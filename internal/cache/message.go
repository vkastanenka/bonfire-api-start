package cache

import (
	"context"
	"errors"
	"slices"
	"strconv"
	"time"

	"bonfire-api/internal/channel"
	"bonfire-api/internal/redis"

	"github.com/google/uuid"
	redisdriver "github.com/redis/go-redis/v9"
)

const (
	messageTTL        = 24 * time.Hour
	channelHistoryMax = 200
)

func messageKey(messageID uuid.UUID) string {
	return "{message:" + messageID.String() + "}"
}

func channelMessagesKey(channelID uuid.UUID) string {
	return "{channel:" + channelID.String() + "}:messages"
}

type MessageCache struct {
	client redisdriver.Cmdable
}

func NewMessageCache(client redisdriver.Cmdable) *MessageCache {
	return &MessageCache{
		client: client,
	}
}

func (c *MessageCache) Get(ctx context.Context, id uuid.UUID) (*channel.Message, error) {
	return getAndUnmarshal(ctx, redis.ScopeMessage, c.client, messageKey(id), unmarshalMessage)
}

func (c *MessageCache) Set(ctx context.Context, msg *channel.Message) error {
	if msg == nil || msg.ID == uuid.Nil {
		return nil
	}

	msgBytes, err := marshalMessage(msg)
	if err != nil {
		return err
	}

	pipe := c.client.Pipeline()
	pipe.Set(ctx, messageKey(msg.ID), msgBytes, messageTTL)
	pipe.ZAdd(ctx, channelMessagesKey(msg.ChannelID), redisdriver.Z{
		Score:  messageScore(msg.CreatedAt),
		Member: msg.ID.String(),
	})
	pipe.ZRemRangeByRank(ctx, channelMessagesKey(msg.ChannelID), 0, -int64(channelHistoryMax+1))

	if _, err := pipe.Exec(ctx); err != nil {
		return redis.NewError(err, redis.ScopeMessage)
	}

	return nil
}

func (c *MessageCache) SetBatch(ctx context.Context, channelID uuid.UUID, messages []*channel.Message) error {
	if channelID == uuid.Nil || len(messages) == 0 {
		return nil
	}

	pipe := c.client.Pipeline()
	zEntries := make([]redisdriver.Z, 0, len(messages))

	for _, msg := range messages {
		if msg == nil || msg.ID == uuid.Nil {
			continue
		}

		msgBytes, err := marshalMessage(msg)
		if err != nil {
			return err
		}

		pipe.Set(ctx, messageKey(msg.ID), msgBytes, messageTTL)
		zEntries = append(zEntries, redisdriver.Z{
			Score:  messageScore(msg.CreatedAt),
			Member: msg.ID.String(),
		})
	}

	if len(zEntries) > 0 {
		pipe.ZAdd(ctx, channelMessagesKey(channelID), zEntries...)
		pipe.ZRemRangeByRank(ctx, channelMessagesKey(channelID), 0, -int64(channelHistoryMax+1))
	}

	if _, err := pipe.Exec(ctx); err != nil {
		return redis.NewError(err, redis.ScopeMessage)
	}

	return nil
}

func (c *MessageCache) Delete(ctx context.Context, channelID, msgID uuid.UUID) error {
	if channelID == uuid.Nil || msgID == uuid.Nil {
		return nil
	}

	pipe := c.client.Pipeline()
	pipe.Del(ctx, messageKey(msgID))
	pipe.ZRem(ctx, channelMessagesKey(channelID), msgID.String())

	if _, err := pipe.Exec(ctx); err != nil {
		return redis.NewError(err, redis.ScopeMessage)
	}
	return nil
}

func (c *MessageCache) GetRecentByChannelID(
	ctx context.Context,
	channelID uuid.UUID,
	limit int,
) ([]*channel.Message, bool, error) {
	if channelID == uuid.Nil {
		return nil, false, nil
	}
	if limit <= 0 {
		limit = 50
	}

	msgIDStrs, err := c.client.ZRevRange(ctx, channelMessagesKey(channelID), 0, int64(limit-1)).Result()
	if err != nil {
		return nil, false, redis.NewError(err, redis.ScopeMessage)
	}
	if len(msgIDStrs) == 0 {
		return nil, false, nil
	}

	messages, err := c.fetchAndUnmarshalBatch(ctx, msgIDStrs)
	if err != nil || messages == nil {
		return nil, false, err
	}

	return messages, true, nil
}

func (c *MessageCache) GetBeforeByChannelID(
	ctx context.Context,
	channelID, cursorID uuid.UUID,
	limit int,
) ([]*channel.Message, bool, bool, error) {
	if channelID == uuid.Nil || cursorID == uuid.Nil {
		return nil, false, false, nil
	}

	score, found, err := c.fetchCursorScore(ctx, channelID, cursorID)
	if err != nil || !found {
		return nil, false, false, err
	}

	ids, err := c.client.ZRevRangeByScore(ctx, channelMessagesKey(channelID), &redisdriver.ZRangeBy{
		Min:   "-inf",
		Max:   formatScoreBound(score, true),
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

	slices.Reverse(messages)

	return messages, hasMoreBefore, true, nil
}

func (c *MessageCache) GetAfterByChannelID(
	ctx context.Context,
	channelID, cursorID uuid.UUID,
	limit int,
) ([]*channel.Message, bool, bool, error) {
	if channelID == uuid.Nil || cursorID == uuid.Nil {
		return nil, false, false, nil
	}

	score, found, err := c.fetchCursorScore(ctx, channelID, cursorID)
	if err != nil || !found {
		return nil, false, false, err
	}

	ids, err := c.client.ZRangeByScore(ctx, channelMessagesKey(channelID), &redisdriver.ZRangeBy{
		Min:   formatScoreBound(score, true),
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

func (c *MessageCache) GetAroundByChannelID(
	ctx context.Context,
	channelID, cursorID uuid.UUID,
	beforeLimit, afterLimit int,
) ([]*channel.Message, bool, bool, bool, error) {
	if channelID == uuid.Nil || cursorID == uuid.Nil {
		return nil, false, false, false, nil
	}

	score, found, err := c.fetchCursorScore(ctx, channelID, cursorID)
	if err != nil || !found {
		return nil, false, false, false, err
	}

	beforeIDs, err := c.client.ZRevRangeByScore(ctx, channelMessagesKey(channelID), &redisdriver.ZRangeBy{
		Min:   "-inf",
		Max:   formatScoreBound(score, false),
		Count: int64(beforeLimit + 1),
	}).Result()
	if err != nil {
		return nil, false, false, false, redis.NewError(err, redis.ScopeMessage)
	}

	afterIDs, err := c.client.ZRangeByScore(ctx, channelMessagesKey(channelID), &redisdriver.ZRangeBy{
		Min:   formatScoreBound(score, true),
		Max:   "+inf",
		Count: int64(afterLimit + 1),
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

	slices.Reverse(beforeIDs)
	allIDs := append(beforeIDs, afterIDs...)

	if len(allIDs) == 0 {
		return nil, false, false, false, nil
	}

	messages, err := c.fetchAndUnmarshalBatch(ctx, allIDs)
	if err != nil || messages == nil {
		return nil, false, false, false, err
	}

	return messages, hasMoreBefore, hasMoreAfter, true, nil
}

func (c *MessageCache) fetchCursorScore(ctx context.Context, channelID, cursorID uuid.UUID) (float64, bool, error) {
	score, err := c.client.ZScore(ctx, channelMessagesKey(channelID), cursorID.String()).Result()
	if err != nil {
		if errors.Is(err, redisdriver.Nil) {
			return 0, false, nil
		}
		return 0, false, redis.NewError(err, redis.ScopeMessage)
	}
	return score, true, nil
}

func (c *MessageCache) fetchAndUnmarshalBatch(ctx context.Context, idStrs []string) ([]*channel.Message, error) {
	if len(idStrs) == 0 {
		return []*channel.Message{}, nil
	}

	pipe := c.client.Pipeline()
	cmds := make([]*redisdriver.StringCmd, len(idStrs))

	for i, idStr := range idStrs {
		parsedID, err := uuid.Parse(idStr)
		if err != nil {
			return nil, nil
		}
		cmds[i] = pipe.Get(ctx, messageKey(parsedID))
	}

	_, err := pipe.Exec(ctx)
	if err != nil && !errors.Is(err, redisdriver.Nil) {
		return nil, redis.NewError(err, redis.ScopeMessage)
	}

	messages := make([]*channel.Message, 0, len(idStrs))
	for _, cmd := range cmds {
		rawBytes, err := cmd.Bytes()
		if err != nil {
			return nil, nil
		}

		msg, err := unmarshalMessage(rawBytes)
		if err != nil || msg == nil {
			return nil, nil
		}

		messages = append(messages, msg)
	}

	return messages, nil
}

func messageScore(t time.Time) float64 {
	return float64(t.UnixNano())
}

func formatScoreBound(score float64, exclusive bool) string {
	formatted := strconv.FormatFloat(score, 'f', -1, 64)
	if exclusive {
		return "(" + formatted
	}
	return formatted
}
