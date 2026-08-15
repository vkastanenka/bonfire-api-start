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

func channelKey(id fields.ID) string {
	return fmt.Sprintf("channel:%s", id.String())
}

type Channel struct {
	ID            uuid.UUID `json:"id"`
	Type          int16     `json:"type"`
	Name          string    `json:"name"`
	IconURL       string    `json:"icon_url"`
	LastMessageID uuid.UUID `json:"last_message_id"`
	LastMessageAt time.Time `json:"last_message_at"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func (c Channel) ToDomain() (*channel.Channel, error) {
	id, err := fields.ParseRequiredID("id", c.ID)
	if err != nil {
		return nil, err
	}

	chType, err := channel.ParseChannelType(c.Type)
	if err != nil {
		return nil, err
	}

	name, err := channel.ParseChannelName(c.Name)
	if err != nil {
		return nil, err
	}

	iconURL, err := fields.ParseURL("icon_url", c.IconURL)
	if err != nil {
		return nil, err
	}

	lastMessageID, err := fields.ParseID("last_message_id", c.LastMessageID)
	if err != nil {
		return nil, err
	}

	return channel.ParseChannel(
		id,
		chType,
		name,
		iconURL,
		lastMessageID,
		fields.NewTimestamp(c.LastMessageAt),
		fields.NewTimestamp(c.CreatedAt),
		fields.NewTimestamp(c.UpdatedAt),
	), nil
}

func ParseChannel(ch *channel.Channel) Channel {
	return Channel{
		ID:            ch.ID().UUID(),
		Type:          ch.Type().Int16(),
		Name:          ch.Name().String(),
		IconURL:       ch.IconURL().String(),
		LastMessageID: ch.LastMessageID().UUID(),
		LastMessageAt: ch.LastMessageAt().Time(),
		CreatedAt:     ch.CreatedAt().Time(),
		UpdatedAt:     ch.UpdatedAt().Time(),
	}
}

type ChannelCache struct {
	store *JSONCache[fields.ID, Channel]
	ttl   time.Duration
}

func NewChannelCache(client redisdriver.Cmdable, ttl time.Duration) *ChannelCache {
	return &ChannelCache{
		store: NewJSONCache[fields.ID, Channel](client, redis.ScopeChannel, channelKey),
		ttl:   ttl,
	}
}

func (c *ChannelCache) Get(ctx context.Context, id fields.ID) (*channel.Channel, error) {
	dto, err := c.store.Get(ctx, id)
	if err != nil || dto == nil {
		return nil, err
	}

	return dto.ToDomain()
}

func (c *ChannelCache) GetBatch(
	ctx context.Context,
	ids []fields.ID,
) (map[fields.ID]*channel.Channel, []fields.ID, error) {
	dtos, missing, err := c.store.GetBatch(ctx, ids)
	if err != nil {
		return nil, nil, err
	}

	found := make(map[fields.ID]*channel.Channel, len(dtos))
	for id, dto := range dtos {
		if dto == nil {
			missing = append(missing, id)
			continue
		}

		ch, err := dto.ToDomain()
		if err != nil {
			// Malformed cache hit: mark as missing so callers fall back to DB safely
			missing = append(missing, id)
			continue
		}
		found[id] = ch
	}

	return found, missing, nil
}

func (c *ChannelCache) Set(ctx context.Context, ch *channel.Channel) error {
	if ch == nil {
		return nil
	}
	return c.store.Set(ctx, ch.ID(), ParseChannel(ch), c.ttl)
}

func (c *ChannelCache) SetBatch(ctx context.Context, channels []*channel.Channel) error {
	dtos := make(map[fields.ID]Channel, len(channels))
	for _, ch := range channels {
		if ch == nil {
			continue
		}
		dtos[ch.ID()] = ParseChannel(ch)
	}

	return c.store.SetBatch(ctx, dtos, c.ttl)
}

func (c *ChannelCache) Invalidate(ctx context.Context, id fields.ID) error {
	return c.store.Invalidate(ctx, id)
}

func (c *ChannelCache) InvalidateBatch(ctx context.Context, ids []fields.ID) error {
	return c.store.InvalidateBatch(ctx, ids)
}
