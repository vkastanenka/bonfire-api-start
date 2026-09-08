package cache

import (
	"bonfire-api/internal/channel"
	"bonfire-api/internal/fields"
	"bonfire-api/internal/redis"
	"context"
	"encoding/json"
	"errors"

	redisdriver "github.com/redis/go-redis/v9"
)

const (
	channelDomainKey = "channel:"
)

// String / Hash
func channelKey(id fields.ID) string {
	return "{channel:" + id.String() + "}"
}

// Hash
func channelMembersKey(id fields.ID) string {
	return "{channel:" + id.String() + "}:members"
}

// ZSet: Score = UUIDv7 Timestamp (or 0 for Lex), Member = msg_id
func channelMessagesKey(channelID fields.ID) string {
	return "{channel:" + channelID.String() + "}:messages"
}

// String / Hash: Serialized Message Object
func messageKey(msgID fields.ID) string {
	return "{message:" + msgID.String() + "}"
}

// ZSet
func userChannelsKey(userID fields.ID) string {
	return "{user:" + userID.String() + "}:channels"
}

type ChannelCache struct {
	client redisdriver.Cmdable
}

func NewChannelCache(client redisdriver.Cmdable) *ChannelCache {
	return &ChannelCache{
		client: client,
	}
}

func (c *ChannelCache) Get(ctx context.Context, id fields.ID) (*channel.Channel, error) {
	return getAndUnmarshal(ctx, c.client, userKey(id), redis.ScopeChannel, unmarshalChannel)
}

func (c *ChannelCache) Set(ctx context.Context, ch *channel.Channel) error {
	return marshalAndSet(ctx, c.client, userKey(ch.ID()), ch, userTTL, redis.ScopeChannel, marshalChannel)
}

func (c *ChannelCache) Delete(ctx context.Context, id fields.ID) error {
	if err := c.client.Del(ctx, channelKey(id)).Err(); err != nil {
		return redis.NewError(err, redis.ScopeChannel)
	}
	return nil
}

// InvalidateMembers evicts the entire members Hash for a channel (used on topology/membership changes).
func (c *ChannelCache) InvalidateMembers(ctx context.Context, channelID fields.ID) error {
	if err := c.client.Del(ctx, channelMembersKey(channelID)).Err(); err != nil {
		return redis.NewError(err, redis.ScopeChannel)
	}
	return nil
}

// InvalidateMember removes a single user field from the channel's members Hash.
func (c *ChannelCache) InvalidateMember(ctx context.Context, channelID, userID fields.ID) error {
	if err := c.client.HDel(ctx, channelMembersKey(channelID), userID.String()).Err(); err != nil {
		return redis.NewError(err, redis.ScopeChannel)
	}
	return nil
}

func (c *ChannelCache) AddMembers(ctx context.Context, channelID fields.ID, members []*channel.Member) error {
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
		memberMap[m.UserID().String()] = mBytes
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
		score := float64(m.CreatedAt().Time().Unix())
		pipe.ZAdd(ctx, userChannelsKey(m.UserID()), redisdriver.Z{
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
		memberMap[m.UserID().String()] = mBytes
	}

	pipe := c.client.Pipeline()

	// 1. Set channel metadata
	pipe.Set(ctx, channelKey(ch.ID()), chBytes, 0)

	// 2. Set channel members hash (field = userID, value = member JSON)
	if len(memberMap) > 0 {
		pipe.HSet(ctx, channelMembersKey(ch.ID()), memberMap)
	}

	// 3. Add channel ID to each member's userChannels ZSet scored by creation time
	// TODO: Improve to follow actual sidebar sorting score
	score := float64(ch.CreatedAt().Time().Unix())
	for _, m := range members {
		pipe.ZAdd(ctx, userChannelsKey(m.UserID()), redisdriver.Z{
			Score:  score,
			Member: ch.ID().String(),
		})
	}

	_, err = pipe.Exec(ctx)
	return err
}

// GetBatchMembersByChannelIDs fetches members for multiple channels from Redis Hash keys.
// Returns map of found members per channel ID, missing channel IDs, and any execution error.
func (c *ChannelCache) GetBatchMembersByChannelIDs(
	ctx context.Context,
	channelIDs []fields.ID,
) (map[fields.ID][]*channel.Member, []fields.ID, error) {
	if len(channelIDs) == 0 {
		return make(map[fields.ID][]*channel.Member), nil, nil
	}

	pipe := c.client.Pipeline()
	cmds := make(map[fields.ID]*redisdriver.MapStringStringCmd, len(channelIDs))

	for _, id := range channelIDs {
		cmds[id] = pipe.HGetAll(ctx, channelMembersKey(id))
	}

	if _, err := pipe.Exec(ctx); err != nil && !errors.Is(err, redisdriver.Nil) {
		return nil, nil, redis.NewError(err, redis.ScopeChannel)
	}

	found := make(map[fields.ID][]*channel.Member, len(channelIDs))
	var missing []fields.ID

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
			member, err := mDTO.ToDomain()
			if err != nil {
				// TODO
			}
			members = append(members, member)
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
	channelMembersMap map[fields.ID][]*channel.Member,
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
			memberMap[m.UserID().String()] = mBytes
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
	channelID, userID fields.ID,
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

	member, err := mDTO.ToDomain()
	if err != nil {
		_ = c.InvalidateMember(ctx, channelID, userID)
		return nil, nil
	}

	return member, nil
}

// GetVisibleMembersByUserID retrieves member representations for the given user across their cached channel ZSet.
func (c *UserCache) GetVisibleMembersByUserID(
	ctx context.Context,
	userID fields.ID,
	limit int,
) ([]*channel.Member, bool, error) {
	// 1. Get ordered channel IDs for the user using UserCache's native method
	channelIDs, err := c.GetChannelIDs(ctx, userID)
	if err != nil || len(channelIDs) == 0 {
		return nil, false, err
	}

	if limit > 0 && len(channelIDs) > limit {
		channelIDs = channelIDs[:limit]
	}

	// 2. Fetch the user's member object across all these channel Hash keys in a single pipeline
	pipe := c.client.Pipeline()
	userStr := userID.String()
	cmds := make([]*redisdriver.StringCmd, len(channelIDs))

	for i, chID := range channelIDs {
		cmds[i] = pipe.HGet(ctx, channelMembersKey(chID), userStr)
	}

	if _, err := pipe.Exec(ctx); err != nil && !errors.Is(err, redisdriver.Nil) {
		return nil, false, redis.NewError(err, redis.ScopeChannel)
	}

	members := make([]*channel.Member, 0, len(channelIDs))
	for _, cmd := range cmds {
		rawJSON, err := cmd.Result()
		if err != nil {
			// Missing field or hash -> partial cache miss
			return nil, false, nil
		}

		var mDTO Member
		if err := json.Unmarshal([]byte(rawJSON), &mDTO); err != nil {
			return nil, false, nil
		}

		m, err := mDTO.ToDomain()
		if err != nil {
			return nil, false, nil
		}

		members = append(members, m)
	}

	return members, true, nil
}
