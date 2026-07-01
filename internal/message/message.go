package message

import (
	"bonfire-api/internal/repository"
	"time"

	"github.com/google/uuid"
)

type View struct {
	ID        uuid.UUID `json:"id"`
	ChannelID uuid.UUID `json:"channel_id"`
	UserID    uuid.UUID `json:"user_id"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

func NewView(row repository.Message) View {
	return View{
		ID:        row.ID.Bytes,
		ChannelID: row.ChannelID.Bytes,
		UserID:    row.UserID.Bytes,
		Content:   row.Content,
		CreatedAt: row.CreatedAt.Time,
	}
}

// Request models for API validation
type SendReq struct {
	ChannelID uuid.UUID `json:"channel_id" validate:"required"`
	Content   string    `json:"content" validate:"required"`
}
