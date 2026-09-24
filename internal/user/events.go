package user

import (
	"bonfire-api/internal/presence"
	"time"

	"github.com/google/uuid"
)

const (
	EventEmailUpdated             = "user.email_updated"
	EventUsernameUpdated          = "user.username_updated"
	EventPasswordUpdated          = "user.password_updated"
	EventPreferredPresenceUpdated = "user.preferred_presence_updated"
	EventProfileUpdated           = "user.profile_updated"
	EventDisabled                 = "user.disabled"
)

type EventEmailUpdatedPayload struct {
	UserID    uuid.UUID `json:"userId"`
	Email     string    `json:"email"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type EventUsernameUpdatedPayload struct {
	UserID    uuid.UUID `json:"userId"`
	Username  string    `json:"username"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type EventPasswordUpdatedPayload struct {
	UserID    uuid.UUID `json:"userId"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type EventPreferredPresenceUpdatedPayload struct {
	UserID    uuid.UUID         `json:"userId"`
	Presence  presence.Presence `json:"presence"`
	UpdatedAt time.Time         `json:"updatedAt"`
}

type EventProfileUpdatedPayload struct {
	UserID      uuid.UUID `json:"userId"`
	DisplayName string    `json:"displayName"`
	Bio         *string   `json:"bio,omitempty"`
	AvatarURL   *string   `json:"avatarUrl,omitempty"`
	BannerColor *string   `json:"bannerColor,omitempty"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type EventDisabledPayload struct {
	UserID    uuid.UUID `json:"userId"`
	UpdatedAt time.Time `json:"updatedAt"`
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
