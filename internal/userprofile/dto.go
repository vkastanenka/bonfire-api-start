package userprofile

import (
	"bonfire-api/internal/repository"
	"time"

	"github.com/google/uuid"
)

// ==========================================
// HANDLERS
// ==========================================

type CountRes struct {
	Count int64 `json:"count"`
}

type CreateReq struct {
	UserID      string `json:"user_id" validate:"required,uuid"`
	DisplayName string `json:"display_name" validate:"required,min=3,max=32"`
}

type UpdateDisplayNameReq struct {
	DisplayName string `json:"display_name" validate:"required,min=3,max=32"`
}

// ==========================================
// SERVICES
// ==========================================

type CreateParams struct {
	UserID      uuid.UUID
	DisplayName string
}

type UpdateDisplayNameParams struct {
	UserID      uuid.UUID
	DisplayName string
}

// ==========================================
// VIEW
// ==========================================

type View struct {
	UserID      uuid.UUID `json:"user_id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	DisplayName string    `json:"display_name"`
}

func NewView(row repository.UserProfile) View {
	return View{
		UserID:      uuid.UUID(row.UserID.Bytes),
		CreatedAt:   row.CreatedAt.Time,
		UpdatedAt:   row.UpdatedAt.Time,
		DisplayName: row.DisplayName,
	}
}
