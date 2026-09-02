package user

import (
	"context"
	"encoding/json"
	"fmt"

	"bonfire-api/internal/fields"
	"bonfire-api/internal/outbox"
)

const (
	EventUpdateUsername = "user.update-username"
	EventUpdatePresence = "user.update-presence"
	EventUpdateProfile  = "user.update-profile"
	EventDisable        = "user.disable"
)

type EventUpdateUsernamePayload struct {
	UserID    string `json:"user_id"`
	Username  string `json:"new_username"`
	UpdatedAt string `json:"updated_at"`
}

type EventUpdatePresencePayload struct {
	UserID    string `json:"user_id"`
	Presence  string `json:"presence"`
	UpdatedAt string `json:"updated_at"`
}

type EventUpdateProfilePayload struct {
	UserID      string  `json:"user_id"`
	DisplayName string  `json:"display_name"`
	Bio         *string `json:"bio,omitempty"`
	AvatarURL   *string `json:"avatar_url,omitempty"`
	BannerColor *string `json:"banner_color,omitempty"`
	UpdatedAt   string  `json:"updated_at"`
}

type EventDisablePayload struct {
	UserID    string `json:"user_id"`
	UpdatedAt string `json:"updated_at"`
}

// UserEvent defines the contract for payloads that target a user ID.
type UserEvent interface {
	GetUserID() string
}

func (e EventUpdateUsernamePayload) GetUserID() string { return e.UserID }
func (e EventUpdatePresencePayload) GetUserID() string { return e.UserID }
func (e EventUpdateProfilePayload) GetUserID() string  { return e.UserID }
func (e EventDisablePayload) GetUserID() string        { return e.UserID }

// NewUpdateUsernameOutboxHandler handles user username updates.
func NewUpdateUsernameOutboxHandler(gw Broadcaster) outbox.Handler {
	return newBroadcastHandler[EventUpdateUsernamePayload](gw, EventUpdateUsername)
}

// NewUpdatePresenceOutboxHandler handles user presence updates.
func NewUpdatePresenceOutboxHandler(gw Broadcaster) outbox.Handler {
	return newBroadcastHandler[EventUpdatePresencePayload](gw, EventUpdatePresence)
}

// NewUpdateProfileOutboxHandler handles user profile updates.
func NewUpdateProfileOutboxHandler(gw Broadcaster) outbox.Handler {
	return newBroadcastHandler[EventUpdateProfilePayload](gw, EventUpdateProfile)
}

// NewDisableOutboxHandler handles user account disablement.
func NewDisableOutboxHandler(gw Broadcaster) outbox.Handler {
	return newBroadcastHandler[EventDisablePayload](gw, EventDisable)
}

// Generic helper factory that encapsulates unmarshaling, validation, and peer broadcast.
func newBroadcastHandler[T UserEvent](gw Broadcaster, eventType string) outbox.Handler {
	return func(ctx context.Context, payload json.RawMessage) error {
		var evt T
		if err := json.Unmarshal(payload, &evt); err != nil {
			return fmt.Errorf("%w: failed to unmarshal %s outbox event", outbox.ErrFatal, eventType)
		}

		userID, err := fields.ParseIDFromString("id", evt.GetUserID())
		if err != nil {
			return fmt.Errorf("%w: invalid user_id in %s payload: %v", outbox.ErrFatal, eventType, err)
		}

		if err := gw.BroadcastToPeers(ctx, userID, eventType, evt); err != nil {
			return fmt.Errorf("failed to broadcast %s for user %s: %w", eventType, userID, err)
		}

		return nil
	}
}
