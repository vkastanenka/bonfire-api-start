package cache

import (
	"bonfire-api/internal/channel"
	"bonfire-api/internal/pkg/errs"
	"bonfire-api/internal/redis"
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	redisdriver "github.com/redis/go-redis/v9"
)

const (
	channelTTL = 24 * time.Hour
)

// String
func channelKey(channelID uuid.UUID) string {
	return "{channel:" + channelID.String() + "}"
}

// Hash
func channelMembersKey(id uuid.UUID) string {
	return "{channel:" + id.String() + "}:members"
}

// ZSet
func channelMessagesKey(channelID uuid.UUID) string {
	return "{channel:" + channelID.String() + "}:messages"
}

// String
func messageKey(messageID uuid.UUID) string {
	return "{message:" + messageID.String() + "}"
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
	return getAndUnmarshal(ctx, redis.ScopeChannel, c.client, channelKey(id), unmarshalChannel)
}

func (c *ChannelCache) Set(ctx context.Context, ch *channel.Channel) error {
	return marshalAndSet(ctx, redis.ScopeChannel, c.client, channelKey(ch.ID), ch, channelTTL, marshalChannel)
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
	redisKeys := make([]string, len(ids))
	for i, id := range ids {
		redisKeys[i] = channelKey(id)
	}

	return getAndUnmarshalBatch(
		ctx,
		redis.ScopeChannel,
		c.client,
		redisKeys,
		ids,
		unmarshalChannel,
	)
}

// SetBatch stores multiple channels into Redis using chunked pipeline requests.
func (c *ChannelCache) SetBatch(ctx context.Context, channels map[uuid.UUID]*channel.Channel) error {
	return marshalAndSetBatch(
		ctx,
		c.client,
		channels,
		channelKey,
		channelTTL,
		redis.ScopeChannel,
		maxBatchSize,
		marshalChannel,
	)
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

	pipe := c.client.Pipeline()

	// 1. Invalidate channel members hash.
	// Evicting the key ensures we never write a partial member list to a key
	// that was evicted from Redis between checking Exists and calling Exec.
	pipe.Del(ctx, channelMembersKey(channelID))

	// 2. Add channel ID to each new member's userChannels ZSet scored by member creation time
	for _, m := range members {
		if m == nil || m.UserID == uuid.Nil {
			continue
		}

		score := float64(m.CreatedAt.Unix())
		pipe.ZAdd(ctx, userChannelsKey(m.UserID), redisdriver.Z{
			Score:  score,
			Member: channelID.String(),
		})
	}

	if _, err := pipe.Exec(ctx); err != nil {
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

func unmarshalChannel(data []byte) (*channel.Channel, error) {
	if len(data) == 0 {
		return nil, nil
	}

	var dto Channel
	if err := json.Unmarshal(data, &dto); err != nil {
		return nil, err
	}
	return dto.ToDomain(), nil
}

func marshalChannel(ch *channel.Channel) ([]byte, error) {
	if ch == nil {
		return nil, nil
	}

	dto := ParseChannel(ch)
	bytes, err := json.Marshal(dto)
	if err != nil {
		return nil, errs.Internal("Failed to marshal channel json.").
			Meta("scope", redis.ScopeChannel.String()).
			Wrap(err)
	}
	return bytes, nil
}

func unmarshalMessage(data []byte) (*channel.Message, error) {
	if len(data) == 0 {
		return nil, nil
	}

	var dto Message
	if err := json.Unmarshal(data, &dto); err != nil {
		return nil, err
	}
	return dto.ToDomain(), nil
}
