package channel

import (
	"bonfire-api/internal/outbox"
	"bonfire-api/internal/presence"
	"bonfire-api/internal/user"
	"context"
	"time"

	"github.com/google/uuid"
)

type Worker interface {
	RegisterHandler(eventType string, handler outbox.Handler)
}

type Broadcaster interface {
	BroadcastToUser(ctx context.Context, recipientID uuid.UUID, eventType string, payload any, excludeSessionIDs ...uuid.UUID) error
	BroadcastToUsers(ctx context.Context, recipientIDs []uuid.UUID, eventType string, payload any, excludeSessionIDs ...uuid.UUID) error
}

const (
	EventChannelCreated     = "channel.created"
	EventChannelUpdated     = "channel.updated"
	EventMembersAdded       = "members.added"
	EventMemberClosedDirect = "member.closed_direct"
	EventMemberUpdated      = "member.updated"
	EventMemberLeft         = "member.left"
	EventMessageCreated     = "message.created"
	EventMessageUpdated     = "message.updated"
	EventMessageDeleted     = "message.deleted"
	EventReactionToggled    = "reaction.toggled"
)

func RegisterEvents(w Worker, b Broadcaster) {
	w.RegisterHandler(EventChannelCreated, newChannelCreatedHandler(b))
	w.RegisterHandler(EventChannelUpdated, newChannelUpdatedHandler(b))
	w.RegisterHandler(EventMembersAdded, newMembersAddedHandler(b))
	w.RegisterHandler(EventMemberClosedDirect, newMemberClosedDirectHandler(b))
	w.RegisterHandler(EventMemberUpdated, newMemberUpdatedHandler(b))
	w.RegisterHandler(EventMemberLeft, newMemberLeftHandler(b))
	w.RegisterHandler(EventMessageCreated, newMessageCreatedHandler(b))
	w.RegisterHandler(EventMessageUpdated, newMessageUpdatedHandler(b))
	w.RegisterHandler(EventMessageDeleted, newMessageDeletedHandler(b))
	w.RegisterHandler(EventReactionToggled, newReactionTogggledHandler(b))
}

type EventChannelCreatedPayload struct {
	Channel   ChannelView                     `json:"channel"`
	Members   map[uuid.UUID]MemberView        `json:"members"`
	Users     map[uuid.UUID]user.Summary      `json:"users"`
	Presences map[uuid.UUID]presence.Presence `json:"presences"`
	CreatedAt time.Time                       `json:"createdAt"`
}

func newChannelCreatedHandler(b Broadcaster) outbox.Handler {
	return outbox.BindHandler(func(ctx context.Context, m outbox.Metadata, p EventChannelCreatedPayload) error {
		return b.BroadcastToUsers(ctx, m.RecipientIDs, EventChannelCreated, p)
	})
}

type EventChannelUpdatedPayload struct {
	Channel        ChannelView              `json:"channel"`
	Members        map[uuid.UUID]MemberView `json:"members,omitempty"`
	SystemMessages []MessageView            `json:"systemMessages,omitempty"`
	CreatedAt      time.Time                `json:"createdAt"`
}

func newChannelUpdatedHandler(b Broadcaster) outbox.Handler {
	return outbox.BindHandler(func(ctx context.Context, m outbox.Metadata, p EventChannelUpdatedPayload) error {
		return b.BroadcastToUsers(ctx, m.RecipientIDs, EventChannelUpdated, p)
	})
}

type EventMembersAddedPayload struct {
	Channel        ChannelView                     `json:"channel"`
	Members        map[uuid.UUID]MemberView        `json:"members"`
	MemberIDs      []uuid.UUID                     `json:"memberIDs"`
	Users          map[uuid.UUID]user.Summary      `json:"users"`
	Presences      map[uuid.UUID]presence.Presence `json:"presences"`
	SystemMessages []MessageView                   `json:"systemMessages"`
	CreatedAt      time.Time                       `json:"createdAt"`
}

func newMembersAddedHandler(b Broadcaster) outbox.Handler {
	return outbox.BindHandler(func(ctx context.Context, m outbox.Metadata, p EventMembersAddedPayload) error {
		return b.BroadcastToUsers(ctx, m.RecipientIDs, EventMembersAdded, p)
	})
}

type EventMemberClosedDirectPayload struct {
	ChannelID uuid.UUID `json:"channelId"`
	CreatedAt time.Time `json:"createdAt"`
}

func newMemberClosedDirectHandler(b Broadcaster) outbox.Handler {
	return outbox.BindHandler(func(ctx context.Context, m outbox.Metadata, p EventMemberClosedDirectPayload) error {
		return b.BroadcastToUser(ctx, m.ActorID, EventMemberClosedDirect, p)
	})
}

type EventMemberUpdatedPayload struct {
	ChannelID  uuid.UUID  `json:"channelId"`
	LastReadID *uuid.UUID `json:"last_read_message_id,omitempty"`
	PinnedAt   *time.Time `json:"pinned_at,omitempty"`
	MutedUntil *time.Time `json:"muted_until,omitempty"`
	CreatedAt  time.Time  `json:"createdAt"`
}

func newMemberUpdatedHandler(b Broadcaster) outbox.Handler {
	return outbox.BindHandler(func(ctx context.Context, m outbox.Metadata, p EventMemberUpdatedPayload) error {
		return b.BroadcastToUser(ctx, m.ActorID, EventMemberUpdated, p)
	})
}

type EventMemberLeftPayload struct {
	ChannelID     uuid.UUID                 `json:"channelId"`
	MemberID      uuid.UUID                 `json:"memberId"`
	Members       *map[uuid.UUID]MemberView `json:"members,omitempty"`
	SystemMessage *MessageView              `json:"systemMessage,omitempty"`
	CreatedAt     time.Time                 `json:"createdAt"`
}

func newMemberLeftHandler(b Broadcaster) outbox.Handler {
	return outbox.BindHandler(func(ctx context.Context, m outbox.Metadata, p EventMemberLeftPayload) error {
		return b.BroadcastToUsers(ctx, m.RecipientIDs, EventMemberLeft, p)
	})
}

type EventMessageCreatedPayload struct {
	Channel   ChannelView              `json:"channel"`
	Members   map[uuid.UUID]MemberView `json:"members"`
	Message   MessageView              `json:"message"`
	Author    *user.Summary            `json:"author,omitempty"`
	CreatedAt time.Time                `json:"createdAt"`
}

func newMessageCreatedHandler(b Broadcaster) outbox.Handler {
	return outbox.BindHandler(func(ctx context.Context, m outbox.Metadata, p EventMemberLeftPayload) error {
		return b.BroadcastToUsers(ctx, m.RecipientIDs, EventMemberLeft, p)
	})
}

type EventMessageUpdatedPayload struct {
	MessageID       uuid.UUID    `json:"messageId"`
	MessageContent  *string      `json:"content,omitempty"`
	MessagePinnedAt *time.Time   `json:"pinnedAt,omitempty"`
	SystemMessage   *MessageView `json:"systemMessage,omitempty"`
	CreatedAt       time.Time    `json:"createdAt"`
}

func newMessageUpdatedHandler(b Broadcaster) outbox.Handler {
	return outbox.BindHandler(func(ctx context.Context, m outbox.Metadata, p EventMessageUpdatedPayload) error {
		return b.BroadcastToUsers(ctx, m.RecipientIDs, EventMessageUpdated, p)
	})
}

type EventMessageDeletedPayload struct {
	MessageID uuid.UUID `json:"messageId"`
	CreatedAt time.Time `json:"createdAt"`
}

func newMessageDeletedHandler(b Broadcaster) outbox.Handler {
	return outbox.BindHandler(func(ctx context.Context, m outbox.Metadata, p EventMessageDeletedPayload) error {
		return b.BroadcastToUsers(ctx, m.RecipientIDs, EventMessageDeleted, p)
	})
}

type EventReactionToggledPayload struct {
	ActorID    uuid.UUID  `json:"actorId"`
	MessageID  uuid.UUID  `json:"messageId"`
	EmojiCount EmojiCount `json:"emojiCount"`
	CreatedAt  time.Time  `json:"createdAt"`
}

func newReactionTogggledHandler(b Broadcaster) outbox.Handler {
	return outbox.BindHandler(func(ctx context.Context, m outbox.Metadata, p EventReactionToggledPayload) error {
		return b.BroadcastToUsers(ctx, m.RecipientIDs, EventReactionToggled, p)
	})
}
