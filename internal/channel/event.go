package channel

import (
	"bonfire-api/internal/presence"
	"bonfire-api/internal/user"
	"context"
	"time"

	"github.com/google/uuid"
)

type Broadcaster interface {
	BroadcastToUser(ctx context.Context, userID uuid.UUID, excludeSessionIDs []uuid.UUID, eventType string, payload interface{}) error
	BroadcastToUsers(ctx context.Context, recipientIDs []uuid.UUID, excludeSessionIDs []uuid.UUID, eventType string, payload interface{}) error
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
	ExcludeSessionID uuid.UUID                       `json:"exclude_session_id"`
	Channel          *Channel                        `json:"channel"`
	Users            map[uuid.UUID]*user.User        `json:"users"`
	Presences        map[uuid.UUID]presence.Presence `json:"presences"`
	MemberIDs        []uuid.UUID                     `json:"member_ids"`
}

// func NewChannelCreatedEventHandler(gw Broadcaster) outbox.Handler {
// 	return func(ctx context.Context, payload json.RawMessage) error {
// 		p, err := fields.ParseRawJSON[EventChannelCreatedPayload](payload)
// 		if err != nil {
// 			return err
// 		}

// 		return gw.BroadcastToUsers(
// 			ctx,
// 			p.MemberIDs,
// 			[]uuid.UUID{p.ExcludeSessionID},
// 			EventChannelCreated,
// 			payload,
// 		)
// 	}
// }

type EventChannelUpdatedPayload struct {
	ExcludeSessionID uuid.UUID   `json:"exclude_session_id"`
	Channel          *Channel    `json:"channel"`
	MemberIDs        []uuid.UUID `json:"member_ids"`
}

type EventChannelUpdatedClientPayload struct {
	Channel *Channel `json:"channel"`
}

// func NewChannelUpdatedEventHandler(gw Broadcaster) outbox.Handler {
// 	return func(ctx context.Context, payload json.RawMessage) error {
// 		p, err := fields.ParseRawJSON[EventChannelUpdatedPayload](payload)
// 		if err != nil {
// 			return err
// 		}

// 		clientPayload := EventChannelUpdatedClientPayload{
// 			Channel: p.Channel,
// 		}

// 		clientPayloadBytes, err := json.Marshal(clientPayload)
// 		if err != nil {
// 			return err
// 		}

// 		return gw.BroadcastToUsers(
// 			ctx,
// 			p.MemberIDs,
// 			[]uuid.UUID{p.ExcludeSessionID},
// 			EventChannelUpdated,
// 			clientPayloadBytes,
// 		)
// 	}
// }

type EventMembersAddedPayload struct {
	ExcludeSessionID uuid.UUID                       `json:"exclude_session_id"`
	Channel          *Channel                        `json:"channel"`
	Users            map[uuid.UUID]*user.User        `json:"users"`
	Presences        map[uuid.UUID]presence.Presence `json:"presences"`
	MemberIDs        []uuid.UUID                     `json:"member_ids"`
	SystemMessages   []*Message                      `json:"system_messages"`
}

// func NewChannelMembersAddedEventHandler(gw Broadcaster) outbox.Handler {
// 	return func(ctx context.Context, payload json.RawMessage) error {
// 		p, err := fields.ParseRawJSON[EventMembersAddedPayload](payload)
// 		if err != nil {
// 			return err
// 		}

// 		return gw.BroadcastToUsers(
// 			ctx,
// 			p.MemberIDs,
// 			[]uuid.UUID{p.ExcludeSessionID},
// 			EventMembersAdded,
// 			payload,
// 		)
// 	}
// }

type EventMemberClosedDirectPayload struct {
	ExcludeSessionID uuid.UUID `json:"exclude_session_id"`
	MemberID         uuid.UUID `json:"member_id"`
	ChannelID        uuid.UUID `json:"channel_id"`
}

// func NewMemberClosedDirectEventHandler(gw Broadcaster) outbox.Handler {
// 	return func(ctx context.Context, payload json.RawMessage) error {
// 		p, err := fields.ParseRawJSON[EventMemberClosedDirectPayload](payload)
// 		if err != nil {
// 			return err
// 		}

// 		return gw.BroadcastToUser(
// 			ctx,
// 			p.MemberID,
// 			[]uuid.UUID{p.ExcludeSessionID},
// 			EventMemberClosedDirect,
// 			payload,
// 		)
// 	}
// }

type EventMemberUpdatedPayload struct {
	ExcludeSessionID uuid.UUID  `json:"exclude_session_id,omitempty"`
	ChannelID        uuid.UUID  `json:"channel_id"`
	MemberID         uuid.UUID  `json:"member_id"`
	LastReadID       *uuid.UUID `json:"last_read_message_id,omitempty"`
	PinnedAt         *time.Time `json:"pinned_at,omitempty"`
	MutedUntil       *time.Time `json:"muted_until,omitempty"`
}

// func NewMemberUpdatedEventHandler(gw Broadcaster) outbox.Handler {
// 	return func(ctx context.Context, payload json.RawMessage) error {
// 		p, err := fields.ParseRawJSON[EventMemberUpdatedPayload](payload)
// 		if err != nil {
// 			return err
// 		}

// 		return gw.BroadcastToUser(
// 			ctx,
// 			p.MemberID,
// 			[]uuid.UUID{p.ExcludeSessionID},
// 			EventMemberUpdated,
// 			payload,
// 		)
// 	}
// }

type EventMemberLeftPayload struct {
	ExcludeSessionID uuid.UUID   `json:"exclude_session_id"`
	ActorID          uuid.UUID   `json:"actor_id"`
	ChannelID        uuid.UUID   `json:"channel_id"`
	MemberIDs        []uuid.UUID `json:"member_ids"`
	SystemMessage    *Message    `json:"system_message"`
}

// func NewMemberLeftEventHandler(gw Broadcaster) outbox.Handler {
// 	return func(ctx context.Context, payload json.RawMessage) error {
// 		p, err := fields.ParseRawJSON[EventMemberLeftPayload](payload)
// 		if err != nil {
// 			return err
// 		}

// 		return gw.BroadcastToUsers(
// 			ctx,
// 			p.MemberIDs,
// 			[]uuid.UUID{p.ExcludeSessionID},
// 			EventMemberLeft,
// 			payload,
// 		)
// 	}
// }

type EventMessageCreatedPayload struct {
	ExcludeSessionID uuid.UUID   `json:"exclude_session_id"`
	Message          *Message    `json:"message"`
	Author           *user.User  `json:"author"`
	MemberIDs        []uuid.UUID `json:"member_ids"`
}

// func NewMessageCreated(gw Broadcaster) outbox.Handler {
// 	return func(ctx context.Context, payload json.RawMessage) error {
// 		p, err := fields.ParseRawJSON[EventMessageCreatedPayload](payload)
// 		if err != nil {
// 			return err
// 		}

// 		return gw.BroadcastToUsers(
// 			ctx,
// 			p.MemberIDs,
// 			[]uuid.UUID{p.ExcludeSessionID},
// 			EventMessageCreated,
// 			payload,
// 		)
// 	}
// }

type EventMessageUpdatedPayload struct {
	MemberIDs        []uuid.UUID `json:"member_ids"`
	ExcludeSessionID uuid.UUID   `json:"exclude_session_id"`
	MessageID        uuid.UUID   `json:"message_id"`
	MessageContent   *string     `json:"message_content,omitempty"`
	MessagePinnedAt  *time.Time  `json:"message_pinned_at,omitempty"`
	MessageUpdatedAt time.Time   `json:"message_updated_at"`
	SystemMessage    *Message    `json:"system_message"`
}

// func NewMessageUpdatedHandler(gw Broadcaster) outbox.Handler {
// 	return func(ctx context.Context, payload json.RawMessage) error {
// 		p, err := fields.ParseRawJSON[EventMessageUpdatedPayload](payload)
// 		if err != nil {
// 			return err
// 		}

// 		return gw.BroadcastToUsers(
// 			ctx,
// 			p.MemberIDs,
// 			[]uuid.UUID{p.ExcludeSessionID},
// 			EventMessageUpdated,
// 			payload,
// 		)
// 	}
// }

type EventMessageDeletedPayload struct {
	MemberIDs        []uuid.UUID `json:"member_ids"`
	ExcludeSessionID uuid.UUID   `json:"exclude_session_id"`
	MessageID        uuid.UUID   `json:"message_id"`
	MessageDeletedAt time.Time   `json:"message_deleted_at"`
}

// func NewMessageDeletedHandler(gw Broadcaster) outbox.Handler {
// 	return func(ctx context.Context, payload json.RawMessage) error {
// 		p, err := fields.ParseRawJSON[EventMessageDeletedPayload](payload)
// 		if err != nil {
// 			return err
// 		}

// 		return gw.BroadcastToUsers(
// 			ctx,
// 			p.MemberIDs,
// 			[]uuid.UUID{p.ExcludeSessionID},
// 			EventMessageDeleted,
// 			payload,
// 		)
// 	}
// }

type EventReactionToggledPayload struct {
	MemberIDs        []uuid.UUID `json:"member_ids"`
	ExcludeSessionID uuid.UUID   `json:"exclude_session_id"`
	ActorID          uuid.UUID   `json:"actor_id"`
	MessageID        uuid.UUID   `json:"message_id"`
	EmojiCount       EmojiCount  `json:"emoji_count"`
	ToggledAt        time.Time   `json:"toggled_at"`
}

// func NewReactionToggledHandler(gw Broadcaster) outbox.Handler {
// 	return func(ctx context.Context, payload json.RawMessage) error {
// 		p, err := fields.ParseRawJSON[EventReactionToggledPayload](payload)
// 		if err != nil {
// 			return err
// 		}

// 		return gw.BroadcastToUsers(
// 			ctx,
// 			p.MemberIDs,
// 			[]uuid.UUID{p.ExcludeSessionID},
// 			EventReactionToggled,
// 			payload,
// 		)
// 	}
// }
