package user_profile

import (
	"bonfire-api/internal/repository"
	"time"

	"github.com/google/uuid"
)

type UserProfileResponse struct {
	UserID      uuid.UUID `json:"user_id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	DisplayName string    `json:"display_name"`
}

// CreateUserProfileResponse
func CreateUserProfileResponse(row repository.UserProfile) UserProfileResponse {
	return UserProfileResponse{
		UserID:      uuid.UUID(row.UserID.Bytes),
		CreatedAt:   row.CreatedAt.Time,
		UpdatedAt:   row.UpdatedAt.Time,
		DisplayName: row.DisplayName,
	}
}
