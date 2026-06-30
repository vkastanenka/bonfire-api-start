package profile

import (
	"time"

	"bonfire-api/internal/repository"

	"github.com/google/uuid"
)

// --- profile View ---

type View struct {
	UserID      uuid.UUID `json:"user_id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	DisplayName string    `json:"display_name"`
}

func NewView(row repository.Profile) View {
	return View{
		UserID:      uuid.UUID(row.UserID.Bytes),
		CreatedAt:   row.CreatedAt.Time,
		UpdatedAt:   row.UpdatedAt.Time,
		DisplayName: row.DisplayName,
	}
}
