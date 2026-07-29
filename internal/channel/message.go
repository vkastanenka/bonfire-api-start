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
	ErrMessageChannelIDRequired = errors.New("message channel id is required")
	ErrInvalidAuthorID          = errors.New("invalid author id")
	ErrInvalidReplyToID         = errors.New("invalid reply-to message id")
)

// Message represents a user or system post within a channel.
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

// -----------------------------------------------------------------------------
// Getters
// -----------------------------------------------------------------------------

func (m *Message) ID() uuid.UUID                { return m.id }
func (m *Message) ChannelID() uuid.UUID         { return m.channelID }
func (m *Message) AuthorID() *uuid.UUID         { return m.authorID }
func (m *Message) ReplyToMessageID() *uuid.UUID { return m.replyToMessageID }
func (m *Message) Content() Content             { return m.content }
func (m *Message) IsPinned() bool               { return m.isPinned }
func (m *Message) CreatedAt() time.Time         { return m.createdAt }
func (m *Message) EditedAt() *time.Time         { return m.editedAt }

// -----------------------------------------------------------------------------
// Constructors / Factory Methods
// -----------------------------------------------------------------------------

// NewMessage creates a fresh Message domain entity using standard UUIDv7 ordering.
func NewMessage(
	channelID uuid.UUID,
	authorID *uuid.UUID,
	replyToID *uuid.UUID,
	content Content,
) (*Message, error) {
	if channelID == uuid.Nil {
		return nil, ErrMessageChannelIDRequired
	}
	if authorID != nil && *authorID == uuid.Nil {
		return nil, ErrInvalidAuthorID
	}
	if replyToID != nil && *replyToID == uuid.Nil {
		return nil, ErrInvalidReplyToID
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

// ReconstituteMessage restores an existing Message entity from persistence.
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

// -----------------------------------------------------------------------------
// Domain Mutations
// -----------------------------------------------------------------------------

// EditContent updates the message content and records the edit timestamp.
func (m *Message) EditContent(newContent Content) {
	if m.content.Equals(newContent) {
		return
	}

	now := time.Now().UTC()
	m.content = newContent
	m.editedAt = &now
}

// SetPinned updates the pinned status of the message.
func (m *Message) SetPinned(pinned bool) {
	m.isPinned = pinned
}
