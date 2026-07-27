package channel

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

type Message struct {
	id               uuid.UUID
	channelID        uuid.UUID
	authorID         *uuid.UUID
	replyToMessageID *uuid.UUID
	content          string
	isPinned         bool
	createdAt        time.Time
	editedAt         *time.Time
}

func (m *Message) ID() uuid.UUID                { return m.id }
func (m *Message) ChannelID() uuid.UUID         { return m.channelID }
func (m *Message) AuthorID() *uuid.UUID         { return m.authorID }
func (m *Message) ReplyToMessageID() *uuid.UUID { return m.replyToMessageID }
func (m *Message) Content() string              { return m.content }
func (m *Message) IsPinned() bool               { return m.isPinned }
func (m *Message) CreatedAt() time.Time         { return m.createdAt }
func (m *Message) EditedAt() *time.Time         { return m.editedAt }

func NewMessage(
	channelID uuid.UUID,
	authorID *uuid.UUID,
	replyToID *uuid.UUID,
	content Content,
) (*Message, error) {
	if channelID == uuid.Nil {
		return nil, errors.New("channelID is required")
	}

	now := time.Now().UTC()

	return &Message{
		id:               uuid.Must(uuid.NewV7()),
		channelID:        channelID,
		authorID:         authorID,
		replyToMessageID: replyToID,
		content:          content.String(),
		isPinned:         false,
		createdAt:        now,
	}, nil
}

func ReconstituteMessage(
	id, channelID uuid.UUID,
	authorID, replyToMessageID *uuid.UUID,
	content string,
	isPinned bool,
	createdAt time.Time,
	editedAt *time.Time,
) *Message {
	return &Message{
		id:               id,
		channelID:        channelID,
		authorID:         authorID,
		replyToMessageID: replyToMessageID,
		content:          content,
		isPinned:         isPinned,
		createdAt:        createdAt,
		editedAt:         editedAt,
	}
}

func (m *Message) EditContent(newContent Content) {
	now := time.Now().UTC()
	m.content = newContent.String()
	m.editedAt = &now
}

func (m *Message) SetPinned(pinned bool) {
	m.isPinned = pinned
}

// Reactions

type Reaction struct {
	messageID uuid.UUID
	userID    uuid.UUID
	emoji     string
	createdAt time.Time
}

func (r *Reaction) MessageID() uuid.UUID { return r.messageID }
func (r *Reaction) UserID() uuid.UUID    { return r.userID }
func (r *Reaction) Emoji() string        { return r.emoji }
func (r *Reaction) CreatedAt() time.Time { return r.createdAt }

func NewReaction(messageID, userID uuid.UUID, emoji Emoji) (*Reaction, error) {
	if messageID == uuid.Nil || userID == uuid.Nil {
		return nil, errors.New("messageID and userID are required")
	}

	return &Reaction{
		messageID: messageID,
		userID:    userID,
		emoji:     emoji.String(),
		createdAt: time.Now().UTC(),
	}, nil
}

func ReconstituteReaction(
	messageID, userID uuid.UUID,
	emoji string,
	createdAt time.Time,
) *Reaction {
	return &Reaction{
		messageID: messageID,
		userID:    userID,
		emoji:     emoji,
		createdAt: createdAt,
	}
}

type ReactionSummary struct {
	Emoji string
	Count int64
}

func ReconstituteReactionSummary(emoji string, count int64) ReactionSummary {
	return ReactionSummary{
		Emoji: emoji,
		Count: count,
	}
}
