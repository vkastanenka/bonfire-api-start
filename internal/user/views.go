package user

import (
	"bonfire-api/internal/pkg/ptr"
	"time"

	"github.com/google/uuid"
)

// JSON Tags on Views inside Domain: AuthView, PublicView, and PublicProfileView are Presentation Layer DTOs (Data Transfer Objects).
// They have json:"..." tags and ptr map dependencies.
// They belong in your HTTP / API transport layer, not inside the core user package.

type AuthView struct {
	ID                uuid.UUID  `json:"id"`
	Email             string     `json:"email"`
	Username          string     `json:"username"`
	PasswordHash      string     `json:"password_hash"`
	PreferredPresence Presence   `json:"presence"`
	VerifiedAt        *time.Time `json:"verified_at"`
	IsVerified        bool       `json:"is_verified"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

func ToAuthView(u User) AuthView {
	return AuthView{
		ID:                u.ID,
		Email:             u.Email,
		Username:          u.Username,
		PasswordHash:      u.PasswordHash,
		PreferredPresence: u.PreferredPresence,
		IsVerified:        u.IsVerified(),
		VerifiedAt:        ptr.Map(u.VerifiedAt),
		CreatedAt:         u.CreatedAt,
		UpdatedAt:         u.UpdatedAt,
	}
}

type PublicView struct {
	ID                uuid.UUID `json:"id"`
	Email             string    `json:"email"`
	Username          string    `json:"username"`
	PreferredPresence Presence  `json:"presence"`
	IsVerified        bool      `json:"is_verified"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

func ToPublicView(u User) PublicView {
	return PublicView{
		ID:                u.ID,
		Email:             u.Email,
		Username:          u.Username,
		PreferredPresence: u.PreferredPresence,
		IsVerified:        u.IsVerified(),
		CreatedAt:         u.CreatedAt,
		UpdatedAt:         u.UpdatedAt,
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
