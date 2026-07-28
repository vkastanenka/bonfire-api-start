package channel

import (
	"time"

	"github.com/google/uuid"
)

type Reaction struct {
	messageID uuid.UUID
	userID    uuid.UUID
	emoji     Emoji
	createdAt time.Time
}

func (r *Reaction) MessageID() uuid.UUID { return r.messageID }
func (r *Reaction) UserID() uuid.UUID    { return r.userID }
func (r *Reaction) Emoji() Emoji         { return r.emoji }
func (r *Reaction) CreatedAt() time.Time { return r.createdAt }

func NewReaction(messageID, userID uuid.UUID, emoji Emoji) (*Reaction, error) {
	if messageID == uuid.Nil {
		return nil, ErrReactionMessageIDRequired
	}
	if userID == uuid.Nil {
		return nil, ErrReactionUserIDRequired
	}

	return &Reaction{
		messageID: messageID,
		userID:    userID,
		emoji:     emoji,
		createdAt: time.Now().UTC(),
	}, nil
}

func ReconstituteReaction(
	messageID, userID uuid.UUID,
	emoji string,
	createdAt time.Time,
) (*Reaction, error) {
	emojiVO, err := NewEmoji(emoji)
	if err != nil {
		return nil, err
	}

	return &Reaction{
		messageID: messageID,
		userID:    userID,
		emoji:     emojiVO,
		createdAt: createdAt,
	}, nil
}

type ReactionSummary struct {
	emoji      Emoji
	count      int64
	hasReacted bool
}

func (rs ReactionSummary) Emoji() Emoji     { return rs.emoji }
func (rs ReactionSummary) Count() int64     { return rs.count }
func (rs ReactionSummary) HasReacted() bool { return rs.hasReacted }

func ReconstituteReactionSummary(emoji string, count int64, hasReacted bool) (ReactionSummary, error) {
	emojiVO, err := NewEmoji(emoji)
	if err != nil {
		return ReactionSummary{}, err
	}

	return ReactionSummary{
		emoji:      emojiVO,
		count:      count,
		hasReacted: hasReacted,
	}, nil
}
