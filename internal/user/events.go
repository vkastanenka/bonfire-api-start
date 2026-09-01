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
func NewUpdatePresenceOutboxHandler(pub *gateway.Publisher, cache Cache) outbox.Handler {
	return func(ctx context.Context, payload json.RawMessage) error {
		var evt EventUpdatePreferredPresencePayload
		if err := json.Unmarshal(payload, &evt); err != nil {
			return fmt.Errorf("%w: failed to unmarshal update presence outbox event", outbox.ErrFatal)
		}

		userID, err := fields.ParseIDFromString("id", evt.UserID)
		if err != nil {
			return fmt.Errorf("%w: invalid user_id in payload: %v", outbox.ErrFatal, err)
		}

		nodeToUsers, err := cache.GetUpdateRecipientNodes(ctx, userID)
		if err != nil {
			return fmt.Errorf("failed to fetch recipient nodes for presence update of user %s: %w", userID, err)
		}
		if len(nodeToUsers) == 0 {
			return nil
		}

		nodeEvents := make(map[fields.ID]gateway.NodeEvent, len(nodeToUsers))
		for nodeID, targetUserIDs := range nodeToUsers {
			nodeEvents[nodeID] = gateway.NodeEvent{
				UserIDs: fields.UUIDs(targetUserIDs),
				Type:    EventUpdatePresence,
				Data:    payload,
			}
		}

		if err := pub.PublishBatchNodeEvents(ctx, nodeEvents); err != nil {
			return fmt.Errorf("failed to publish batch node events for presence: %w", err)
		}

		return nil
	}
}

// NewUpdateProfileOutboxHandler handles broadcasts when a user changes profile info (DisplayName, Bio, Avatar, etc.).
func NewUpdateProfileOutboxHandler(pub *gateway.Publisher, cache Cache) outbox.Handler {
	return func(ctx context.Context, payload json.RawMessage) error {
		var evt EventUpdateProfilePayload
		if err := json.Unmarshal(payload, &evt); err != nil {
			return fmt.Errorf("%w: failed to unmarshal update profile outbox event", outbox.ErrFatal)
		}

		userID, err := fields.ParseIDFromString("id", evt.UserID)
		if err != nil {
			return fmt.Errorf("%w: invalid user_id in payload: %v", outbox.ErrFatal, err)
		}

		nodeToUsers, err := cache.GetUpdateRecipientNodes(ctx, userID)
		if err != nil {
			return fmt.Errorf("failed to fetch recipient nodes for profile update of user %s: %w", userID, err)
		}
		if len(nodeToUsers) == 0 {
			return nil
		}

		nodeEvents := make(map[fields.ID]gateway.NodeEvent, len(nodeToUsers))
		for nodeID, targetUserIDs := range nodeToUsers {
			nodeEvents[nodeID] = gateway.NodeEvent{
				UserIDs: fields.UUIDs(targetUserIDs),
				Type:    EventUpdateProfile,
				Data:    payload,
			}
		}

		if err := pub.PublishBatchNodeEvents(ctx, nodeEvents); err != nil {
			return fmt.Errorf("failed to publish batch node events for profile: %w", err)
		}

		return nil
	}
}

// NewDisableOutboxHandler targets only the disabled user's active node connections so the Gateway can terminate their WebSocket sessions.
func NewDisableOutboxHandler(pub *gateway.Publisher, cache Cache) outbox.Handler {
	return func(ctx context.Context, payload json.RawMessage) error {
		var evt EventDisablePayload
		if err := json.Unmarshal(payload, &evt); err != nil {
			return fmt.Errorf("%w: failed to unmarshal disable outbox event", outbox.ErrFatal)
		}

		userID, err := fields.ParseIDFromString("id", evt.UserID)
		if err != nil {
			return fmt.Errorf("%w: invalid user_id in payload: %v", outbox.ErrFatal, err)
		}

		// 1. Fetch only the disabled user's active gateway nodes
		nodeIDs, err := cache.GetUserNodes(ctx, userID)
		if err != nil {
			return fmt.Errorf("failed to fetch active nodes for disabled user %s: %w", userID, err)
		}
		if len(nodeIDs) == 0 {
			return nil
		}

		// 2. Target only this user's ID across their node connections
		targetUUIDs := fields.UUIDs([]fields.ID{userID})
		nodeEvents := make(map[fields.ID]gateway.NodeEvent, len(nodeIDs))
		for _, nodeID := range nodeIDs {
			nodeEvents[nodeID] = gateway.NodeEvent{
				UserIDs: targetUUIDs,
				Type:    EventDisable,
				Data:    payload,
			}
		}

		if err := pub.PublishBatchNodeEvents(ctx, nodeEvents); err != nil {
			return fmt.Errorf("failed to publish disable events to gateway: %w", err)
		}

		return nil
	}
}
