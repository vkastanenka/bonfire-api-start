package user

import (
	"bonfire-api/internal/pkg/ptr"
	"bonfire-api/internal/presence"
	"time"

	"github.com/google/uuid"
)

type AuthView struct {
	ID           uuid.UUID          `json:"id"`
	Email        string             `json:"email"`
	Username     string             `json:"username"`
	PasswordHash string             `json:"password_hash"`
	Presence     *presence.Presence `json:"presence"`
	VerifiedAt   *time.Time         `json:"verified_at"`
	IsVerified   bool               `json:"is_verified"`
	CreatedAt    time.Time          `json:"created_at"`
	UpdatedAt    time.Time          `json:"updated_at"`
}

func ToAuthView(u User) AuthView {
	return AuthView{
		ID:           u.ID,
		Email:        u.Email,
		Username:     u.Username,
		PasswordHash: u.PasswordHash,
		Presence:     ptr.Map(u.Presence),
		IsVerified:   u.IsVerified(),
		VerifiedAt:   ptr.Map(u.VerifiedAt),
		CreatedAt:    u.CreatedAt,
		UpdatedAt:    u.UpdatedAt,
	}
}

type PublicView struct {
	ID         uuid.UUID          `json:"id"`
	Email      string             `json:"email"`
	Username   string             `json:"username"`
	Presence   *presence.Presence `json:"presence"`
	IsVerified bool               `json:"is_verified"`
	CreatedAt  time.Time          `json:"created_at"`
	UpdatedAt  time.Time          `json:"updated_at"`
}

func ToPublicView(u User) PublicView {
	return PublicView{
		ID:         u.ID,
		Email:      u.Email,
		Username:   u.Username,
		Presence:   ptr.Map(u.Presence),
		IsVerified: u.IsVerified(),
		CreatedAt:  u.CreatedAt,
		UpdatedAt:  u.UpdatedAt,
	}
}

type PublicProfileView struct {
	UserID      uuid.UUID `json:"user_id"`
	DisplayName string    `json:"display_name"`
	AvatarURL   *string   `json:"avatar_url"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func ToPublicProfileView(up UserProfile) PublicProfileView {
	return PublicProfileView{
		UserID:      up.UserID,
		DisplayName: up.DisplayName,
		AvatarURL:   ptr.Map(up.AvatarURL),
		CreatedAt:   up.CreatedAt,
		UpdatedAt:   up.UpdatedAt,
	}
}
