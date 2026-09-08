package channel

import (
	"bonfire-api/internal/fields"
	"bonfire-api/internal/outbox"
	"bonfire-api/internal/presence"
	"bonfire-api/internal/user"
	"context"
	"encoding/json"
)

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

type EventChannelUpdatedPayload struct {
	ExcludeSessionID fields.ID   `json:"exclude_session_id"`
	Channel          *Channel    `json:"channel"`
	MemberIDs        []fields.ID `json:"member_ids"`
}

type EventChannelUpdatedClientPayload struct {
	ExcludeSessionID fields.ID `json:"excludeSessionId"`
	Channel          *Channel  `json:"channel"`
}

type EventChannelMembersAddedPayload struct {
	ExcludeSessionID fields.ID                       `json:"exclude_session_id"`
	Channel          *Channel                        `json:"channel"`
	Users            map[fields.ID]*user.User        `json:"users"`
	Presences        map[fields.ID]presence.Presence `json:"presences"`
	MemberIDs        []fields.ID                     `json:"member_ids"`
	SystemMessages   []*Message                      `json:"system_messages"`
}

type EventChannelMemberRemovedPayload struct{}

type EventChannelMessageCreatedPayload struct{}
type EventChannelMessageUpdatedPayload struct{}
type EventChannelMessageDeletedPayload struct{}

type EventChannelReactionToggledPayload struct{}

func NewChannelCreatedEventHandler(gw outbox.Broadcaster) outbox.Handler {
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

func NewChannelUpdatedEventHandler(gw outbox.Broadcaster) outbox.Handler {
	return func(ctx context.Context, payload json.RawMessage) error {
		p, err := fields.ParseRawJSON[EventChannelUpdatedPayload](payload)
		if err != nil {
			return err
		}

		clientPayload := EventChannelUpdatedClientPayload{
			ExcludeSessionID: p.ExcludeSessionID,
			Channel:          p.Channel,
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

func NewChannelMembersAddedEventHandler(gw outbox.Broadcaster) outbox.Handler {
	return func(ctx context.Context, payload json.RawMessage) error {
		p, err := fields.ParseRawJSON[EventChannelUpdatedPayload](payload)
		if err != nil {
			return err
		}

		clientPayload := EventChannelUpdatedClientPayload{
			ExcludeSessionID: p.ExcludeSessionID,
			Channel:          p.Channel,
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
