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
	store *redis.Store
	ttl   time.Duration
}

func NewChannelCache(store *redis.Store, ttl time.Duration) *ChannelCache {
	return &ChannelCache{
		store: store.WithScope(redis.ScopeChannel),
		ttl:   ttl,
	}
}

func (c *ChannelCache) Get(ctx context.Context, id fields.ID) (*channel.Channel, error) {
	k := channelKey(id)

	var raw string
	err := c.store.Get(ctx, k, &raw)
	if err != nil {
		return nil, c.store.Err(err)
	}
	if raw == "" {
		return nil, nil
	}

	var cached Channel
	if err := json.Unmarshal([]byte(raw), &cached); err != nil {
		return nil, err
	}

	return cached.ToDomain()
}

func (c *ChannelCache) GetBatch(
	ctx context.Context,
	ids []fields.ID,
) (map[fields.ID]*channel.Channel, []fields.ID, error) {
	if len(ids) == 0 {
		return map[fields.ID]*channel.Channel{}, nil, nil
	}

	keys := make([]string, len(ids))
	for i, id := range ids {
		keys[i] = channelKey(id)
	}

	rawValues, err := c.store.MGet(ctx, keys...)
	if err != nil {
		return nil, nil, c.store.Err(err)
	}

	found := make(map[fields.ID]*channel.Channel, len(ids))
	var missing []fields.ID

	for i, raw := range rawValues {
		id := ids[i]

		if raw == nil || raw == "" {
			missing = append(missing, id)
			continue
		}

		rawStr, ok := raw.(string)
		if !ok {
			missing = append(missing, id)
			continue
		}

		var cached Channel
		if err := json.Unmarshal([]byte(rawStr), &cached); err != nil {
			missing = append(missing, id)
			continue
		}

		ch, err := cached.ToDomain()
		if err != nil {
			missing = append(missing, id)
			continue
		}

		found[id] = ch
	}

	return found, missing, nil
}

func (c *ChannelCache) Set(ctx context.Context, ch *channel.Channel) error {
	k := channelKey(ch.ID())
	dto := ParseChannel(ch)

	if err := c.store.Set(ctx, k, dto, c.ttl); err != nil {
		return c.store.Err(err)
	}

	return nil
}

func (c *ChannelCache) SetBatch(ctx context.Context, channels []*channel.Channel) error {
	if len(channels) == 0 {
		return nil
	}

	return c.store.ExecPipeline(ctx, func(pipeCtx context.Context) error {
		for _, ch := range channels {
			k := channelKey(ch.ID())
			dto := ParseChannel(ch)

			if err := c.store.Set(pipeCtx, k, dto, c.ttl); err != nil {
				return err
			}
		}
		return nil
	})
}

func (c *ChannelCache) Invalidate(ctx context.Context, id fields.ID) error {
	return c.store.Delete(ctx, channelKey(id))
}

func (c *ChannelCache) InvalidateBatch(ctx context.Context, ids []fields.ID) error {
	if len(ids) == 0 {
		return nil
	}

	return c.store.ExecPipeline(ctx, func(pipeCtx context.Context) error {
		for _, id := range ids {
			if err := c.store.Delete(pipeCtx, channelKey(id)); err != nil {
				return err
			}
		}
		return nil
	})
}
