package userdeleterequest

import (
	"bonfire-api/internal/repository"
	"time"

	"github.com/google/uuid"
)

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
