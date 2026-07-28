package channel

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrMessageChannelIDRequired  = errors.New("message channel id is required")
	ErrReactionMessageIDRequired = errors.New("reaction message id is required")
	ErrReactionUserIDRequired    = errors.New("reaction user id is required")
)

type Message struct {
	id               uuid.UUID
	channelID        uuid.UUID
	authorID         *uuid.UUID
	replyToMessageID *uuid.UUID
	content          Content
	isPinned         bool
	createdAt        time.Time
	editedAt         *time.Time
}

func (m *Message) ID() uuid.UUID                { return m.id }
func (m *Message) ChannelID() uuid.UUID         { return m.channelID }
func (m *Message) AuthorID() *uuid.UUID         { return m.authorID }
func (m *Message) ReplyToMessageID() *uuid.UUID { return m.replyToMessageID }
func (m *Message) Content() Content             { return m.content }
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
		return nil, ErrMessageChannelIDRequired
	}

	now := time.Now().UTC()

	return &Message{
		id:               uuid.Must(uuid.NewV7()),
		channelID:        channelID,
		authorID:         authorID,
		replyToMessageID: replyToID,
		content:          content,
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
) (*Message, error) {
	contentVO, err := NewContent(content)
	if err != nil {
		return nil, err
	}

	return &Message{
		id:               id,
		channelID:        channelID,
		authorID:         authorID,
		replyToMessageID: replyToMessageID,
		content:          contentVO,
		isPinned:         isPinned,
		createdAt:        createdAt,
		editedAt:         editedAt,
	}, nil
}

func (m *Message) EditContent(newContent Content) {
	now := time.Now().UTC()
	m.content = newContent
	m.editedAt = &now
}

func (m *Message) SetPinned(pinned bool) {
	m.isPinned = pinned
}
