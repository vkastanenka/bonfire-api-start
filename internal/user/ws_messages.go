package user

import (
	"bonfire-api/internal/gateway"
	"time"

	"github.com/google/uuid"
)

const (
	MessageEmailUpdated             = "user.email_updated"
	MessageUsernameUpdated          = "user.username_updated"
	MessagePasswordUpdated          = "user.password_updated"
	MessagePreferredPresenceUpdated = "user.preferred_presence_updated"
	MessageProfileUpdated           = "user.profile_updated"
	MessageDisabled                 = "user.disabled"
)

type MessageEmailUpdatedPayload struct {
	UserID    uuid.UUID `json:"userId"`
	Email     string    `json:"email"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func newEmailUpdatedPayload(id uuid.UUID, email string, updatedAt time.Time) MessageEmailUpdatedPayload {
	return MessageEmailUpdatedPayload{
		UserID:    id,
		Email:     email,
		UpdatedAt: updatedAt,
	}
}

type MessageUsernameUpdatedPayload struct {
	UserID    uuid.UUID `json:"userId"`
	Username  string    `json:"username"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func newUsernameUpdatedPayload(id uuid.UUID, username string, updatedAt time.Time) MessageUsernameUpdatedPayload {
	return MessageUsernameUpdatedPayload{
		UserID:    id,
		Username:  username,
		UpdatedAt: updatedAt,
	}
}

type MessagePreferredPresenceUpdatedPayload struct {
	UserID    uuid.UUID         `json:"userId"`
	Presence  *gateway.Presence `json:"presence,omitempty"`
	UpdatedAt time.Time         `json:"updatedAt"`
}

func newPreferredPresenceUpdatedPayload(id uuid.UUID, presence *gateway.Presence, updatedAt time.Time) MessagePreferredPresenceUpdatedPayload {
	return MessagePreferredPresenceUpdatedPayload{
		UserID:    id,
		Presence:  presence,
		UpdatedAt: updatedAt,
	}
}

type MessageProfileUpdatedPayload struct {
	UserID      uuid.UUID `json:"userId"`
	DisplayName string    `json:"displayName"`
	Bio         *string   `json:"bio,omitempty"`
	AvatarURL   *string   `json:"avatarUrl,omitempty"`
	BannerColor *string   `json:"bannerColor,omitempty"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

func newProfileUpdatedPayload(id uuid.UUID, displayName string, bio, avatarURL, bannerColor *string, updatedAt time.Time) MessageProfileUpdatedPayload {
	return MessageProfileUpdatedPayload{
		UserID:      id,
		DisplayName: displayName,
		Bio:         bio,
		AvatarURL:   avatarURL,
		BannerColor: bannerColor,
		UpdatedAt:   updatedAt,
	}
}

type MessageDisabledPayload struct {
	UserID    uuid.UUID `json:"userId"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func newDisabledPayload(id uuid.UUID, updatedAt time.Time) MessageDisabledPayload {
	return MessageDisabledPayload{
		UserID:    id,
		UpdatedAt: updatedAt,
	}
}
