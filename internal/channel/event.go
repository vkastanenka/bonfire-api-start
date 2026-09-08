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
	EventChannelMembersAdded       = "channel.members.added"
	EventChannelMemberClosedDirect = "channel.member.closed_direct"
	EventChannelMemberUpdated      = "channel.member.updated"
	EventChannelMemberLeft         = "channel.member.left"

	// Message Events
	EventChannelMessageCreated = "channel.message.created"
	EventChannelMessageUpdated = "channel.message.updated"
	EventChannelMessageDeleted = "channel.message.deleted"

	// Reaction Events
	EventChannelReactionToggled = "channel.reaction.toggled"
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

type EventChannelMembersAddedPayload struct {
	ExcludeSessionID fields.ID                       `json:"exclude_session_id"`
	Channel          *Channel                        `json:"channel"`
	Users            map[fields.ID]*user.User        `json:"users"`
	Presences        map[fields.ID]presence.Presence `json:"presences"`
	MemberIDs        []fields.ID                     `json:"member_ids"`
	SystemMessages   []*Message                      `json:"system_messages"`
}

func NewChannelMembersAddedEventHandler(gw Broadcaster) outbox.Handler {
	return func(ctx context.Context, payload json.RawMessage) error {
		p, err := fields.ParseRawJSON[EventChannelMembersAddedPayload](payload)
		if err != nil {
			return err
		}

		return gw.BroadcastToUsers(
			ctx,
			p.MemberIDs,
			[]fields.ID{p.ExcludeSessionID},
			EventChannelMembersAdded,
			payload,
		)
	}
}

type EventChannelMemberClosedDirectPayload struct {
	ExcludeSessionID fields.ID `json:"exclude_session_id"`
	MemberID         fields.ID `json:"member_id"`
	ChannelID        fields.ID `json:"channel_id"`
}

func NewMemberClosedDirectEventHandler(gw Broadcaster) outbox.Handler {
	return func(ctx context.Context, payload json.RawMessage) error {
		p, err := fields.ParseRawJSON[EventChannelMemberClosedDirectPayload](payload)
		if err != nil {
			return err
		}

		return gw.BroadcastToUser(
			ctx,
			p.MemberID,
			[]fields.ID{p.ExcludeSessionID},
			EventChannelMemberClosedDirect,
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
			EventChannelMemberUpdated,
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
			EventChannelMemberLeft,
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
			EventChannelMessageCreated,
			payload,
		)
	}
}
