package user

import (
	"bonfire-api/internal/outbox"
	"bonfire-api/internal/presence"
	"context"
	"time"

	"github.com/google/uuid"
)

type Worker interface {
	RegisterHandler(eventType string, handler outbox.Handler)
}

type Broadcaster interface {
	BroadcastToUser(ctx context.Context, recipientID uuid.UUID, eventType string, payload any, excludeSessionIDs ...uuid.UUID) error
	BroadcastToUsers(ctx context.Context, recipientIDs []uuid.UUID, eventType string, payload any, excludeSessionIDs ...uuid.UUID) error
}

const (
	EventEmailUpdated             = "user.email_updated"
	EventUsernameUpdated          = "user.username_updated"
	EventPasswordUpdated          = "user.password_updated"
	EventPreferredPresenceUpdated = "user.preferred_presence_updated"
	EventProfileUpdated           = "user.profile_updated"
	EventDisabled                 = "user.disabled"
)

func RegisterEvents(w Worker, b Broadcaster) {
	w.RegisterHandler(EventEmailUpdated, newEmailUpdatedEventHandler(b))
	w.RegisterHandler(EventUsernameUpdated, newUsernameUpdatedEventHandler(b))
	w.RegisterHandler(EventPasswordUpdated, newPasswordUpdatedEventHandler(b))
	w.RegisterHandler(EventPreferredPresenceUpdated, newPreferredPresenceUpdatedEventHandler(b))
	w.RegisterHandler(EventProfileUpdated, newProfileUpdatedEventHandler(b))
	w.RegisterHandler(EventDisabled, newDisabledEventHandler(b))
}

type EventEmailUpdatedPayload struct {
	UserID    uuid.UUID `json:"userId"`
	Email     string    `json:"email"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func newEmailUpdatedEventHandler(b Broadcaster) outbox.Handler {
	return outbox.BindHandler(func(ctx context.Context, m outbox.Metadata, p EventEmailUpdatedPayload) error {
		return b.BroadcastToUser(ctx, m.ActorID, EventEmailUpdated, p)
	})
}

type EventUsernameUpdatedPayload struct {
	UserID    uuid.UUID `json:"userId"`
	Username  string    `json:"username"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func newUsernameUpdatedEventHandler(b Broadcaster) outbox.Handler {
	return outbox.BindHandler(func(ctx context.Context, m outbox.Metadata, p EventUsernameUpdatedPayload) error {
		return b.BroadcastToUsers(ctx, m.RecipientIDs, EventUsernameUpdated, p)
	})
}

type EventPasswordUpdatedPayload struct {
	UserID    uuid.UUID `json:"userId"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func newPasswordUpdatedEventHandler(b Broadcaster) outbox.Handler {
	return outbox.BindHandler(func(ctx context.Context, m outbox.Metadata, p EventPasswordUpdatedPayload) error {
		return b.BroadcastToUser(ctx, m.ActorID, EventPasswordUpdated, p)
	})
}

type EventPreferredPresenceUpdatedPayload struct {
	UserID    uuid.UUID         `json:"userId"`
	Presence  presence.Presence `json:"presence"`
	UpdatedAt time.Time         `json:"updatedAt"`
}

func newPreferredPresenceUpdatedEventHandler(b Broadcaster) outbox.Handler {
	return outbox.BindHandler(func(ctx context.Context, m outbox.Metadata, p EventPreferredPresenceUpdatedPayload) error {
		return b.BroadcastToUsers(ctx, m.RecipientIDs, EventPreferredPresenceUpdated, p)
	})
}

type EventProfileUpdatedPayload struct {
	UserID      uuid.UUID `json:"userId"`
	DisplayName string    `json:"displayName"`
	Bio         *string   `json:"bio,omitempty"`
	AvatarURL   *string   `json:"avatarUrl,omitempty"`
	BannerColor *string   `json:"bannerColor,omitempty"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

func newProfileUpdatedEventHandler(b Broadcaster) outbox.Handler {
	return outbox.BindHandler(func(ctx context.Context, m outbox.Metadata, p EventProfileUpdatedPayload) error {
		return b.BroadcastToUsers(ctx, m.RecipientIDs, EventProfileUpdated, p)
	})
}

type EventDisabledPayload struct {
	UserID    uuid.UUID `json:"userId"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func newDisabledEventHandler(b Broadcaster) outbox.Handler {
	return outbox.BindHandler(func(ctx context.Context, m outbox.Metadata, p EventDisabledPayload) error {
		return b.BroadcastToUsers(ctx, m.RecipientIDs, EventDisabled, p)
	})
}
