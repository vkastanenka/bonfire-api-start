package cache

import (
	"context"
	"errors"
	"time"

	"bonfire-api/internal/channel"
	"bonfire-api/internal/fields"
	"bonfire-api/internal/redis"

	"github.com/google/uuid"
	redisdriver "github.com/redis/go-redis/v9"
)

type MemberKeyIDs struct {
	ChannelID fields.ID
	UserID    fields.ID
}

func MemberKey(k MemberKeyIDs) string {
	return "member:" + k.ChannelID.String() + ":" + k.UserID.String()
}

func ChannelMembersKey(channelID fields.ID) string {
	return "channel:" + channelID.String() + ":members"
}

type Member struct {
	ChannelID         uuid.UUID `json:"channel_id"`
	UserID            uuid.UUID `json:"user_id"`
	LastReadMessageID uuid.UUID `json:"last_read_message_id"`
	LastReadMessageAt time.Time `json:"last_read_message_at"`
	PinnedAt          time.Time `json:"pinned_at"`
	MutedUntil        time.Time `json:"muted_until"`
	MentionCount      int32     `json:"mention_count"`
	IsVisible         bool      `json:"is_visible"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

func (m Member) ToDomain() (*channel.Member, error) {
	channelID, err := fields.ParseRequiredID("channel_id", m.ChannelID)
	if err != nil {
		return nil, err
	}

	userID, err := fields.ParseRequiredID("user_id", m.UserID)
	if err != nil {
		return nil, err
	}

	lastReadMessageID, err := fields.ParseID("last_read_message_id", m.LastReadMessageID)
	if err != nil {
		return nil, err
	}

	return channel.ParseMember(
		channelID,
		userID,
		lastReadMessageID,
		fields.NewTimestamp(m.LastReadMessageAt),
		fields.NewTimestamp(m.PinnedAt),
		fields.NewTimestamp(m.MutedUntil),
		m.MentionCount,
		m.IsVisible,
		fields.NewTimestamp(m.CreatedAt),
		fields.NewTimestamp(m.UpdatedAt),
	), nil
}

func ParseMember(m *channel.Member) Member {
	return Member{
		ChannelID:         m.ChannelID().UUID(),
		UserID:            m.UserID().UUID(),
		LastReadMessageID: m.LastReadMessageID().UUID(),
		LastReadMessageAt: m.LastReadMessageAt().Time(),
		PinnedAt:          m.PinnedAt().Time(),
		MutedUntil:        m.MutedUntil().Time(),
		MentionCount:      m.MentionCount(),
		IsVisible:         m.IsVisible(),
		CreatedAt:         m.CreatedAt().Time(),
		UpdatedAt:         m.UpdatedAt().Time(),
	}
}

type MemberCache struct {
	*ScopeCache[MemberKeyIDs, Member]
	client redisdriver.Cmdable
	ttl    time.Duration
}

func NewMemberCache(client redisdriver.Cmdable, ttl time.Duration) *MemberCache {
	return &MemberCache{
		ScopeCache: NewScopeCache[MemberKeyIDs, Member](client, redis.ScopeMember, MemberKey),
		client:     client,
		ttl:        ttl,
	}
}

// Get fetches a single channel member.
func (c *MemberCache) Get(ctx context.Context, channelID, userID fields.ID) (*channel.Member, error) {
	key := MemberKeyIDs{ChannelID: channelID, UserID: userID}
	dto, err := c.ScopeCache.Get(ctx, key)
	if err != nil || dto == nil {
		return nil, err
	}

	return dto.ToDomain()
}

// GetBatch fetches individual member keys specified in keys slice.
func (c *MemberCache) GetBatch(
	ctx context.Context,
	keys []MemberKeyIDs,
) (map[MemberKeyIDs]*channel.Member, []MemberKeyIDs, error) {
	dtos, missing, err := c.ScopeCache.GetBatch(ctx, keys)
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

// GetByChannelID fetches all members belonging to a channel using the channel:member-id set index.
// If the member ID index set doesn't exist, hitSetMiss will be true (triggering a DB fallback).
func (c *MemberCache) GetByChannelID(
	ctx context.Context,
	channelID fields.ID,
) (members []*channel.Member, hitSetMiss bool, err error) {
	setKey := ChannelMembersKey(channelID)

	opCtx, cancel := context.WithTimeout(ctx, ScopeBatchTimeout)
	defer cancel()

	userIDsStr, err := c.client.SMembers(opCtx, setKey).Result()
	if err != nil {
		return nil, false, redis.NewError(err, redis.ScopeMember)
	}

	// Index set missing in Redis -> caller should fallback to DB to populate cache
	if len(userIDsStr) == 0 {
		return nil, true, nil
	}

	keys := make([]MemberKeyIDs, 0, len(userIDsStr))
	for _, rawUID := range userIDsStr {
		parsedUUID, err := uuid.Parse(rawUID)
		if err != nil {
			return nil, true, nil // Corrupted UUID in set; treat as cache miss to repair set from DB
		}

		uid, err := fields.ParseRequiredID("user_id", parsedUUID)
		if err != nil {
			return nil, true, nil // Corrupted ID; treat as cache miss
		}

		keys = append(keys, MemberKeyIDs{
			ChannelID: channelID,
			UserID:    uid,
		})
	}

	foundMap, missingKeys, err := c.GetBatch(ctx, keys)
	if err != nil {
		return nil, false, err
	}

	// If any individual member DTO keys expired, mark as miss so DB can re-populate whole set consistently
	if len(missingKeys) > 0 {
		return nil, true, nil
	}

	members = make([]*channel.Member, 0, len(foundMap))
	for _, m := range foundMap {
		members = append(members, m)
	}

	return members, false, nil
}

// Set stores an individual member DTO and ensures their ID is added to the channel member index set.
func (c *MemberCache) Set(ctx context.Context, mem *channel.Member) error {
	if mem == nil {
		return nil
	}

	opCtx, cancel := context.WithTimeout(ctx, ScopeBatchTimeout)
	defer cancel()

	pipe := c.client.Pipeline()

	// 1. Save member DTO key
	key := MemberKeyIDs{ChannelID: mem.ChannelID(), UserID: mem.UserID()}
	dto := ParseMember(mem)
	if err := c.ScopeCache.Set(ctx, key, dto, c.ttl); err != nil {
		return err
	}

	// 2. Add to channel set index
	setKey := ChannelMembersKey(mem.ChannelID())
	pipe.SAdd(opCtx, setKey, mem.UserID().String())
	pipe.Expire(opCtx, setKey, c.ttl)

	if _, err := pipe.Exec(opCtx); err != nil {
		return redis.NewError(err, redis.ScopeMember)
	}

	return nil
}

// SetBatch stores all provided members and initializes/refreshes the channel index sets.
func (c *MemberCache) SetBatch(ctx context.Context, members []*channel.Member) error {
	if len(members) == 0 {
		return nil
	}

	// 1. Set member DTOs
	if err := c.ScopeCache.SetBatch(ctx, toMemberDTOMap(members), c.ttl); err != nil {
		return err
	}

	// 2. Group User IDs by Channel ID for pipeline update
	channelUserMap := make(map[fields.ID][]any)
	for _, mem := range members {
		if mem == nil {
			continue
		}
		cid := mem.ChannelID()
		channelUserMap[cid] = append(channelUserMap[cid], mem.UserID().String())
	}

	opCtx, cancel := context.WithTimeout(ctx, ScopeBatchTimeout)
	defer cancel()

	pipe := c.client.Pipeline()
	for cid, userIDs := range channelUserMap {
		setKey := ChannelMembersKey(cid)
		pipe.SAdd(opCtx, setKey, userIDs...)
		pipe.Expire(opCtx, setKey, c.ttl)
	}

	if _, err := pipe.Exec(opCtx); err != nil {
		return redis.NewError(err, redis.ScopeMember)
	}

	return nil
}

// RemoveMember removes a user from a channel set index and deletes their member key.
func (c *MemberCache) RemoveMember(ctx context.Context, channelID, userID fields.ID) error {
	opCtx, cancel := context.WithTimeout(ctx, ScopeBatchTimeout)
	defer cancel()

	pipe := c.client.Pipeline()
	setKey := ChannelMembersKey(channelID)
	memKey := MemberKey(MemberKeyIDs{ChannelID: channelID, UserID: userID})

	pipe.SRem(opCtx, setKey, userID.String())
	pipe.Del(opCtx, memKey)

	if _, err := pipe.Exec(opCtx); err != nil {
		return redis.NewError(err, redis.ScopeMember)
	}

	return nil
}

func toMemberDTOMap(members []*channel.Member) map[MemberKeyIDs]Member {
	dtos := make(map[MemberKeyIDs]Member, len(members))
	for _, mem := range members {
		if mem == nil {
			continue
		}
		key := MemberKeyIDs{ChannelID: mem.ChannelID(), UserID: mem.UserID()}
		dtos[key] = ParseMember(mem)
	}
	return dtos
}

// GetBatchByChannelIDs fetches members for multiple channels concurrently using pipelined Redis operations.
func (c *MemberCache) GetBatchByChannelIDs(
	ctx context.Context,
	channelIDs []fields.ID,
) (found map[fields.ID][]*channel.Member, missingChannelIDs []fields.ID, err error) {
	if len(channelIDs) == 0 {
		return make(map[fields.ID][]*channel.Member), nil, nil
	}

	opCtx, cancel := context.WithTimeout(ctx, ScopeBatchTimeout)
	defer cancel()

	// 1. Pipeline all SMembers calls to fetch user IDs for every channel in 1 round-trip
	pipe := c.client.Pipeline()
	cmds := make(map[fields.ID]*redisdriver.StringSliceCmd, len(channelIDs))
	for _, cid := range channelIDs {
		cmds[cid] = pipe.SMembers(opCtx, ChannelMembersKey(cid))
	}

	if _, err := pipe.Exec(opCtx); err != nil && !errors.Is(err, redisdriver.Nil) {
		return nil, nil, redis.NewError(err, redis.ScopeMember)
	}

	// 2. Process set results and build a single flat slice of MemberKeyIDs
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

	// 3. Batch fetch ALL member DTOs across ALL channels in a single MGet
	dtosMap, missingMemberKeys, err := c.GetBatch(ctx, allMemberKeys)
	if err != nil {
		return nil, nil, err
	}

	// Index missing member keys for fast lookup
	missingSet := make(map[MemberKeyIDs]struct{}, len(missingMemberKeys))
	for _, mk := range missingMemberKeys {
		missingSet[mk] = struct{}{}
	}

	// 4. Assemble final result map by channel
	found = make(map[fields.ID][]*channel.Member, len(channelIDs))

	for cid, keys := range channelUserKeys {
		channelHasMissingMember := false
		members := make([]*channel.Member, 0, len(keys))

		for _, mk := range keys {
			if _, isMissing := missingSet[mk]; isMissing {
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

		// If any individual member key expired, mark whole channel as miss for DB backfill
		if channelHasMissingMember {
			missingChannelIDs = append(missingChannelIDs, cid)
		} else {
			found[cid] = members
		}
	}

	return found, missingChannelIDs, nil
}
