package user

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID                uuid.UUID
	Email             Email
	Username          Username
	PasswordHash      string
	PreferredPresence Presence
	VerifiedAt        *time.Time
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

func (u User) IsVerified() bool {
	return u.VerifiedAt != nil
}

type Profile struct {
	UserID      uuid.UUID
	DisplayName ProfileDisplayName
	AvatarURL   *string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
