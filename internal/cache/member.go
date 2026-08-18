package cache

import (
	"context"
	"encoding/json"
	"time"

	"bonfire-api/internal/channel"
	"bonfire-api/internal/errs"
	"bonfire-api/internal/fields"
	"bonfire-api/internal/redis"

	"github.com/google/uuid"
	redisdriver "github.com/redis/go-redis/v9"
)

type MemberCache struct {
	*KeyCache[MemberKeyIDs, Member]
	client redisdriver.Cmdable
	ttl    time.Duration
}

func NewMemberCache(client redisdriver.Cmdable, ttl time.Duration) *MemberCache {
	return &MemberCache{
		KeyCache: NewKeyCache[MemberKeyIDs, Member](client, redis.ScopeMember, MemberKey),
		client:   client,
		ttl:      ttl,
	}
}

func (c *MemberCache) Get(ctx context.Context, channelID, userID fields.ID) (*channel.Member, error) {
	key := MemberKeyIDs{ChannelID: channelID, UserID: userID}
	dto, err := c.KeyCache.Get(ctx, key)
	if err != nil || dto == nil {
		return nil, err
	}

	return dto.ToDomain()
}

func (c *MemberCache) GetBatch(
	ctx context.Context,
	keys []MemberKeyIDs,
) (map[MemberKeyIDs]*channel.Member, []MemberKeyIDs, error) {
	dtos, missing, err := c.KeyCache.GetBatch(ctx, keys)
	if err != nil {
		return nil, nil, err
	}

	found := make(map[MemberKeyIDs]*channel.Member, len(dtos))
	for k, dto := range dtos {
		if dto == nil {
			missing = append(missing, k)
			continue
		}

		mem, err := dto.ToDomain()
		if err != nil {
			missing = append(missing, k)
			continue
		}
		found[k] = mem
	}

	return found, missing, nil
}

func (c *MemberCache) GetBatchByChannelIDs(
	ctx context.Context,
	channelIDs []fields.ID,
) (found map[fields.ID][]*channel.Member, missingChannelIDs []fields.ID, err error) {
	pipe := c.client.Pipeline()
	cmds := make(map[fields.ID]*redisdriver.StringSliceCmd, len(channelIDs))
	for _, cid := range channelIDs {
		cmds[cid] = pipe.SMembers(ctx, ChannelMembersKey(cid))
	}

	if _, err := pipe.Exec(ctx); err != nil && !redis.IsCacheMiss(err) {
		return nil, nil, redis.NewError(err, redis.ScopeMember)
	}

	channelUserKeys := make(map[fields.ID][]MemberKeyIDs)
	var allMemberKeys []MemberKeyIDs

	for cid, cmd := range cmds {
		userIDsStr, err := cmd.Result()
		if err != nil || len(userIDsStr) == 0 {
			missingChannelIDs = append(missingChannelIDs, cid)
			continue
		}

		var keys []MemberKeyIDs
		corrupted := false
		for _, rawUID := range userIDsStr {
			parsedUUID, err := uuid.Parse(rawUID)
			if err != nil {
				corrupted = true
				break
			}
			uid, err := fields.ParseRequiredID("user_id", parsedUUID)
			if err != nil {
				corrupted = true
				break
			}

			mKey := MemberKeyIDs{ChannelID: cid, UserID: uid}
			keys = append(keys, mKey)
			allMemberKeys = append(allMemberKeys, mKey)
		}

		if corrupted {
			missingChannelIDs = append(missingChannelIDs, cid)
			continue
		}

		channelUserKeys[cid] = keys
	}

	if len(allMemberKeys) == 0 {
		return make(map[fields.ID][]*channel.Member), missingChannelIDs, nil
	}

	dtosMap, missingMemberKeys, err := c.GetBatch(ctx, allMemberKeys)
	if err != nil {
		return nil, nil, err
	}

	missingMemberSet := make(map[MemberKeyIDs]struct{}, len(missingMemberKeys))
	for _, mk := range missingMemberKeys {
		missingMemberSet[mk] = struct{}{}
	}

	found = make(map[fields.ID][]*channel.Member, len(channelIDs))

	for cid, keys := range channelUserKeys {
		channelHasMissingMember := false
		members := make([]*channel.Member, 0, len(keys))

		for _, mk := range keys {
			if _, isMissing := missingMemberSet[mk]; isMissing {
				channelHasMissingMember = true
				break
			}

			mem, ok := dtosMap[mk]
			if !ok || mem == nil {
				channelHasMissingMember = true
				break
			}

			members = append(members, mem)
		}

		if channelHasMissingMember {
			missingChannelIDs = append(missingChannelIDs, cid)
		} else {
			found[cid] = members
		}
	}

	return found, missingChannelIDs, nil
}

// ListVisibleByUserID

func (c *MemberCache) Set(ctx context.Context, member *channel.Member) error {
	key := MemberKeyIDs{ChannelID: member.ChannelID(), UserID: member.UserID()}
	dto := ParseMember(member)

	bytes, err := json.Marshal(dto)
	if err != nil {
		return errs.Internal("Failed to marshal cached json.").
			Meta("scope", redis.ScopeMember.String()).
			Wrap(err)
	}

	pipe := c.client.Pipeline()

	pipe.Set(ctx, MemberKey(key), bytes, c.ttl)

	setKey := ChannelMembersKey(member.ChannelID())
	pipe.SAdd(ctx, setKey, member.UserID().String())
	pipe.Expire(ctx, setKey, c.ttl)

	if _, err := pipe.Exec(ctx); err != nil {
		return redis.NewError(err, redis.ScopeMember)
	}

	return nil
}

func (c *MemberCache) SetBatch(ctx context.Context, members []*channel.Member) error {
	if len(members) == 0 {
		return nil
	}

	dtos := make(map[MemberKeyIDs]Member, len(members))
	channelUserMap := make(map[fields.ID][]any)

	for _, mem := range members {
		if mem == nil {
			continue
		}
		cid := mem.ChannelID()
		uid := mem.UserID()

		dtos[MemberKeyIDs{ChannelID: cid, UserID: uid}] = ParseMember(mem)
		channelUserMap[cid] = append(channelUserMap[cid], uid.String())
	}

	if err := c.KeyCache.SetBatch(ctx, dtos, c.ttl); err != nil {
		return err
	}

	pipe := c.client.Pipeline()
	for cid, userIDs := range channelUserMap {
		setKey := ChannelMembersKey(cid)
		pipe.SAdd(ctx, setKey, userIDs...)
		pipe.Expire(ctx, setKey, c.ttl)
	}

	if _, err := pipe.Exec(ctx); err != nil {
		return redis.NewError(err, redis.ScopeMember)
	}

	return nil
}

func (c *MemberCache) Delete(ctx context.Context, channelID, userID fields.ID) error {
	pipe := c.client.Pipeline()

	setKey := ChannelMembersKey(channelID)
	memKey := MemberKey(MemberKeyIDs{ChannelID: channelID, UserID: userID})

	pipe.SRem(ctx, setKey, userID.String())
	pipe.Del(ctx, memKey)

	if _, err := pipe.Exec(ctx); err != nil {
		return redis.NewError(err, redis.ScopeMember)
	}

	return nil
}
