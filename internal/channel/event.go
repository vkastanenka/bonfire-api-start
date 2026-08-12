package channel

import (
	"time"

	"github.com/google/uuid"
)

const (
	EventMessageCreated     = "channel.message_created"
	EventMessageUpdated     = "channel.message_updated"
	EventMessageDeleted     = "channel.message_deleted"
	EventMessagePinned      = "channel.message_pinned"
	EventReactionAdded      = "channel.reaction_added"
	EventReactionRemoved    = "channel.reaction_removed"
	EventChannelUpdated     = "channel.updated"
	EventChannelDeleted     = "channel.deleted"
	EventChannelReadUpdated = "channel.read_updated"
)

type DeletedPayload struct {
	ChannelID uuid.UUID `json:"channelId"`
	ActorID   uuid.UUID `json:"actorId"`
}

type MessageCreatedPayload struct {
	MessageID uuid.UUID  `json:"messageId"`
	ChannelID uuid.UUID  `json:"channelId"`
	AuthorID  *uuid.UUID `json:"authorId,omitempty"`
	Content   string     `json:"content"`
	ReplyToID *uuid.UUID `json:"replyToId,omitempty"`
	CreatedAt time.Time  `json:"createdAt"`
}

type MessageUpdatedPayload struct {
	MessageID uuid.UUID  `json:"messageId"`
	ChannelID uuid.UUID  `json:"channelId"`
	AuthorID  *uuid.UUID `json:"authorId,omitempty"`
	Content   string     `json:"content"`
	EditedAt  *time.Time `json:"editedAt,omitempty"`
}

type MessagePinnedPayload struct {
	MessageID uuid.UUID `json:"messageId"`
	ChannelID uuid.UUID `json:"channelId"`
	IsPinned  bool      `json:"isPinned"`
}

type MessageDeletedPayload struct {
	MessageID uuid.UUID `json:"messageId"`
	ChannelID uuid.UUID `json:"channelId"`
	ActorID   uuid.UUID `json:"actorId,omitempty"`
	DeletedAt time.Time `json:"deletedAt"`
}

type ReactionPayload struct {
	MessageID uuid.UUID `json:"messageId"`
	ChannelID uuid.UUID `json:"channelId"`
	UserID    uuid.UUID `json:"userId"`
	Emoji     string    `json:"emoji"`
}

type ChannelUpdatedPayload struct {
	ChannelID uuid.UUID `json:"channel_id"`
	ActorID   uuid.UUID `json:"actor_id"`
	Name      *string   `json:"name"`
	IconURL   *string   `json:"icon_url"`
	UpdatedAt time.Time `json:"updated_at"`
}

type ChannelReadUpdatedPayload struct {
	ChannelID         uuid.UUID  `json:"channelId"`
	UserID            uuid.UUID  `json:"userId"`
	LastReadMessageID *uuid.UUID `json:"lastReadMessageId,omitempty"`
	LastReadAt        time.Time  `json:"lastReadAt"`
}

// func RegisterOutboxHandlers(w *outbox.Worker, cacheStore *cache.Store) {
// 	pubToChannel := func(ctx context.Context, channelID uuid.UUID, eventType string, payload any) error {
// 		topic := fmt.Sprintf("channel:%s:events", channelID.String())
// 		wsEvent := map[string]any{
// 			"type": eventType,
// 			"data": payload,
// 		}
// 		return cacheStore.Publish(ctx, topic, wsEvent)
// 	}

// 	w.RegisterHandler(EventMessageCreated, func(ctx context.Context, raw json.RawMessage) error {
// 		var p MessageCreatedPayload
// 		if err := json.Unmarshal(raw, &p); err != nil {
// 			return fmt.Errorf("%w: malformed message created payload: %v", outbox.ErrFatal, err)
// 		}
// 		return pubToChannel(ctx, p.ChannelID, "MESSAGE_CREATED", p)
// 	})

// 	w.RegisterHandler(EventMessageUpdated, func(ctx context.Context, raw json.RawMessage) error {
// 		var p MessageUpdatedPayload
// 		if err := json.Unmarshal(raw, &p); err != nil {
// 			return fmt.Errorf("%w: malformed message updated payload: %v", outbox.ErrFatal, err)
// 		}
// 		return pubToChannel(ctx, p.ChannelID, "MESSAGE_UPDATED", p)
// 	})

// 	w.RegisterHandler(EventMessagePinned, func(ctx context.Context, raw json.RawMessage) error {
// 		var p MessagePinnedPayload
// 		if err := json.Unmarshal(raw, &p); err != nil {
// 			return fmt.Errorf("%w: malformed message pinned payload: %v", outbox.ErrFatal, err)
// 		}
// 		return pubToChannel(ctx, p.ChannelID, "MESSAGE_PINNED", p)
// 	})

// 	w.RegisterHandler(EventMessageDeleted, func(ctx context.Context, raw json.RawMessage) error {
// 		var p MessageDeletedPayload
// 		if err := json.Unmarshal(raw, &p); err != nil {
// 			return fmt.Errorf("%w: malformed message deleted payload: %v", outbox.ErrFatal, err)
// 		}
// 		return pubToChannel(ctx, p.ChannelID, "MESSAGE_DELETED", p)
// 	})

// 	w.RegisterHandler(EventReactionAdded, func(ctx context.Context, raw json.RawMessage) error {
// 		var p ReactionPayload
// 		if err := json.Unmarshal(raw, &p); err != nil {
// 			return fmt.Errorf("%w: malformed reaction added payload: %v", outbox.ErrFatal, err)
// 		}
// 		return pubToChannel(ctx, p.ChannelID, "REACTION_ADDED", p)
// 	})

// 	w.RegisterHandler(EventReactionRemoved, func(ctx context.Context, raw json.RawMessage) error {
// 		var p ReactionPayload
// 		if err := json.Unmarshal(raw, &p); err != nil {
// 			return fmt.Errorf("%w: malformed reaction removed payload: %v", outbox.ErrFatal, err)
// 		}
// 		return pubToChannel(ctx, p.ChannelID, "REACTION_REMOVED", p)
// 	})

// 	w.RegisterHandler(EventChannelUpdated, func(ctx context.Context, raw json.RawMessage) error {
// 		var p ChannelUpdatedPayload
// 		if err := json.Unmarshal(raw, &p); err != nil {
// 			return fmt.Errorf("%w: malformed channel updated payload: %v", outbox.ErrFatal, err)
// 		}
// 		return pubToChannel(ctx, p.ChannelID, "CHANNEL_UPDATED", p)
// 	})

// 	w.RegisterHandler(EventChannelReadUpdated, func(ctx context.Context, raw json.RawMessage) error {
// 		var p ChannelReadUpdatedPayload
// 		if err := json.Unmarshal(raw, &p); err != nil {
// 			return fmt.Errorf("%w: malformed channel read updated payload: %v", outbox.ErrFatal, err)
// 		}
// 		return pubToChannel(ctx, p.ChannelID, "CHANNEL_READ_UPDATED", p)
// 	})
// }
