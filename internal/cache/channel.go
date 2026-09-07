package cache

import (
	"bonfire-api/internal/channel"
	"bonfire-api/internal/fields"
	"context"
	"encoding/json"

	redisdriver "github.com/redis/go-redis/v9"
)

const (
	channelDomainKey = "channel:"
)

// String / Hash
func channelKey(channelID fields.ID) string {
	return "{channel:" + channelID.String() + "}"
}

// Hash
func channelMembersKey(channelID fields.ID) string {
	return "{channel:" + channelID.String() + "}:members"
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
