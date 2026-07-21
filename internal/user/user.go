package user

import (
	"time"

	"github.com/google/uuid"
)

// Rich Domain Model Example

type User struct {
	ID           uuid.UUID
	Email        string
	Username     string
	PasswordHash string
	Presence     Presence
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



