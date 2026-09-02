package user

import (
	"context"
	"encoding/json"

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

// NewUpdateUsernameOutboxHandler handles user username updates by broadcasting to peers.
func NewUpdateUsernameOutboxHandler(gw outbox.Broadcaster) outbox.Handler {
	return func(ctx context.Context, payload json.RawMessage) error {
		p, err := outbox.ParsePayload[EventUpdateUsernamePayload](EventUpdateUsername, payload)
		if err != nil {
			return err
		}
		return outbox.NewPeersHandler(gw, EventUpdateUsername, p.UserID)(ctx, payload)
	}
}

// NewUpdatePresenceOutboxHandler handles user presence updates by broadcasting to peers.
func NewUpdatePresenceOutboxHandler(gw outbox.Broadcaster) outbox.Handler {
	return func(ctx context.Context, payload json.RawMessage) error {
		p, err := outbox.ParsePayload[EventUpdatePresencePayload](EventUpdatePresence, payload)
		if err != nil {
			return err
		}
		return outbox.NewPeersHandler(gw, EventUpdatePresence, p.UserID)(ctx, payload)
	}
}

// NewUpdateProfileOutboxHandler handles user profile updates by broadcasting to peers.
func NewUpdateProfileOutboxHandler(gw outbox.Broadcaster) outbox.Handler {
	return func(ctx context.Context, payload json.RawMessage) error {
		p, err := outbox.ParsePayload[EventUpdateProfilePayload](EventUpdateProfile, payload)
		if err != nil {
			return err
		}
		return outbox.NewPeersHandler(gw, EventUpdateProfile, p.UserID)(ctx, payload)
	}
}

// NewDisableOutboxHandler handles user account disablement by broadcasting to peers.
func NewDisableOutboxHandler(gw outbox.Broadcaster) outbox.Handler {
	return func(ctx context.Context, payload json.RawMessage) error {
		p, err := outbox.ParsePayload[EventDisablePayload](EventDisable, payload)
		if err != nil {
			return err
		}
		return outbox.NewPeersHandler(gw, EventDisable, p.UserID)(ctx, payload)
	}
}
