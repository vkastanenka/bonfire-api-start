package user

import (
	"bonfire-api/internal/gateway"
	"bonfire-api/internal/outbox"
	"context"
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

func newEmailUpdatedPayload(id uuid.UUID, email string, updatedAt time.Time) EventEmailUpdatedPayload {
	return EventEmailUpdatedPayload{
		UserID:    id,
		Email:     email,
		UpdatedAt: updatedAt,
	}
}

func newEmailUpdatedEventHandler(b Broadcaster) outbox.Handler {
	return outbox.BindHandler(func(ctx context.Context, m outbox.Metadata, p EventEmailUpdatedPayload) error {
		return b.BroadcastToUser(ctx, *m.ActorID, EventEmailUpdated, p)
	})
}

type EventUsernameUpdatedPayload struct {
	UserID    uuid.UUID `json:"userId"`
	Username  string    `json:"username"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func newUsernameUpdatedPayload(id uuid.UUID, username string, updatedAt time.Time) EventUsernameUpdatedPayload {
	return EventUsernameUpdatedPayload{
		UserID:    id,
		Username:  username,
		UpdatedAt: updatedAt,
	}
}

func newUsernameUpdatedEventHandler(b Broadcaster) outbox.Handler {
	return outbox.BindHandler(func(ctx context.Context, m outbox.Metadata, p EventUsernameUpdatedPayload) error {
		return b.BroadcastToUsers(ctx, m.RecipientIDs, EventUsernameUpdated, p)
	})
}

func newPasswordUpdatedEventHandler(b Broadcaster) outbox.Handler {
	return outbox.BindHandler(func(ctx context.Context, m outbox.Metadata, p struct{}) error {
		return b.BroadcastToUser(ctx, *m.ActorID, EventPasswordUpdated, p, *m.SessionID)
	})
}

type EventPreferredPresenceUpdatedPayload struct {
	UserID    uuid.UUID         `json:"userId"`
	Presence  *gateway.Presence `json:"presence,omitempty"`
	UpdatedAt time.Time         `json:"updatedAt"`
}

func newPreferredPresenceUpdatedPayload(id uuid.UUID, presence *gateway.Presence, updatedAt time.Time) EventPreferredPresenceUpdatedPayload {
	return EventPreferredPresenceUpdatedPayload{
		UserID:    id,
		Presence:  presence,
		UpdatedAt: updatedAt,
	}
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

func newProfileUpdatedPayload(id uuid.UUID, displayName string, bio, avatarURL, bannerColor *string, updatedAt time.Time) EventProfileUpdatedPayload {
	return EventProfileUpdatedPayload{
		UserID:      id,
		DisplayName: displayName,
		Bio:         bio,
		AvatarURL:   avatarURL,
		BannerColor: bannerColor,
		UpdatedAt:   updatedAt,
	}
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

func newDisabledPayload(id uuid.UUID, updatedAt time.Time) EventDisabledPayload {
	return EventDisabledPayload{
		UserID:    id,
		UpdatedAt: updatedAt,
	}
}

func newDisabledEventHandler(b Broadcaster) outbox.Handler {
	return outbox.BindHandler(func(ctx context.Context, m outbox.Metadata, p EventDisabledPayload) error {
		return b.BroadcastToUsers(ctx, m.RecipientIDs, EventDisabled, p)
	})
}
