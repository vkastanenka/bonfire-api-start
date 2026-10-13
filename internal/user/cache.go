package user

import (
	"bonfire-api/internal/presence"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

const (
	CacheTTL = 24 * time.Hour
)

func CacheKey(id uuid.UUID) string { return "{user:" + id.String() + "}" }

type CacheDTO struct {
	ID                     uuid.UUID          `json:"id"`
	Email                  string             `json:"email"`
	Username               string             `json:"username"`
	DisplayName            string             `json:"display_name"`
	Phone                  *string            `json:"phone,omitempty"`
	Bio                    *string            `json:"bio,omitempty"`
	AvatarURL              *string            `json:"avatar_url,omitempty"`
	BannerColor            *string            `json:"banner_color,omitempty"`
	PreferredPresence      *presence.Presence `json:"preferred_presence,omitempty"`
	PreferredPresenceUntil *time.Time         `json:"preferred_presence_until,omitempty"`
	VerifiedAt             *time.Time         `json:"verified_at,omitempty"`
	DisabledAt             *time.Time         `json:"disabled_at,omitempty"`
	DeleteScheduledAt      *time.Time         `json:"delete_scheduled_at,omitempty"`
	CreatedAt              time.Time          `json:"created_at"`
	UpdatedAt              time.Time          `json:"updated_at"`
}

func NewCacheDTO(u *User) User {
	return User{
		ID:                     u.ID,
		Email:                  u.Email,
		Username:               u.Username,
		DisplayName:            u.DisplayName,
		Phone:                  u.Phone,
		Bio:                    u.Bio,
		AvatarURL:              u.AvatarURL,
		BannerColor:            u.BannerColor,
		PreferredPresence:      u.PreferredPresence,
		PreferredPresenceUntil: u.PreferredPresenceUntil,
		VerifiedAt:             u.VerifiedAt,
		DisabledAt:             u.DisabledAt,
		DeleteScheduledAt:      u.DeleteScheduledAt,
		CreatedAt:              u.CreatedAt,
		UpdatedAt:              u.UpdatedAt,
	}
}

func (u User) ToDomain() *User {
	return Reconstitute(
		u.ID,
		u.Email,
		u.Username,
		"",
		u.Phone,
		u.DisplayName,
		u.Bio,
		u.AvatarURL,
		u.BannerColor,
		u.PreferredPresence,
		u.PreferredPresenceUntil,
		u.VerifiedAt,
		u.DisabledAt,
		u.DeleteScheduledAt,
		u.CreatedAt,
		u.UpdatedAt,
	)
}

func UnmarshalCacheDTO(data []byte) (*User, error) {
	var dto User
	if err := json.Unmarshal(data, &dto); err != nil {
		return nil, err
	}
	return dto.ToDomain(), nil
}

func MarshalCacheDTO(usr *User) ([]byte, error) {
	dto := NewCacheDTO(usr)
	bytes, err := json.Marshal(dto)
	if err != nil {
		return nil, err
	}
	return bytes, nil
}
