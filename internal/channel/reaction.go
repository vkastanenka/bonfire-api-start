package channel

import (
	"bonfire-api/internal/fields"
)

type Reaction struct {
	messageID fields.ID
	userID    fields.ID
	emoji     ReactionEmoji
	createdAt fields.Timestamp
}

type EmojiCount struct {
	Emoji   string
	Count   int
	Reacted bool
}

type ReactionSummary struct {
	MessageID fields.ID
	Counts    []EmojiCount
}

func ReconstituteReaction(
	messageID fields.ID,
	userID fields.ID,
	emoji ReactionEmoji,
	createdAt fields.Timestamp,
) *Reaction {
	return &Reaction{
		messageID: messageID,
		userID:    userID,
		emoji:     emoji,
		createdAt: createdAt,
	}
}

func (r *Reaction) MessageID() fields.ID        { return r.messageID }
func (r *Reaction) UserID() fields.ID           { return r.userID }
func (r *Reaction) Emoji() ReactionEmoji        { return r.emoji }
func (r *Reaction) CreatedAt() fields.Timestamp { return r.createdAt }
