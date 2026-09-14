package user

import (
	"bonfire-api/internal/presence"
	"time"

	"github.com/google/uuid"
)

const (
	EventUpdateUsername = "user.update_username"
	EventUpdatePresence = "user.update_presence"
	EventUpdateProfile  = "user.update_profile"
	EventDisable        = "user.disable"
)

type EventUpdateUsernamePayload struct {
	UserID    uuid.UUID `json:"user_id"`
	Username  string    `json:"new_username"`
	UpdatedAt time.Time `json:"updated_at"`
}

type EventUpdatePresencePayload struct {
	UserID    uuid.UUID         `json:"user_id"`
	Presence  presence.Presence `json:"presence"`
	UpdatedAt time.Time         `json:"updated_at"`
}

type EventUpdateProfilePayload struct {
	UserID      uuid.UUID `json:"user_id"`
	DisplayName string    `json:"display_name"`
	Bio         *string   `json:"bio,omitempty"`
	AvatarURL   *string   `json:"avatar_url,omitempty"`
	BannerColor *string   `json:"banner_color,omitempty"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type EventDisablePayload struct {
	UserID    uuid.UUID `json:"user_id"`
	UpdatedAt time.Time `json:"updated_at"`
}

// type Broadcaster interface {
// 	BroadcastToPeers(ctx context.Context, actorID fields.ID, eventType string, payload interface{}) error
// 	BroadcastToUser(ctx context.Context, actorID, targetUserID fields.ID, eventType string, payload interface{}) error
// }

// func NewUpdateUsernameOutboxHandler(gw outbox.Broadcaster) outbox.Handler {
// 	return func(ctx context.Context, payload json.RawMessage) error {
// 		p, err := fields.ParseRawJSON[EventUpdateUsernamePayload](payload)
// 		if err != nil {
// 			return err
// 		}
// 		return outbox.NewPeersHandler(gw, EventUpdateUsername, p.UserID)(ctx, payload)
// 	}
// }

// func NewUpdatePresenceOutboxHandler(gw outbox.Broadcaster) outbox.Handler {
// 	return func(ctx context.Context, payload json.RawMessage) error {
// 		p, err := fields.ParseRawJSON[EventUpdatePresencePayload](payload)
// 		if err != nil {
// 			return err
// 		}
// 		return outbox.NewPeersHandler(gw, EventUpdatePresence, p.UserID)(ctx, payload)
// 	}
// }

// func NewUpdateProfileOutboxHandler(gw outbox.Broadcaster) outbox.Handler {
// 	return func(ctx context.Context, payload json.RawMessage) error {
// 		p, err := fields.ParseRawJSON[EventUpdateProfilePayload](payload)
// 		if err != nil {
// 			return err
// 		}
// 		return outbox.NewPeersHandler(gw, EventUpdateProfile, p.UserID)(ctx, payload)
// 	}
// }

// func NewDisableOutboxHandler(gw outbox.Broadcaster) outbox.Handler {
// 	return func(ctx context.Context, payload json.RawMessage) error {
// 		p, err := fields.ParseRawJSON[EventDisablePayload](payload)
// 		if err != nil {
// 			return err
// 		}
// 		return outbox.NewPeersHandler(gw, EventDisable, p.UserID)(ctx, payload)
// 	}
// }
