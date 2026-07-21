package user

import (
	"bonfire-api/internal/presence"
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

type UserProfile struct {
	UserID      uuid.UUID
	DisplayName string
	AvatarURL   *string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
