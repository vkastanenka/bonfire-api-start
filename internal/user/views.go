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
	IsDisabled  bool      `json:"isDisabled"`
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

func ParseSummariesMap(users map[uuid.UUID]*User) map[uuid.UUID]Summary {
	if len(users) == 0 {
		return make(map[uuid.UUID]Summary)
	}

	summaries := make(map[uuid.UUID]Summary, len(users))
	for id, u := range users {
		if u == nil {
			continue
		}
		summaries[id] = ParseSummary(u)
	}

	return summaries
}

type View struct {
	ID          uuid.UUID `json:"id"`
	Username    string    `json:"username"`
	DisplayName string    `json:"displayName"`
	AvatarURL   *string   `json:"avatarUrl,omitempty"`
	Bio         *string   `json:"bio,omitempty"`
	BannerColor *string   `json:"bannerColor,omitempty"`
	IsDisabled  bool      `json:"isDisabled"`
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

func ParseViewsMap(users map[uuid.UUID]*User) map[uuid.UUID]View {
	if len(users) == 0 {
		return make(map[uuid.UUID]View)
	}

	views := make(map[uuid.UUID]View, len(users))
	for id, u := range users {
		if u == nil {
			continue
		}
		views[id] = ParseView(u)
	}

	return views
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
		PreferredPresence:      u.PreferredPresence,
		PreferredPresenceUntil: u.PreferredPresenceUntil,
		CreatedAt:              u.CreatedAt,
		UpdatedAt:              u.UpdatedAt,
	}
}
