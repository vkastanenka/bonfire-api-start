package user

import (
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
		ID:          u.ID().UUID(),
		Username:    u.Username().String(),
		DisplayName: u.DisplayName().String(),
		AvatarURL:   u.AvatarURL().StringPtr(),
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
		ID:          u.ID().UUID(),
		Username:    u.Username().String(),
		DisplayName: u.DisplayName().String(),
		AvatarURL:   u.AvatarURL().StringPtr(),
		Bio:         u.Bio().StringPtr(),
		BannerColor: u.BannerColor().StringPtr(),
		IsDisabled:  u.IsDisabled(),
		CreatedAt:   u.CreatedAt().Time(),
	}
}

type Me struct {
	ID                     uuid.UUID  `json:"id"`
	Email                  string     `json:"email"`
	Username               string     `json:"username"`
	DisplayName            string     `json:"displayName"`
	AvatarURL              *string    `json:"avatarUrl,omitempty"`
	Bio                    *string    `json:"bio,omitempty"`
	BannerColor            *string    `json:"bannerColor,omitempty"`
	IsVerified             bool       `json:"isVerified"`
	VerifiedAt             *time.Time `json:"verifiedAt,omitempty"`
	DisabledAt             *time.Time `json:"disabledAt,omitempty"`
	DeleteScheduledAt      *time.Time `json:"deleteScheduledAt,omitempty"`
	PreferredPresence      *int       `json:"preferredPresence,omitempty"`
	PreferredPresenceUntil *time.Time `json:"preferredPresenceUntil,omitempty"`
	CreatedAt              time.Time  `json:"createdAt"`
	UpdatedAt              time.Time  `json:"updatedAt"`
}

func ParseMe(u *User) Me {
	return Me{
		ID:                     u.ID().UUID(),
		Email:                  u.Email().String(),
		Username:               u.Username().String(),
		DisplayName:            u.DisplayName().String(),
		AvatarURL:              u.AvatarURL().StringPtr(),
		Bio:                    u.Bio().StringPtr(),
		BannerColor:            u.BannerColor().StringPtr(),
		IsVerified:             u.IsVerified(),
		VerifiedAt:             u.VerifiedAt().TimePtr(),
		DisabledAt:             u.DisabledAt().TimePtr(),
		DeleteScheduledAt:      u.DeleteScheduledAt().TimePtr(),
		PreferredPresence:      u.PreferredPresence().value.IntPtr(),
		PreferredPresenceUntil: u.PreferredPresenceUntil().TimePtr(),
		CreatedAt:              u.CreatedAt().Time(),
		UpdatedAt:              u.UpdatedAt().Time(),
	}
}
