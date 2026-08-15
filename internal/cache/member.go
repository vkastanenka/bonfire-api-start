package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"bonfire-api/internal/channel"
	"bonfire-api/internal/fields"
	"bonfire-api/internal/redis"

	"github.com/google/uuid"
)

func memberKey(channelID, userID fields.ID) string {
	return fmt.Sprintf("member:%s:%s", channelID.String(), userID.String())
}

type MemberKeyIDs struct {
	ChannelID fields.ID
	UserID    fields.ID
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
	store *redis.Store
	ttl   time.Duration
}

func NewMemberCache(store *redis.Store, ttl time.Duration) *MemberCache {
	return &MemberCache{
		store: store.WithScope(redis.ScopeChannelMember),
		ttl:   ttl,
	}
}

func (c *MemberCache) Get(ctx context.Context, channelID, userID fields.ID) (*channel.Member, error) {
	k := memberKey(channelID, userID)

	var raw string
	err := c.store.Get(ctx, k, &raw)
	if err != nil {
		return nil, c.store.Err(err)
	}
	if raw == "" {
		return nil, nil
	}

	var cached Member
	if err := json.Unmarshal([]byte(raw), &cached); err != nil {
		return nil, err
	}

	return cached.ToDomain()
}

func (c *MemberCache) GetBatch(
	ctx context.Context,
	keys []MemberKeyIDs,
) (map[MemberKeyIDs]*channel.Member, []MemberKeyIDs, error) {
	if len(keys) == 0 {
		return map[MemberKeyIDs]*channel.Member{}, nil, nil
	}

	redisKeys := make([]string, len(keys))
	for i, k := range keys {
		redisKeys[i] = memberKey(k.ChannelID, k.UserID)
	}

	rawValues, err := c.store.MGet(ctx, redisKeys...)
	if err != nil {
		return nil, nil, c.store.Err(err)
	}

	found := make(map[MemberKeyIDs]*channel.Member, len(keys))
	var missing []MemberKeyIDs

	for i, raw := range rawValues {
		keyIDs := keys[i]

		if raw == nil || raw == "" {
			missing = append(missing, keyIDs)
			continue
		}

		rawStr, ok := raw.(string)
		if !ok {
			missing = append(missing, keyIDs)
			continue
		}

		var cached Member
		if err := json.Unmarshal([]byte(rawStr), &cached); err != nil {
			missing = append(missing, keyIDs)
			continue
		}

		mem, err := cached.ToDomain()
		if err != nil {
			missing = append(missing, keyIDs)
			continue
		}

		found[keyIDs] = mem
	}

	return found, missing, nil
}

func (c *MemberCache) Set(ctx context.Context, mem *channel.Member) error {
	k := memberKey(mem.ChannelID(), mem.UserID())
	dto := ParseMember(mem)

	if err := c.store.Set(ctx, k, dto, c.ttl); err != nil {
		return c.store.Err(err)
	}

	return nil
}

func (c *MemberCache) SetBatch(ctx context.Context, members []*channel.Member) error {
	if len(members) == 0 {
		return nil
	}

	return c.store.ExecPipeline(ctx, func(pipeCtx context.Context) error {
		for _, mem := range members {
			k := memberKey(mem.ChannelID(), mem.UserID())
			dto := ParseMember(mem)

			if err := c.store.Set(pipeCtx, k, dto, c.ttl); err != nil {
				return err
			}
		}
		return nil
	})
}

func (c *MemberCache) Invalidate(ctx context.Context, channelID, userID fields.ID) error {
	return c.store.Delete(ctx, memberKey(channelID, userID))
}

func (c *MemberCache) InvalidateBatch(ctx context.Context, keys []MemberKeyIDs) error {
	if len(keys) == 0 {
		return nil
	}

	return c.store.ExecPipeline(ctx, func(pipeCtx context.Context) error {
		for _, k := range keys {
			if err := c.store.Delete(pipeCtx, memberKey(k.ChannelID, k.UserID)); err != nil {
				return err
			}
		}
		return nil
	})
}
