package user

import (
	"bonfire-api/internal/fields"
	"time"

	"github.com/google/uuid"
)

type UserView struct {
	ID          uuid.UUID `json:"id"`
	Username    string    `json:"username"`
	DisplayName string    `json:"displayName"`
	AvatarURL   *string   `json:"avatarUrl,omitempty"`
	Bio         *string   `json:"bio,omitempty"`
	BannerColor *string   `json:"bannerColor,omitempty"`
	Presence    Presence  `json:"presence"`
	IsDisabled  bool      `json:"isDisabled,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

func ToUserView(u *User, presence Presence, now fields.Timestamp) UserView {
	p := u.EffectivePresence(now).Presence()
	if !p.IsValid() {
		p = presence
	}

	return UserView{
		ID:          u.ID().UUID(),
		Username:    u.Username().String(),
		DisplayName: u.DisplayName().String(),
		AvatarURL:   u.AvatarURL().StringPtr(),
		Bio:         u.Bio().StringPtr(),
		BannerColor: u.BannerColor().StringPtr(),
		Presence:    p,
		IsDisabled:  u.IsDisabled(),
		CreatedAt:   u.CreatedAt().Time(),
		UpdatedAt:   u.UpdatedAt().Time(),
	}
}

type UserMeView struct {
	ID                uuid.UUID  `json:"id"`
	Email             string     `json:"email"`
	Username          string     `json:"username"`
	DisplayName       string     `json:"displayName"`
	AvatarURL         *string    `json:"avatarUrl,omitempty"`
	PreferredPresence *int       `json:"preferredPresence,omitempty"`
	IsVerified        bool       `json:"isVerified"`
	VerifiedAt        *time.Time `json:"verifiedAt,omitempty"`
	CreatedAt         time.Time  `json:"createdAt"`
	UpdatedAt         time.Time  `json:"updatedAt"`
}

func ToUserMeView(u *User) UserMeView {
	return UserMeView{
		ID:                u.ID().UUID(),
		Email:             u.Email().String(),
		Username:          u.Username().String(),
		DisplayName:       u.DisplayName().String(),
		AvatarURL:         u.AvatarURL().StringPtr(),
		PreferredPresence: u.PreferredPresence().Presence().IntPtr(),
		IsVerified:        u.IsVerified(),
		VerifiedAt:        u.VerifiedAt().TimePtr(),
		CreatedAt:         u.CreatedAt().Time(),
		UpdatedAt:         u.UpdatedAt().Time(),
	}
}
