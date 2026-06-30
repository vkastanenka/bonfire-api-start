package delete_request

import (
	"time"

	"bonfire-api/internal/repository"

	"github.com/google/uuid"
)

// --- delete_request constants ---

const GracePeriod = 7 * 24 * time.Hour

// --- delete_request View ---

type View struct {
	UserID      uuid.UUID `json:"user_id"`
	CreatedAt   time.Time `json:"created_at"`
	ScheduledAt time.Time `json:"scheduled_at"`
}

func NewView(row repository.DeleteRequest) View {
	return View{
		UserID:      uuid.UUID(row.UserID.Bytes),
		CreatedAt:   row.CreatedAt.Time,
		ScheduledAt: row.ScheduledAt.Time,
	}
}
