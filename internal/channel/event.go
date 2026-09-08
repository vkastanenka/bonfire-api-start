package channel

import (
	"bonfire-api/internal/fields"
	"bonfire-api/internal/outbox"
	"bonfire-api/internal/presence"
	"bonfire-api/internal/user"
	"context"
	"encoding/json"
)

type Broadcaster interface {
	BroadcastToUser(ctx context.Context, userID fields.ID, excludeSessionIDs []fields.ID, eventType string, payload interface{}) error
	BroadcastToUsers(ctx context.Context, recipientIDs []fields.ID, excludeSessionIDs []fields.ID, eventType string, payload interface{}) error
}

const (
	// Channel Lifecycle
	EventChannelCreated = "channel.created"
	EventChannelUpdated = "channel.updated"

	// Membership Events
	EventMembersAdded       = "members.added"
	EventMemberClosedDirect = "member.closed_direct"
	EventMemberUpdated      = "member.updated"
	EventMemberLeft         = "member.left"

	// Message Events
	EventMessageCreated = "message.created"
	EventMessageUpdated = "message.updated"
	EventMessageDeleted = "message.deleted"

	// Reaction Events
	EventReactionToggled = "reaction.toggled"
)

type EventChannelCreatedPayload struct {
	ExcludeSessionID fields.ID                       `json:"exclude_session_id"`
	Channel          *Channel                        `json:"channel"`
	Users            map[fields.ID]*user.User        `json:"users"`
	Presences        map[fields.ID]presence.Presence `json:"presences"`
	MemberIDs        []fields.ID                     `json:"member_ids"`
}

func NewChannelCreatedEventHandler(gw Broadcaster) outbox.Handler {
	return func(ctx context.Context, payload json.RawMessage) error {
		p, err := fields.ParseRawJSON[EventChannelCreatedPayload](payload)
		if err != nil {
			return err
		}

		return gw.BroadcastToUsers(
			ctx,
			p.MemberIDs,
			[]fields.ID{p.ExcludeSessionID},
			EventChannelCreated,
			payload,
		)
	}
}

type EventChannelUpdatedPayload struct {
	ExcludeSessionID fields.ID   `json:"exclude_session_id"`
	Channel          *Channel    `json:"channel"`
	MemberIDs        []fields.ID `json:"member_ids"`
}

type EventChannelUpdatedClientPayload struct {
	Channel *Channel `json:"channel"`
}

func NewChannelUpdatedEventHandler(gw Broadcaster) outbox.Handler {
	return func(ctx context.Context, payload json.RawMessage) error {
		p, err := fields.ParseRawJSON[EventChannelUpdatedPayload](payload)
		if err != nil {
			return err
		}

		clientPayload := EventChannelUpdatedClientPayload{
			Channel: p.Channel,
		}

		clientPayloadBytes, err := json.Marshal(clientPayload)
		if err != nil {
			return err
		}

		return gw.BroadcastToUsers(
			ctx,
			p.MemberIDs,
			[]fields.ID{p.ExcludeSessionID},
			EventChannelUpdated,
			clientPayloadBytes,
		)
	}
}

type EventMembersAddedPayload struct {
	ExcludeSessionID fields.ID                       `json:"exclude_session_id"`
	Channel          *Channel                        `json:"channel"`
	Users            map[fields.ID]*user.User        `json:"users"`
	Presences        map[fields.ID]presence.Presence `json:"presences"`
	MemberIDs        []fields.ID                     `json:"member_ids"`
	SystemMessages   []*Message                      `json:"system_messages"`
}

func NewChannelMembersAddedEventHandler(gw Broadcaster) outbox.Handler {
	return func(ctx context.Context, payload json.RawMessage) error {
		p, err := fields.ParseRawJSON[EventMembersAddedPayload](payload)
		if err != nil {
			return err
		}

		return gw.BroadcastToUsers(
			ctx,
			p.MemberIDs,
			[]fields.ID{p.ExcludeSessionID},
			EventMembersAdded,
			payload,
		)
	}
}

type EventMemberClosedDirectPayload struct {
	ExcludeSessionID fields.ID `json:"exclude_session_id"`
	MemberID         fields.ID `json:"member_id"`
	ChannelID        fields.ID `json:"channel_id"`
}

func NewMemberClosedDirectEventHandler(gw Broadcaster) outbox.Handler {
	return func(ctx context.Context, payload json.RawMessage) error {
		p, err := fields.ParseRawJSON[EventMemberClosedDirectPayload](payload)
		if err != nil {
			return err
		}

		return gw.BroadcastToUser(
			ctx,
			p.MemberID,
			[]fields.ID{p.ExcludeSessionID},
			EventMemberClosedDirect,
			payload,
		)
	}
}

type EventMemberUpdatedPayload struct {
	ExcludeSessionID fields.ID         `json:"exclude_session_id,omitempty"`
	ChannelID        fields.ID         `json:"channel_id"`
	MemberID         fields.ID         `json:"member_id"`
	LastReadID       *fields.ID        `json:"last_read_message_id,omitempty"`
	PinnedAt         *fields.Timestamp `json:"pinned_at,omitempty"`
	MutedUntil       *fields.Timestamp `json:"muted_until,omitempty"`
}

func NewMemberUpdatedEventHandler(gw Broadcaster) outbox.Handler {
	return func(ctx context.Context, payload json.RawMessage) error {
		p, err := fields.ParseRawJSON[EventMemberUpdatedPayload](payload)
		if err != nil {
			return err
		}

		return gw.BroadcastToUser(
			ctx,
			p.MemberID,
			[]fields.ID{p.ExcludeSessionID},
			EventMemberUpdated,
			payload,
		)
	}
}

type EventMemberLeftPayload struct {
	ExcludeSessionID fields.ID   `json:"exclude_session_id"`
	ActorID          fields.ID   `json:"actor_id"`
	ChannelID        fields.ID   `json:"channel_id"`
	MemberIDs        []fields.ID `json:"member_ids"`
	SystemMessage    *Message    `json:"system_message"`
}

func NewMemberLeftEventHandler(gw Broadcaster) outbox.Handler {
	return func(ctx context.Context, payload json.RawMessage) error {
		p, err := fields.ParseRawJSON[EventMemberLeftPayload](payload)
		if err != nil {
			return err
		}

		return gw.BroadcastToUsers(
			ctx,
			p.MemberIDs,
			[]fields.ID{p.ExcludeSessionID},
			EventMemberLeft,
			payload,
		)
	}
}

type EventMessageCreatedPayload struct {
	ExcludeSessionID fields.ID   `json:"exclude_session_id"`
	Message          *Message    `json:"message"`
	Author           *user.User  `json:"author"`
	MemberIDs        []fields.ID `json:"member_ids"`
}

func NewMessageCreated(gw Broadcaster) outbox.Handler {
	return func(ctx context.Context, payload json.RawMessage) error {
		p, err := fields.ParseRawJSON[EventMessageCreatedPayload](payload)
		if err != nil {
			return err
		}

		return gw.BroadcastToUsers(
			ctx,
			p.MemberIDs,
			[]fields.ID{p.ExcludeSessionID},
			EventMessageCreated,
			payload,
		)
	}
}

type EventMessageUpdatedPayload struct {
	MemberIDs        []fields.ID       `json:"member_ids"`
	ExcludeSessionID fields.ID         `json:"exclude_session_id"`
	MessageID        fields.ID         `json:"message_id"`
	MessageContent   *MessageContent   `json:"message_content,omitempty"`
	MessagePinnedAt  *fields.Timestamp `json:"message_pinned_at,omitempty"`
	MessageUpdatedAt fields.Timestamp  `json:"message_updated_at"`
	SystemMessage    *Message          `json:"system_message"`
}

func NewMessageUpdatedHandler(gw Broadcaster) outbox.Handler {
	return func(ctx context.Context, payload json.RawMessage) error {
		p, err := fields.ParseRawJSON[EventMessageUpdatedPayload](payload)
		if err != nil {
			return err
		}

		return gw.BroadcastToUsers(
			ctx,
			p.MemberIDs,
			[]fields.ID{p.ExcludeSessionID},
			EventMessageUpdated,
			payload,
		)
	}
}

type EventMessageDeletedPayload struct {
	MemberIDs        []fields.ID      `json:"member_ids"`
	ExcludeSessionID fields.ID        `json:"exclude_session_id"`
	MessageID        fields.ID        `json:"message_id"`
	MessageDeletedAt fields.Timestamp `json:"message_deleted_at"`
}

func NewMessageDeletedHandler(gw Broadcaster) outbox.Handler {
	return func(ctx context.Context, payload json.RawMessage) error {
		p, err := fields.ParseRawJSON[EventMessageDeletedPayload](payload)
		if err != nil {
			return err
		}

		return gw.BroadcastToUsers(
			ctx,
			p.MemberIDs,
			[]fields.ID{p.ExcludeSessionID},
			EventMessageDeleted,
			payload,
		)
	}
}

type EventReactionToggledPayload struct {
	MemberIDs        []fields.ID      `json:"member_ids"`
	ExcludeSessionID fields.ID        `json:"exclude_session_id"`
	ActorID          fields.ID        `json:"actor_id"`
	MessageID        fields.ID        `json:"message_id"`
	EmojiCount       EmojiCount       `json:"emoji_count"`
	ToggledAt        fields.Timestamp `json:"toggled_at"`
}

func NewReactionToggledHandler(gw Broadcaster) outbox.Handler {
	return func(ctx context.Context, payload json.RawMessage) error {
		p, err := fields.ParseRawJSON[EventReactionToggledPayload](payload)
		if err != nil {
			return err
		}

		return gw.BroadcastToUsers(
			ctx,
			p.MemberIDs,
			[]fields.ID{p.ExcludeSessionID},
			EventReactionToggled,
			payload,
		)
	}
}
