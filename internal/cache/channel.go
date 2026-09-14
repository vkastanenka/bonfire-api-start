package cache

import (
	"bonfire-api/internal/channel"
	"bonfire-api/internal/redis"
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	redisdriver "github.com/redis/go-redis/v9"
)

const (
	channelDomainKey = "channel:"
	channelTTL       = 24 * time.Hour
)

// String / Hash
func channelKey(id uuid.UUID) string {
	return "{channel:" + id.String() + "}"
}

// Hash
func channelMembersKey(id uuid.UUID) string {
	return "{channel:" + id.String() + "}:members"
}

// ZSet: Score = UUIDv7 Timestamp (or 0 for Lex), Member = msg_id
func channelMessagesKey(channelID uuid.UUID) string {
	return "{channel:" + channelID.String() + "}:messages"
}

// String / Hash: Serialized Message Object
func messageKey(msgID uuid.UUID) string {
	return "{message:" + msgID.String() + "}"
}

type ChannelCache struct {
	client redisdriver.Cmdable
}

func NewChannelCache(client redisdriver.Cmdable) *ChannelCache {
	return &ChannelCache{
		client: client,
	}
}

func (c *ChannelCache) Get(ctx context.Context, id uuid.UUID) (*channel.Channel, error) {
	return getAndUnmarshal(ctx, c.client, channelKey(id), redis.ScopeChannel, unmarshalChannel)
}

func (c *ChannelCache) Set(ctx context.Context, ch *channel.Channel) error {
	return marshalAndSet(ctx, c.client, channelKey(ch.ID), ch, channelTTL, redis.ScopeChannel, marshalChannel)
}

func (c *ChannelCache) Delete(ctx context.Context, id uuid.UUID) error {
	if err := c.client.Del(ctx, channelKey(id)).Err(); err != nil {
		return redis.NewError(err, redis.ScopeChannel)
	}
	return nil
}

// GetBatch retrieves multiple channels by their IDs in chunked MGET calls.
func (c *ChannelCache) GetBatch(
	ctx context.Context,
	ids []uuid.UUID,
) (map[uuid.UUID]*channel.Channel, []uuid.UUID, error) {
	if len(ids) == 0 {
		return make(map[uuid.UUID]*channel.Channel), nil, nil
	}

	found := make(map[uuid.UUID]*channel.Channel, len(ids))
	missing := make([]uuid.UUID, 0, len(ids))
	var corruptedKeys []string

	for i := 0; i < len(ids); i += maxBatchSize {
		if err := ctx.Err(); err != nil {
			return nil, nil, err
		}

		end := min(i+maxBatchSize, len(ids))
		chunk := ids[i:end]

		redisKeys := make([]string, len(chunk))
		for j, id := range chunk {
			redisKeys[j] = channelKey(id)
		}

		vals, err := getBatchKeys(ctx, c.client, redisKeys, redis.ScopeChannel)
		if err != nil {
			return nil, nil, err
		}

		for j, raw := range vals {
			id := chunk[j]
			rKey := redisKeys[j]

			data, ok := toBytes(raw)
			if !ok {
				missing = append(missing, id)
				continue
			}

			ch, err := unmarshalChannel(data)
			if err != nil {
				corruptedKeys = append(corruptedKeys, rKey)
				missing = append(missing, id)
				continue
			}

			found[id] = ch
		}
	}

	if len(corruptedKeys) > 0 {
		deleteBatchKeys(ctx, c.client, corruptedKeys, redis.ScopeChannel)
	}

	return found, missing, nil
}

// SetBatch stores multiple channels into Redis using chunked pipeline requests.
func (c *ChannelCache) SetBatch(ctx context.Context, channels map[uuid.UUID]*channel.Channel) error {
	if len(channels) == 0 {
		return nil
	}

	items := make([]CacheItem, 0, len(channels))
	for id, ch := range channels {
		if ch == nil || id == uuid.Nil {
			continue
		}

		bytes, err := marshalChannel(ch)
		if err != nil {
			return err
		}

		items = append(items, CacheItem{
			Key:   channelKey(id),
			Value: bytes,
		})
	}

	return setBatchPipeline(ctx, c.client, items, channelTTL, redis.ScopeChannel)
}

// InvalidateMembers evicts the entire members Hash for a channel (used on topology/membership changes).
func (c *ChannelCache) InvalidateMembers(ctx context.Context, channelID uuid.UUID) error {
	if err := c.client.Del(ctx, channelMembersKey(channelID)).Err(); err != nil {
		return redis.NewError(err, redis.ScopeChannel)
	}
	return nil
}

// InvalidateMember removes a single user field from the channel's members Hash.
func (c *ChannelCache) InvalidateMember(ctx context.Context, channelID, userID uuid.UUID) error {
	if err := c.client.HDel(ctx, channelMembersKey(channelID), userID.String()).Err(); err != nil {
		return redis.NewError(err, redis.ScopeChannel)
	}
	return nil
}

func (c *ChannelCache) AddMembers(ctx context.Context, channelID uuid.UUID, members []*channel.Member) error {
	if len(members) == 0 {
		return nil
	}

	memberMap := make(map[string]any, len(members))
	for _, m := range members {
		mDTO := ParseMember(m)
		mBytes, err := json.Marshal(mDTO)
		if err != nil {
			return err
		}
		memberMap[m.UserID.String()] = mBytes
	}

	hashKey := channelMembersKey(channelID)

	// Check if the channel members hash exists in cache to prevent partial writes on an evicted hash
	exists, err := c.client.Exists(ctx, hashKey).Result()
	if err != nil {
		return redis.NewError(err, redis.ScopeChannel)
	}

	pipe := c.client.Pipeline()

	// 1. If channel members hash is cached, add the new members
	if exists > 0 {
		pipe.HSet(ctx, hashKey, memberMap)
	}

	// 2. Add channel ID to each new member's userChannels ZSet scored by member creation time
	for _, m := range members {
		score := float64(m.CreatedAt.Unix())
		pipe.ZAdd(ctx, userChannelsKey(m.UserID), redisdriver.Z{
			Score:  score,
			Member: channelID.String(),
		})
	}

	_, err = pipe.Exec(ctx)
	if err != nil {
		return redis.NewError(err, redis.ScopeChannel)
	}

	return nil
}

func (c *ChannelCache) CreateGroup(ctx context.Context, ch *channel.Channel, members []*channel.Member) error {
	chDTO := ParseChannel(ch)
	chBytes, err := json.Marshal(chDTO)
	if err != nil {
		return err
	}

	memberMap := make(map[string]any, len(members))
	for _, m := range members {
		mDTO := ParseMember(m)
		mBytes, err := json.Marshal(mDTO)
		if err != nil {
			return err
		}
		memberMap[m.UserID.String()] = mBytes
	}

	pipe := c.client.Pipeline()

	// 1. Set channel metadata
	pipe.Set(ctx, channelKey(ch.ID), chBytes, 0)

	// 2. Set channel members hash (field = userID, value = member JSON)
	if len(memberMap) > 0 {
		pipe.HSet(ctx, channelMembersKey(ch.ID), memberMap)
	}

	// 3. Add channel ID to each member's userChannels ZSet scored by creation time
	// TODO: Improve to follow actual sidebar sorting score
	score := float64(ch.CreatedAt.Unix())
	for _, m := range members {
		pipe.ZAdd(ctx, userChannelsKey(m.UserID), redisdriver.Z{
			Score:  score,
			Member: ch.ID.String(),
		})
	}

	_, err = pipe.Exec(ctx)
	return err
}

// GetBatchMembersByChannelIDs fetches members for multiple channels from Redis Hash keys.
// Returns map of found members per channel ID, missing channel IDs, and any execution error.
func (c *ChannelCache) GetBatchMembersByChannelIDs(
	ctx context.Context,
	channelIDs []uuid.UUID,
) (map[uuid.UUID][]*channel.Member, []uuid.UUID, error) {
	if len(channelIDs) == 0 {
		return make(map[uuid.UUID][]*channel.Member), nil, nil
	}

	pipe := c.client.Pipeline()
	cmds := make(map[uuid.UUID]*redisdriver.MapStringStringCmd, len(channelIDs))

	for _, id := range channelIDs {
		cmds[id] = pipe.HGetAll(ctx, channelMembersKey(id))
	}

	if _, err := pipe.Exec(ctx); err != nil && !errors.Is(err, redisdriver.Nil) {
		return nil, nil, redis.NewError(err, redis.ScopeChannel)
	}

	found := make(map[uuid.UUID][]*channel.Member, len(channelIDs))
	var missing []uuid.UUID

	for id, cmd := range cmds {
		rawMap, err := cmd.Result()
		if err != nil || len(rawMap) == 0 {
			missing = append(missing, id)
			continue
		}

		members := make([]*channel.Member, 0, len(rawMap))
		for _, rawJSON := range rawMap {
			var mDTO Member
			if err := json.Unmarshal([]byte(rawJSON), &mDTO); err != nil {
				// If serialization fails, mark channel as missing to trigger backfill
				missing = append(missing, id)
				delete(found, id)
				break
			}
			members = append(members, mDTO.ToDomain())
		}

		if _, isMissing := found[id]; !isMissing {
			found[id] = members
		}
	}

	return found, missing, nil
}

// SetBatchMembers writes a map of channel members into Redis Hashes via a single pipeline.
func (c *ChannelCache) SetBatchMembers(
	ctx context.Context,
	channelMembersMap map[uuid.UUID][]*channel.Member,
) error {
	if len(channelMembersMap) == 0 {
		return nil
	}

	pipe := c.client.Pipeline()

	for channelID, members := range channelMembersMap {
		if len(members) == 0 {
			continue
		}

		memberMap := make(map[string]any, len(members))
		for _, m := range members {
			mDTO := ParseMember(m)
			mBytes, err := json.Marshal(mDTO)
			if err != nil {
				return err
			}
			memberMap[m.UserID.String()] = mBytes
		}

		pipe.HSet(ctx, channelMembersKey(channelID), memberMap)
	}

	_, err := pipe.Exec(ctx)
	if err != nil {
		return redis.NewError(err, redis.ScopeChannel)
	}

	return nil
}

func (c *ChannelCache) GetMember(
	ctx context.Context,
	channelID, userID uuid.UUID,
) (*channel.Member, error) {
	rawJSON, err := c.client.HGet(ctx, channelMembersKey(channelID), userID.String()).Result()
	if err != nil {
		if errors.Is(err, redisdriver.Nil) {
			return nil, nil
		}
		return nil, redis.NewError(err, redis.ScopeChannel)
	}

	var mDTO Member
	if err := json.Unmarshal([]byte(rawJSON), &mDTO); err != nil {
		// On serialization error, invalidate this field so DB backfills valid data
		_ = c.InvalidateMember(ctx, channelID, userID)
		return nil, nil
	}

	return mDTO.ToDomain(), nil
}
