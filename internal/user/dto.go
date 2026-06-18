package user

import (
	"time"

	"github.com/google/uuid"
)

type UserResponse struct {
	ID         uuid.UUID  `json:"id"`
	Email      string     `json:"email"`
	Username   string     `json:"username"`
	VerifiedAt *time.Time `json:"verified_at"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}
