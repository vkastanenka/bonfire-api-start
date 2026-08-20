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
		cmds[cid] = pipe.SMembers(ctx, ChannelMemberIDsKey(cid))
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

func (c *MemberCache) GetVisibleByUserID(ctx context.Context, userID fields.ID) ([]*channel.Member, []fields.ID, error) {
	setKey := UserChannelIDsKey(userID)

	rawIDs, err := c.client.SMembers(ctx, setKey).Result()
	if redis.IsCacheMiss(err) || len(rawIDs) == 0 {
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, redis.NewError(err, c.scope)
	}

	channelIDs := make([]fields.ID, 0, len(rawIDs))
	for _, rawID := range rawIDs {
		parsedUUID, err := uuid.Parse(rawID)
		if err != nil {
			continue
		}
		id, err := fields.ParseRequiredID("channel_id", parsedUUID)
		if err != nil {
			continue
		}
		channelIDs = append(channelIDs, id)
	}

	if len(channelIDs) == 0 {
		return nil, nil, nil
	}

	resultMap, missingChannelIDs, err := c.GetBatchByChannelIDs(ctx, channelIDs)
	if err != nil {
		return nil, nil, err
	}

	if len(missingChannelIDs) > 0 {
		return nil, missingChannelIDs, nil
	}

	var members []*channel.Member
	for _, cid := range channelIDs {
		cachedMembers, ok := resultMap[cid]
		if !ok || len(cachedMembers) == 0 {
			return nil, channelIDs, nil
		}
		members = append(members, cachedMembers...)
	}

	if len(resultMap) != len(channelIDs) {
		return nil, channelIDs, nil
	}

	return members, nil, nil
}

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

	channelSetKey := ChannelMemberIDsKey(member.ChannelID())
	pipe.SAdd(ctx, channelSetKey, member.UserID().String())
	pipe.Expire(ctx, channelSetKey, c.ttl)

	userSetKey := UserChannelIDsKey(member.UserID())
	pipe.SAdd(ctx, userSetKey, member.ChannelID().String())
	pipe.Expire(ctx, userSetKey, c.ttl)

	if _, err := pipe.Exec(ctx); err != nil {
		return redis.NewError(err, redis.ScopeMember)
	}

	return nil
}

func (c *MemberCache) SetBatch(ctx context.Context, members []*channel.Member) error {
	dtos := make(map[MemberKeyIDs]Member, len(members))
	channelUserMap := make(map[fields.ID][]any)
	userChannelMap := make(map[fields.ID][]any)

	for _, mem := range members {
		if mem == nil {
			continue
		}
		cid := mem.ChannelID()
		uid := mem.UserID()

		dtos[MemberKeyIDs{ChannelID: cid, UserID: uid}] = ParseMember(mem)
		channelUserMap[cid] = append(channelUserMap[cid], uid.String())
		userChannelMap[uid] = append(userChannelMap[uid], cid.String())
	}

	if err := c.KeyCache.SetBatch(ctx, dtos, c.ttl); err != nil {
		return err
	}

	pipe := c.client.Pipeline()

	for cid, userIDs := range channelUserMap {
		setKey := ChannelMemberIDsKey(cid)
		pipe.SAdd(ctx, setKey, userIDs...)
		pipe.Expire(ctx, setKey, c.ttl)
	}

	for uid, channelIDs := range userChannelMap {
		userSetKey := UserChannelIDsKey(uid)
		pipe.SAdd(ctx, userSetKey, channelIDs...)
		pipe.Expire(ctx, userSetKey, c.ttl)
	}

	if _, err := pipe.Exec(ctx); err != nil {
		return redis.NewError(err, redis.ScopeMember)
	}

	return nil
}

func (c *MemberCache) Delete(ctx context.Context, channelID, userID fields.ID) error {
	pipe := c.client.Pipeline()

	channelSetKey := ChannelMemberIDsKey(channelID)
	userSetKey := UserChannelIDsKey(userID)
	memKey := MemberKey(MemberKeyIDs{ChannelID: channelID, UserID: userID})

	pipe.SRem(ctx, channelSetKey, userID.String())
	pipe.SRem(ctx, userSetKey, channelID.String())
	pipe.Del(ctx, memKey)

	if _, err := pipe.Exec(ctx); err != nil {
		return redis.NewError(err, redis.ScopeMember)
	}

	return nil
}
