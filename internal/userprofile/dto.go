package userprofile

import (
	"bonfire-api/internal/repository"
	"time"

	"github.com/google/uuid"
)

type UserProfileView struct {
	UserID      uuid.UUID `json:"user_id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	DisplayName string    `json:"display_name"`
}

func NewUserProfileView(row repository.UserProfile) UserProfileView {
	return UserProfileView{
		UserID:      uuid.UUID(row.UserID.Bytes),
		CreatedAt:   row.CreatedAt.Time,
		UpdatedAt:   row.UpdatedAt.Time,
		DisplayName: row.DisplayName,
	}
}
