package channel

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// -----------------------------------------------------------------------------
// Domain Errors
// -----------------------------------------------------------------------------

var (
	ErrReactionMessageIDRequired = errors.New("reaction message id is required")
	ErrReactionUserIDRequired    = errors.New("reaction user id is required")
	ErrInvalidReactionCount      = errors.New("reaction count cannot be negative")
)

// Reaction represents an individual emoji reaction on a message by a user.
type Reaction struct {
	messageID uuid.UUID
	userID    uuid.UUID
	emoji     Emoji
	createdAt time.Time
}

// -----------------------------------------------------------------------------
// Getters
// -----------------------------------------------------------------------------

func (r *Reaction) MessageID() uuid.UUID { return r.messageID }
func (r *Reaction) UserID() uuid.UUID    { return r.userID }
func (r *Reaction) Emoji() Emoji         { return r.emoji }
func (r *Reaction) CreatedAt() time.Time { return r.createdAt }

// -----------------------------------------------------------------------------
// Constructors / Factory Methods
// -----------------------------------------------------------------------------

// NewReaction creates a fresh Reaction domain entity.
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

// ReconstituteReaction restores an existing Reaction entity from persistence.
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

// -----------------------------------------------------------------------------
// Read Models / Aggregations
// -----------------------------------------------------------------------------

// ReactionSummary represents an aggregated summary of reactions for UI display.
type ReactionSummary struct {
	emoji      Emoji
	count      int64
	hasReacted bool
}

func (rs ReactionSummary) Emoji() Emoji     { return rs.emoji }
func (rs ReactionSummary) Count() int64     { return rs.count }
func (rs ReactionSummary) HasReacted() bool { return rs.hasReacted }

// ReconstituteReactionSummary constructs a ReactionSummary read model.
func ReconstituteReactionSummary(emoji string, count int64, hasReacted bool) (ReactionSummary, error) {
	if count < 0 {
		return ReactionSummary{}, ErrInvalidReactionCount
	}

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
