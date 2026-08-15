package cache

import (
	"context"
	"fmt"
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

func memberKey(k MemberKeyIDs) string {
	return fmt.Sprintf("member:%s:%s", k.ChannelID.String(), k.UserID.String())
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
	store *JSONCache[MemberKeyIDs, Member]
	ttl   time.Duration
}

func NewMemberCache(client redisdriver.Cmdable, ttl time.Duration) *MemberCache {
	return &MemberCache{
		store: NewJSONCache[MemberKeyIDs, Member](client, redis.ScopeMember, memberKey),
		ttl:   ttl,
	}
}

func (c *MemberCache) Get(ctx context.Context, channelID, userID fields.ID) (*channel.Member, error) {
	key := MemberKeyIDs{ChannelID: channelID, UserID: userID}
	dto, err := c.store.Get(ctx, key)
	if err != nil || dto == nil {
		return nil, err
	}

	return dto.ToDomain()
}

func (c *MemberCache) GetBatch(
	ctx context.Context,
	keys []MemberKeyIDs,
) (map[MemberKeyIDs]*channel.Member, []MemberKeyIDs, error) {
	dtos, missing, err := c.store.GetBatch(ctx, keys)
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

func (c *MemberCache) Set(ctx context.Context, mem *channel.Member) error {
	if mem == nil {
		return nil
	}
	key := MemberKeyIDs{ChannelID: mem.ChannelID(), UserID: mem.UserID()}
	return c.store.Set(ctx, key, ParseMember(mem), c.ttl)
}

func (c *MemberCache) SetBatch(ctx context.Context, members []*channel.Member) error {
	dtos := make(map[MemberKeyIDs]Member, len(members))
	for _, mem := range members {
		if mem == nil {
			continue
		}
		key := MemberKeyIDs{ChannelID: mem.ChannelID(), UserID: mem.UserID()}
		dtos[key] = ParseMember(mem)
	}

	return c.store.SetBatch(ctx, dtos, c.ttl)
}

func (c *MemberCache) Invalidate(ctx context.Context, channelID, userID fields.ID) error {
	return c.store.Invalidate(ctx, MemberKeyIDs{ChannelID: channelID, UserID: userID})
}

func (c *MemberCache) InvalidateBatch(ctx context.Context, keys []MemberKeyIDs) error {
	return c.store.InvalidateBatch(ctx, keys)
}
