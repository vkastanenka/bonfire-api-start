package user

import (
	"bonfire-api/internal/presence"
	"time"

	"github.com/google/uuid"
)

type Summary struct {
	ID          uuid.UUID `json:"id"`
	Username    string    `json:"username"`
	DisplayName string    `json:"displayName"`
	AvatarURL   *string   `json:"avatarUrl,omitempty"`
	IsDisabled  bool      `json:"isDisabled,omitempty"`
}

func ParseSummary(u *User) Summary {
	return Summary{
		ID:          u.ID,
		Username:    u.Username,
		DisplayName: u.DisplayName,
		AvatarURL:   u.AvatarURL,
		IsDisabled:  u.IsDisabled(),
	}
}

type View struct {
	ID          uuid.UUID `json:"id"`
	Username    string    `json:"username"`
	DisplayName string    `json:"displayName"`
	AvatarURL   *string   `json:"avatarUrl,omitempty"`
	Bio         *string   `json:"bio,omitempty"`
	BannerColor *string   `json:"bannerColor,omitempty"`
	IsDisabled  bool      `json:"isDisabled,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
}

func ParseView(u *User) View {
	return View{
		ID:          u.ID,
		Username:    u.Username,
		DisplayName: u.DisplayName,
		AvatarURL:   u.AvatarURL,
		Bio:         u.Bio,
		BannerColor: u.BannerColor,
		IsDisabled:  u.IsDisabled(),
		CreatedAt:   u.CreatedAt,
	}
}

type Me struct {
	ID                     uuid.UUID          `json:"id"`
	Email                  string             `json:"email"`
	Username               string             `json:"username"`
	DisplayName            string             `json:"displayName"`
	AvatarURL              *string            `json:"avatarUrl,omitempty"`
	Bio                    *string            `json:"bio,omitempty"`
	BannerColor            *string            `json:"bannerColor,omitempty"`
	IsVerified             bool               `json:"isVerified"`
	VerifiedAt             *time.Time         `json:"verifiedAt,omitempty"`
	DisabledAt             *time.Time         `json:"disabledAt,omitempty"`
	DeleteScheduledAt      *time.Time         `json:"deleteScheduledAt,omitempty"`
	PreferredPresence      *presence.Presence `json:"preferredPresence,omitempty"`
	PreferredPresenceUntil *time.Time         `json:"preferredPresenceUntil,omitempty"`
	CreatedAt              time.Time          `json:"createdAt"`
	UpdatedAt              time.Time          `json:"updatedAt"`
}

func ParseMe(u *User) Me {
	return Me{
		ID:                     u.ID,
		Email:                  u.Email,
		Username:               u.Username,
		DisplayName:            u.DisplayName,
		AvatarURL:              u.AvatarURL,
		Bio:                    u.Bio,
		BannerColor:            u.BannerColor,
		IsVerified:             u.IsVerified(),
		VerifiedAt:             u.VerifiedAt,
		DisabledAt:             u.DisabledAt,
		DeleteScheduledAt:      u.DeleteScheduledAt,
		PreferredPresence:      u.PreferredPresence,
		PreferredPresenceUntil: u.PreferredPresenceUntil,
		CreatedAt:              u.CreatedAt,
		UpdatedAt:              u.UpdatedAt,
	}
}
