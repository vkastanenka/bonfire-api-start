package channel

import (
	"time"

	"github.com/google/uuid"
)

type Reaction struct {
	MessageID uuid.UUID
	UserID    uuid.UUID
	Emoji     string
	CreatedAt time.Time
}

type EmojiCount struct {
	Emoji   string
	Count   int
	Reacted bool
}

type ReactionSummary struct {
	MessageID uuid.UUID
	Counts    []EmojiCount
}

func ReconstituteReaction(
	messageID uuid.UUID,
	userID uuid.UUID,
	emoji string,
	createdAt time.Time,
) *Reaction {
	return &Reaction{
		MessageID: messageID,
		UserID:    userID,
		Emoji:     emoji,
		CreatedAt: createdAt,
	}
}
