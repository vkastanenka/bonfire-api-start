package me

import (
	"bonfire-api/internal/presence"
	"bonfire-api/internal/user"
	"time"

	"github.com/google/uuid"
)

type View struct {
	ID          uuid.UUID          `json:"id"`
	Email       string             `json:"email"`
	Username    string             `json:"username"`
	DisplayName string             `json:"display_name"`
	AvatarURL   *string            `json:"avatar_url,omitempty"`
	Presence    *presence.Presence `json:"presence,omitempty"`
	CreatedAt   time.Time          `json:"created_at"`
	UpdatedAt   time.Time          `json:"updated_at"`
}

func ToView(u user.User, p user.UserProfile) View {
	return View{
		ID:          u.ID,
		Email:       u.Email,
		Username:    u.Username,
		DisplayName: p.DisplayName,
		AvatarURL:   p.AvatarURL,
		Presence:    u.Presence,
		CreatedAt:   u.CreatedAt,
		UpdatedAt:   u.UpdatedAt,
	}
}
