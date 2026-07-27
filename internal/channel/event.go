package channel

import (
	"context"
	"encoding/json"
	"fmt"

	"bonfire-api/internal/cache"
	"bonfire-api/internal/outbox"

	"github.com/google/uuid"
)

const (
	EventMessageCreated  = "channel.message_created"
	EventMessageUpdated  = "channel.message_updated"
	EventMessageDeleted  = "channel.message_deleted"
	EventReactionAdded   = "channel.reaction_added"
	EventReactionRemoved = "channel.reaction_removed"
	EventChannelUpdated  = "channel.updated"
	EventChannelDeleted  = "channel.deleted"
)

type MessageCreatedPayload struct {
	MessageID uuid.UUID  `json:"message_id"`
	ChannelID uuid.UUID  `json:"channel_id"`
	AuthorID  *uuid.UUID `json:"author_id,omitempty"`
	Content   string     `json:"content"`
	ReplyToID *uuid.UUID `json:"reply_to_id,omitempty"`
}

type MessageUpdatedPayload struct {
	MessageID uuid.UUID `json:"message_id"`
	ChannelID uuid.UUID `json:"channel_id"`
	Content   string    `json:"content"`
}

type MessageDeletedPayload struct {
	MessageID uuid.UUID `json:"message_id"`
	ChannelID uuid.UUID `json:"channel_id"`
}

type ReactionPayload struct {
	MessageID uuid.UUID `json:"message_id"`
	ChannelID uuid.UUID `json:"channel_id"`
	UserID    uuid.UUID `json:"user_id"`
	Emoji     string    `json:"emoji"`
}

type ChannelUpdatedPayload struct {
	ChannelID uuid.UUID `json:"channel_id"`
	Name      *string   `json:"name,omitempty"`
	IconURL   *string   `json:"icon_url,omitempty"`
}

func RegisterOutboxHandlers(w *outbox.Worker, cacheStore *cache.Store) {
	pubToChannel := func(ctx context.Context, channelID uuid.UUID, eventType string, payload any) error {
		topic := fmt.Sprintf("channel:%s:events", channelID.String())
		wsEvent := map[string]any{
			"type": eventType,
			"data": payload,
		}
		return cacheStore.Publish(ctx, topic, wsEvent)
	}

	w.RegisterHandler(EventMessageCreated, func(ctx context.Context, raw json.RawMessage) error {
		var p MessageCreatedPayload
		if err := json.Unmarshal(raw, &p); err != nil {
			return fmt.Errorf("%w: malformed message created payload: %v", outbox.ErrFatal, err)
		}
		return pubToChannel(ctx, p.ChannelID, "MESSAGE_CREATED", p)
	})

	w.RegisterHandler(EventMessageUpdated, func(ctx context.Context, raw json.RawMessage) error {
		var p MessageUpdatedPayload
		if err := json.Unmarshal(raw, &p); err != nil {
			return fmt.Errorf("%w: malformed message updated payload: %v", outbox.ErrFatal, err)
		}
		return pubToChannel(ctx, p.ChannelID, "MESSAGE_UPDATED", p)
	})

	w.RegisterHandler(EventMessageDeleted, func(ctx context.Context, raw json.RawMessage) error {
		var p MessageDeletedPayload
		if err := json.Unmarshal(raw, &p); err != nil {
			return fmt.Errorf("%w: malformed message deleted payload: %v", outbox.ErrFatal, err)
		}
		return pubToChannel(ctx, p.ChannelID, "MESSAGE_DELETED", p)
	})

	w.RegisterHandler(EventReactionAdded, func(ctx context.Context, raw json.RawMessage) error {
		var p ReactionPayload
		if err := json.Unmarshal(raw, &p); err != nil {
			return fmt.Errorf("%w: malformed reaction added payload: %v", outbox.ErrFatal, err)
		}
		return pubToChannel(ctx, p.ChannelID, "REACTION_ADDED", p)
	})

	w.RegisterHandler(EventReactionRemoved, func(ctx context.Context, raw json.RawMessage) error {
		var p ReactionPayload
		if err := json.Unmarshal(raw, &p); err != nil {
			return fmt.Errorf("%w: malformed reaction removed payload: %v", outbox.ErrFatal, err)
		}
		return pubToChannel(ctx, p.ChannelID, "REACTION_REMOVED", p)
	})

	w.RegisterHandler(EventChannelUpdated, func(ctx context.Context, raw json.RawMessage) error {
		var p ChannelUpdatedPayload
		if err := json.Unmarshal(raw, &p); err != nil {
			return fmt.Errorf("%w: malformed channel updated payload: %v", outbox.ErrFatal, err)
		}
		return pubToChannel(ctx, p.ChannelID, "CHANNEL_UPDATED", p)
	})
}
