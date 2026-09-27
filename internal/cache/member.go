package cache

import (
	"context"

	"bonfire-api/internal/channel"
	"bonfire-api/internal/redis"

	"github.com/google/uuid"
	redisdriver "github.com/redis/go-redis/v9"
)

func channelMembersKey(id uuid.UUID) string { return "{channel:" + id.String() + "}:members" }

type MemberCache struct {
	client redisdriver.Cmdable
}

func NewMemberCache(client redisdriver.Cmdable) *MemberCache {
	return &MemberCache{client: client}
}

func (c *MemberCache) Get(
	ctx context.Context,
	channelID, userID uuid.UUID,
) (*channel.Member, error) {
	if channelID == uuid.Nil || userID == uuid.Nil {
		return nil, nil
	}

	rawBytes, err := c.client.HGet(ctx, channelMembersKey(channelID), userID.String()).Bytes()
	if err != nil {
		if redis.IsCacheMiss(err) {
			return nil, nil
		}
		return nil, redis.NewError(err, redis.ScopeMember)
	}

	member, err := unmarshalMember(rawBytes)
	if err != nil {
		_ = c.Invalidate(ctx, channelID, userID)
		return nil, nil
	}

	return member, nil
}

func (c *MemberCache) Invalidate(ctx context.Context, channelID, userID uuid.UUID) error {
	if err := c.client.HDel(ctx, channelMembersKey(channelID), userID.String()).Err(); err != nil {
		return redis.NewError(err, redis.ScopeMember)
	}
	return nil
}

func (c *MemberCache) InvalidateChannel(ctx context.Context, channelID uuid.UUID) error {
	if err := c.client.Del(ctx, channelMembersKey(channelID)).Err(); err != nil {
		return redis.NewError(err, redis.ScopeMember)
	}
	return nil
}

func (c *MemberCache) SetUserMembers(ctx context.Context, userID uuid.UUID, members []*channel.Member) error {
	if userID == uuid.Nil || len(members) == 0 {
		return nil
	}

	pipe := c.client.Pipeline()

	zKey := userChannelsKey(userID)
	pipe.Del(ctx, zKey)

	for _, m := range members {
		if m == nil || m.ChannelID == uuid.Nil {
			continue
		}

		var score float64
		if m.PinnedAt != nil && !m.PinnedAt.IsZero() {
			score = 1e12 + float64(m.PinnedAt.Unix())
		} else {
			score = float64(m.CreatedAt.Unix())
		}

		pipe.ZAdd(ctx, zKey, redisdriver.Z{
			Score:  score,
			Member: m.ChannelID.String(),
		})
	}

	for _, m := range members {
		if m == nil || m.ChannelID == uuid.Nil || m.UserID == uuid.Nil {
			continue
		}

		mBytes, err := marshalMember(m)
		if err != nil {
			return err
		}

		pipe.HSet(ctx, channelMembersKey(m.ChannelID), m.UserID.String(), mBytes)
	}

	if _, err := pipe.Exec(ctx); err != nil {
		return redis.NewError(err, redis.ScopeMember)
	}

	return nil
}

func (c *MemberCache) Add(ctx context.Context, channelID uuid.UUID, members []*channel.Member) error {
	if len(members) == 0 {
		return nil
	}

	memberMap := make(map[string]any, len(members))
	for _, m := range members {
		if m == nil || m.UserID == uuid.Nil {
			continue
		}

		mBytes, err := marshalMember(m)
		if err != nil {
			return err
		}
		memberMap[m.UserID.String()] = mBytes
	}

	if len(memberMap) == 0 {
		return nil
	}

	pipe := c.client.Pipeline()
	pipe.HSet(ctx, channelMembersKey(channelID), memberMap)

	channelIDStr := channelID.String()
	for _, m := range members {
		if m == nil || m.UserID == uuid.Nil {
			continue
		}

		var score float64
		if m.PinnedAt != nil && !m.PinnedAt.IsZero() {
			score = 1e12 + float64(m.PinnedAt.Unix())
		} else {
			score = float64(m.CreatedAt.Unix())
		}

		pipe.ZAdd(ctx, userChannelsKey(m.UserID), redisdriver.Z{
			Score:  score,
			Member: channelIDStr,
		})
	}

	if _, err := pipe.Exec(ctx); err != nil {
		return redis.NewError(err, redis.ScopeMember)
	}

	return nil
}

func (c *MemberCache) Remove(ctx context.Context, channelID, userID uuid.UUID) error {
	pipe := c.client.Pipeline()

	pipe.HDel(ctx, channelMembersKey(channelID), userID.String())
	pipe.ZRem(ctx, userChannelsKey(userID), channelID.String())

	if _, err := pipe.Exec(ctx); err != nil {
		return redis.NewError(err, redis.ScopeMember)
	}
	return nil
}

func (c *MemberCache) GetBatchByChannelIDs(
	ctx context.Context,
	channelIDs []uuid.UUID,
) (map[uuid.UUID][]*channel.Member, []uuid.UUID, error) {
	if len(channelIDs) == 0 {
		return make(map[uuid.UUID][]*channel.Member), nil, nil
	}

	keys := make([]string, 0, len(channelIDs))
	validIDs := make([]uuid.UUID, 0, len(channelIDs))

	for _, id := range channelIDs {
		if id == uuid.Nil {
			continue
		}
		keys = append(keys, channelMembersKey(id))
		validIDs = append(validIDs, id)
	}

	rawMaps, err := getBatchHashMaps(ctx, redis.ScopeMember, c.client, keys)
	if err != nil {
		return nil, nil, err
	}

	found := make(map[uuid.UUID][]*channel.Member, len(validIDs))
	var missing []uuid.UUID

	for i, rawMap := range rawMaps {
		id := validIDs[i]
		if len(rawMap) == 0 {
			missing = append(missing, id)
			continue
		}

		members := make([]*channel.Member, 0, len(rawMap))
		corrupted := false

		for _, rawJSON := range rawMap {
			m, err := unmarshalMember([]byte(rawJSON))
			if err != nil || m == nil {
				_ = c.InvalidateChannel(ctx, id)
				missing = append(missing, id)
				corrupted = true
				break
			}
			members = append(members, m)
		}

		if !corrupted {
			found[id] = members
		}
	}

	return found, missing, nil
}

func (c *MemberCache) SetBatchByChannelIDs(
	ctx context.Context,
	channelMembersMap map[uuid.UUID][]*channel.Member,
) error {
	if len(channelMembersMap) == 0 {
		return nil
	}

	pipe := c.client.Pipeline()
	hasOps := false

	for channelID, members := range channelMembersMap {
		if channelID == uuid.Nil || len(members) == 0 {
			continue
		}

		memberMap := make(map[string]any, len(members))
		for _, m := range members {
			if m == nil || m.UserID == uuid.Nil {
				continue
			}

			mBytes, err := marshalMember(m)
			if err != nil {
				return err
			}
			memberMap[m.UserID.String()] = mBytes
		}

		if len(memberMap) > 0 {
			pipe.HSet(ctx, channelMembersKey(channelID), memberMap)
			hasOps = true
		}
	}

	if !hasOps {
		return nil
	}

	if _, err := pipe.Exec(ctx); err != nil {
		return redis.NewError(err, redis.ScopeMember)
	}

	return nil
}
