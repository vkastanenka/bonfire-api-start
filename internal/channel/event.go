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
	EventChannelMemberRemoved      = "channel.member.removed"

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

type EventChannelMemberCloseDirectPayload struct {
	ExcludeSessionID fields.ID `json:"exclude_session_id"`
	MemberID         fields.ID `json:"member_id"`
	ChannelID        fields.ID `json:"channel_id"`
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
