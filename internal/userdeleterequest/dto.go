package userdeleterequest

import (
	"bonfire-api/internal/repository"
	"time"

	"github.com/google/uuid"
)

// ==========================================
// HANDLERS
// ==========================================

type PingRes struct {
	Status string `json:"status"`
}

type CountRes struct {
	Count int64 `json:"count"`
}

type CreateReq struct {
	UserID      string    `json:"user_id" validate:"required,uuid"`
	ScheduledAt time.Time `json:"scheduled_at" validate:"required"`
}

// ==========================================
// SERVICES
// ==========================================

type CreateParams struct {
	UserID      uuid.UUID `json:"user_id"`
	ScheduledAt time.Time `json:"scheduled_at"`
}

// ==========================================
// VIEW
// ==========================================

type View struct {
	UserID      uuid.UUID `json:"user_id"`
	CreatedAt   time.Time `json:"created_at"`
	ScheduledAt time.Time `json:"scheduled_at"`
}

func NewView(row repository.UserDeleteRequest) View {
	return View{
		UserID:      row.UserID.Bytes,
		CreatedAt:   row.CreatedAt.Time,
		ScheduledAt: row.ScheduledAt.Time,
	}
}
