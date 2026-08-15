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

// messageKey generates a Redis key unique to a message entity.
func messageKey(id fields.ID) string {
	return fmt.Sprintf("message:%s", id.String())
}

// Message is the serializable DTO for Redis caching.
type Message struct {
	ID                 uuid.UUID       `json:"id"`
	ChannelID          uuid.UUID       `json:"channel_id"`
	AuthorID           uuid.UUID       `json:"author_id"`
	MsgType            int16           `json:"msg_type"`
	Content            json.RawMessage `json:"content"`
	SystemMetadata     json.RawMessage `json:"system_metadata"`
	ReplyToMessageID   uuid.UUID       `json:"reply_to_message_id"`
	ForwardedMessageID uuid.UUID       `json:"forwarded_message_id"`
	ForwardedChannelID uuid.UUID       `json:"forwarded_channel_id"`
	PinnedAt           time.Time       `json:"pinned_at"`
	CreatedAt          time.Time       `json:"created_at"`
	UpdatedAt          time.Time       `json:"updated_at"`
	EditedAt           time.Time       `json:"edited_at"`
}

// ToDomain converts the cached DTO back into the rich channel.Message domain entity.
func (m Message) ToDomain() (*channel.Message, error) {
	id, err := fields.ParseRequiredID("id", m.ID)
	if err != nil {
		return nil, err
	}

	channelID, err := fields.ParseRequiredID("channel_id", m.ChannelID)
	if err != nil {
		return nil, err
	}

	authorID, err := fields.ParseRequiredID("author_id", m.AuthorID)
	if err != nil {
		return nil, err
	}

	replyToMessageID, err := fields.ParseID("reply_to_message_id", m.ReplyToMessageID)
	if err != nil {
		return nil, err
	}

	forwardedMessageID, err := fields.ParseID("forwarded_message_id", m.ForwardedMessageID)
	if err != nil {
		return nil, err
	}

	forwardedChannelID, err := fields.ParseID("forwarded_channel_id", m.ForwardedChannelID)
	if err != nil {
		return nil, err
	}

	var content channel.MessageContent
	if len(m.Content) > 0 {
		if err := json.Unmarshal(m.Content, &content); err != nil {
			return nil, fmt.Errorf("failed to unmarshal message content: %w", err)
		}
	}

	var sysMeta fields.JSON
	if len(m.SystemMetadata) > 0 {
		var err error
		sysMeta, err = fields.ParseJSON("system_metadata", m.SystemMetadata)
		if err != nil {
			return nil, fmt.Errorf("failed to parse system metadata: %w", err)
		}
	}

	msgType, err := channel.ParseMessageType(m.MsgType)
	if err != nil {
		return nil, err
	}

	return channel.ParseMessage(
		id,
		channelID,
		authorID,
		msgType,
		content,
		sysMeta,
		replyToMessageID,
		forwardedMessageID,
		forwardedChannelID,
		fields.NewTimestamp(m.PinnedAt),
		fields.NewTimestamp(m.CreatedAt),
		fields.NewTimestamp(m.UpdatedAt),
		fields.NewTimestamp(m.EditedAt),
	), nil
}

func ParseMessage(m *channel.Message) (Message, error) {
	contentBytes, err := json.Marshal(m.Content())
	if err != nil {
		return Message{}, fmt.Errorf("failed to marshal message content: %w", err)
	}

	return Message{
		ID:                 m.ID().UUID(),
		ChannelID:          m.ChannelID().UUID(),
		AuthorID:           m.AuthorID().UUID(),
		MsgType:            m.Type().Int16(),
		Content:            contentBytes,
		SystemMetadata:     m.SystemMetadata().Bytes(),
		ReplyToMessageID:   m.ReplyToMessageID().UUID(),
		ForwardedMessageID: m.ForwardedMessageID().UUID(),
		ForwardedChannelID: m.ForwardedChannelID().UUID(),
		PinnedAt:           m.PinnedAt().Time(),
		CreatedAt:          m.CreatedAt().Time(),
		UpdatedAt:          m.UpdatedAt().Time(),
		EditedAt:           m.EditedAt().Time(),
	}, nil
}

type MessageCache struct {
	store *redis.Store
	ttl   time.Duration
}

func NewMessageCache(store *redis.Store, ttl time.Duration) *MessageCache {
	return &MessageCache{
		store: store.WithScope(redis.ScopeMessage),
		ttl:   ttl,
	}
}

func (c *MessageCache) Get(ctx context.Context, id fields.ID) (*channel.Message, error) {
	k := messageKey(id)

	var raw string
	err := c.store.Get(ctx, k, &raw)
	if err != nil {
		return nil, c.store.Err(err)
	}
	if raw == "" {
		return nil, nil
	}

	var cached Message
	if err := json.Unmarshal([]byte(raw), &cached); err != nil {
		return nil, err
	}

	return cached.ToDomain()
}

func (c *MessageCache) GetBatch(
	ctx context.Context,
	ids []fields.ID,
) (map[fields.ID]*channel.Message, []fields.ID, error) {
	if len(ids) == 0 {
		return map[fields.ID]*channel.Message{}, nil, nil
	}

	redisKeys := make([]string, len(ids))
	for i, id := range ids {
		redisKeys[i] = messageKey(id)
	}

	rawValues, err := c.store.MGet(ctx, redisKeys...)
	if err != nil {
		return nil, nil, c.store.Err(err)
	}

	found := make(map[fields.ID]*channel.Message, len(ids))
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

		var cached Message
		if err := json.Unmarshal([]byte(rawStr), &cached); err != nil {
			missing = append(missing, id)
			continue
		}

		msg, err := cached.ToDomain()
		if err != nil {
			missing = append(missing, id)
			continue
		}

		found[id] = msg
	}

	return found, missing, nil
}

func (c *MessageCache) Set(ctx context.Context, msg *channel.Message) error {
	k := messageKey(msg.ID())
	dto, err := ParseMessage(msg)
	if err != nil {
		return err
	}

	if err := c.store.Set(ctx, k, dto, c.ttl); err != nil {
		return c.store.Err(err)
	}

	return nil
}

func (c *MessageCache) SetBatch(ctx context.Context, messages []*channel.Message) error {
	if len(messages) == 0 {
		return nil
	}

	dtos := make([]Message, len(messages))
	for i, msg := range messages {
		dto, err := ParseMessage(msg)
		if err != nil {
			return err
		}
		dtos[i] = dto
	}

	return c.store.ExecPipeline(ctx, func(pipeCtx context.Context) error {
		for _, dto := range dtos {
			k := messageKey(fields.NewID(dto.ID))
			if err := c.store.Set(pipeCtx, k, dto, c.ttl); err != nil {
				return err
			}
		}
		return nil
	})
}

func (c *MessageCache) Invalidate(ctx context.Context, id fields.ID) error {
	return c.store.Delete(ctx, messageKey(id))
}

func (c *MessageCache) InvalidateBatch(ctx context.Context, ids []fields.ID) error {
	if len(ids) == 0 {
		return nil
	}

	return c.store.ExecPipeline(ctx, func(pipeCtx context.Context) error {
		for _, id := range ids {
			if err := c.store.Delete(pipeCtx, messageKey(id)); err != nil {
				return err
			}
		}
		return nil
	})
}
