package user

import (
	"bonfire-api/internal/fields"
	"bonfire-api/internal/gateway"
	"bonfire-api/internal/outbox"
	"context"
	"encoding/json"
	"fmt"
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
	UserID   string `json:"user_id"`
	Presence string `json:"presence"`
}

type EventUpdatePreferredPresencePayload struct {
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

// NewUpdatePresenceOutboxHandler handles broadcasts for preferred presence changes (e.g. Online, Away, DND).
func NewUpdatePresenceOutboxHandler(gw *gateway.Service) outbox.Handler {
	return func(ctx context.Context, payload json.RawMessage) error {
		var evt EventUpdatePreferredPresencePayload
		if err := json.Unmarshal(payload, &evt); err != nil {
			return fmt.Errorf("%w: failed to unmarshal update presence outbox event", outbox.ErrFatal)
		}

		userID, err := fields.ParseIDFromString("id", evt.UserID)
		if err != nil {
			return fmt.Errorf("%w: invalid user_id in payload: %v", outbox.ErrFatal, err)
		}

		if err := gw.BroadcastToPeers(ctx, userID, EventUpdatePresence, evt); err != nil {
			return fmt.Errorf("failed to broadcast presence update for user %s: %w", userID, err)
		}

		return nil
	}
}

// NewUpdateProfileOutboxHandler handles broadcasts when a user changes profile info (DisplayName, Bio, Avatar, etc.).
func NewUpdateProfileOutboxHandler(gw *gateway.Service) outbox.Handler {
	return func(ctx context.Context, payload json.RawMessage) error {
		var evt EventUpdateProfilePayload
		if err := json.Unmarshal(payload, &evt); err != nil {
			return fmt.Errorf("%w: failed to unmarshal update profile outbox event", outbox.ErrFatal)
		}

		userID, err := fields.ParseIDFromString("id", evt.UserID)
		if err != nil {
			return fmt.Errorf("%w: invalid user_id in payload: %v", outbox.ErrFatal, err)
		}

		if err := gw.BroadcastToPeers(ctx, userID, EventUpdateProfile, evt); err != nil {
			return fmt.Errorf("failed to broadcast profile update for user %s: %w", userID, err)
		}

		return nil
	}
}

// NewDisableOutboxHandler targets only the disabled user's active node connections so the Gateway can terminate their WebSocket sessions.
func NewDisableOutboxHandler(gw *gateway.Service) outbox.Handler {
	return func(ctx context.Context, payload json.RawMessage) error {
		var evt EventDisablePayload
		if err := json.Unmarshal(payload, &evt); err != nil {
			return fmt.Errorf("%w: failed to unmarshal disable outbox event", outbox.ErrFatal)
		}

		userID, err := fields.ParseIDFromString("id", evt.UserID)
		if err != nil {
			return fmt.Errorf("%w: invalid user_id in payload: %v", outbox.ErrFatal, err)
		}

		if err := gw.BroadcastToUser(ctx, userID, userID, EventDisable, evt); err != nil {
			return fmt.Errorf("failed to broadcast disable event for user %s: %w", userID, err)
		}

		return nil
	}
}
