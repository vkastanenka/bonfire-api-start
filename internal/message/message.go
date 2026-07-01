package message

import (
	"bonfire-api/internal/repository"
	"time"

	"github.com/google/uuid"
)

type View struct {
	ID             uuid.UUID `json:"id"`
	ConversationID uuid.UUID `json:"conversation_id"`
	AuthorID       uuid.UUID `json:"author_id"`
	Content        string    `json:"content"`
	CreatedAt      time.Time `json:"created_at"`
}

func NewView(row repository.Message) View {
	return View{
		ID:             row.ID.Bytes,
		ConversationID: row.ConversationID.Bytes,
		AuthorID:       row.AuthorID.Bytes,
		Content:        row.Content,
		CreatedAt:      row.CreatedAt.Time,
	}
}
