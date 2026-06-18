package user

import (
	"time"

	"github.com/google/uuid"
)

type DTO struct {
	ID        uuid.UUID `json:"id"`
	Email     string    `json:"email"`
	Username  string    `json:"username"`
	CreatedAt time.Time `json:"created_at"` // Assuming your row has this
}
