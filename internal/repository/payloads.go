package repository

import (
	"bonfire-api/internal/user"
	"time"

	"github.com/google/uuid"
)

type userUpdatePayload struct {
	ID                     uuid.UUID  `json:"id"`
	Email                  string     `json:"email"`
	Username               string     `json:"username"`
	DisplayName            string     `json:"display_name"`
	PasswordHash           string     `json:"password_hash"`
	Phone                  *string    `json:"phone"`
	Bio                    *string    `json:"bio"`
	AvatarURL              *string    `json:"avatar_url"`
	BannerColor            *string    `json:"banner_color"`
	PreferredPresence      *int16     `json:"preferred_presence"`
	PreferredPresenceUntil *time.Time `json:"preferred_presence_until"`
	VerifiedAt             *time.Time `json:"verified_at"`
	DisabledAt             *time.Time `json:"disabled_at"`
	DeleteScheduledAt      *time.Time `json:"delete_scheduled_at"`
	UpdatedAt              time.Time  `json:"updated_at"`
}

func parseUserUpdatePayload(u *user.User) *userUpdatePayload {
	if u == nil {
		return nil
	}

	return &userUpdatePayload{
		ID:                     u.ID,
		Email:                  u.Email,
		Username:               u.Username,
		DisplayName:            u.DisplayName,
		PasswordHash:           u.PasswordHash,
		Phone:                  u.Phone,
		Bio:                    u.Bio,
		AvatarURL:              u.AvatarURL,
		BannerColor:            u.BannerColor,
		PreferredPresence:      toInt16Ptr(u.PreferredPresence),
		PreferredPresenceUntil: u.PreferredPresenceUntil,
		VerifiedAt:             u.VerifiedAt,
		DisabledAt:             u.DisabledAt,
		DeleteScheduledAt:      u.DeleteScheduledAt,
		UpdatedAt:              u.UpdatedAt,
	}
}

func parseUserUpdatePayloads(users []*user.User) []*userUpdatePayload {
	payloads := make([]*userUpdatePayload, len(users))

	for i, u := range users {
		payloads[i] = parseUserUpdatePayload(u)
	}

	return payloads
}

func toInt16Ptr[T ~int | ~int16 | ~int32](val *T) *int16 {
	if val == nil {
		return nil
	}
	v := int16(*val)
	return &v
}
