package user

import (
	"bonfire-api/internal/pkg/ptr"
	"bonfire-api/internal/presence"
	"bonfire-api/internal/repository"
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID           uuid.UUID
	Email        string
	Username     string
	PasswordHash string
	Presence     *presence.Presence
	VerifiedAt   *time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (u User) IsVerified() bool {
	return u.VerifiedAt != nil
}

func FromRepository(row repository.User) User {
	u := User{
		ID:           uuid.UUID(row.ID.Bytes),
		Email:        row.Email,
		Username:     row.Username,
		PasswordHash: row.PasswordHash,
		CreatedAt:    row.CreatedAt.Time,
		UpdatedAt:    row.UpdatedAt.Time,
	}

	if row.VerifiedAt.Valid {
		u.VerifiedAt = ptr.To(row.VerifiedAt.Time)
	}

	if row.Presence.Valid {
		u.Presence = ptr.To(presence.Presence(row.Presence.Int16))
	}

	return u
}

type View struct {
	ID         uuid.UUID          `json:"id"`
	Email      string             `json:"email"`
	Username   string             `json:"username"`
	Presence   *presence.Presence `json:"presence"`
	VerifiedAt *time.Time         `json:"verified_at"`
	IsVerified bool               `json:"is_verified"`
	CreatedAt  time.Time          `json:"created_at"`
	UpdatedAt  time.Time          `json:"updated_at"`
}

func ToView(u User) View {
	return View{
		ID:         u.ID,
		Email:      u.Email,
		Username:   u.Username,
		Presence:   ptr.Map(u.Presence),
		IsVerified: u.IsVerified(),
		VerifiedAt: ptr.Map(u.VerifiedAt),
		CreatedAt:  u.CreatedAt,
		UpdatedAt:  u.UpdatedAt,
	}
}

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
		ID:         u.ID,
		Email:      u.Email,
		Username:   u.Username,
		PasswordHash: u.PasswordHash,
		Presence:   ptr.Map(u.Presence),
		IsVerified: u.IsVerified(),
		VerifiedAt: ptr.Map(u.VerifiedAt),
		CreatedAt:  u.CreatedAt,
		UpdatedAt:  u.UpdatedAt,
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

type UserProfile struct {
	UserID      uuid.UUID
	DisplayName string
	AvatarURL   *string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func ProfileFromRepository(row repository.UserProfile) UserProfile {
	up := UserProfile{
		UserID:      uuid.UUID(row.UserID.Bytes),
		DisplayName: row.DisplayName,
		CreatedAt:   row.CreatedAt.Time,
		UpdatedAt:   row.UpdatedAt.Time,
	}

	if row.AvatarUrl.Valid {
		up.AvatarURL = ptr.To(row.AvatarUrl.String)
	}

	return up
}

type ProfileView struct {
	UserID      uuid.UUID `json:"user_id"`
	DisplayName string    `json:"display_name"`
	AvatarURL   *string   `json:"avatar_url"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func ToProfileView(up UserProfile) ProfileView {
	return ProfileView{
		UserID:      up.UserID,
		DisplayName: up.DisplayName,
		AvatarURL:   ptr.Map(up.AvatarURL),
		CreatedAt:   up.CreatedAt,
		UpdatedAt:   up.UpdatedAt,
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
