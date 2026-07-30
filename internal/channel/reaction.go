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
	messageID MessageID
	userID    UserID
	emoji     Emoji
	createdAt time.Time
}

// -----------------------------------------------------------------------------
// Getters
// -----------------------------------------------------------------------------

func (r *Reaction) MessageID() MessageID { return r.messageID }
func (r *Reaction) UserID() UserID       { return r.userID }
func (r *Reaction) Emoji() Emoji         { return r.emoji }
func (r *Reaction) CreatedAt() time.Time { return r.createdAt }

// -----------------------------------------------------------------------------
// Constructors / Factory Methods
// -----------------------------------------------------------------------------

// NewReaction creates a fresh Reaction domain entity.
func NewReaction(rawMessageID, rawUserID uuid.UUID, emoji Emoji) (*Reaction, error) {
	msgID, err := NewMessageID(rawMessageID)
	if err != nil {
		return nil, ErrReactionMessageIDRequired
	}

	uID, err := NewUserID(rawUserID)
	if err != nil {
		return nil, ErrReactionUserIDRequired
	}

	if !emoji.IsValid() {
		return nil, ErrEmojiEmpty
	}

	return &Reaction{
		messageID: msgID,
		userID:    uID,
		emoji:     emoji,
		createdAt: time.Now().UTC(),
	}, nil
}

// ReconstituteReaction restores an existing Reaction entity from persistence.
func ReconstituteReaction(
	rawMessageID, rawUserID uuid.UUID,
	rawEmoji string,
	createdAt time.Time,
) (*Reaction, error) {
	msgID, err := NewMessageID(rawMessageID)
	if err != nil {
		return nil, ErrReactionMessageIDRequired
	}

	uID, err := NewUserID(rawUserID)
	if err != nil {
		return nil, ErrReactionUserIDRequired
	}

	emojiVO, err := NewEmoji(rawEmoji)
	if err != nil {
		return nil, err
	}

	return &Reaction{
		messageID: msgID,
		userID:    uID,
		emoji:     emojiVO,
		createdAt: createdAt.UTC(),
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

// ReconstituteReactionSummary constructs a ReactionSummary read model from persistence/queries.
func ReconstituteReactionSummary(rawEmoji string, count int64, hasReacted bool) (ReactionSummary, error) {
	if count < 0 {
		return ReactionSummary{}, ErrInvalidReactionCount
	}

	emojiVO, err := NewEmoji(rawEmoji)
	if err != nil {
		return ReactionSummary{}, err
	}

	return ReactionSummary{
		emoji:      emojiVO,
		count:      count,
		hasReacted: hasReacted,
	}, nil
}
