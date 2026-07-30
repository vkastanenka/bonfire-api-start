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
	ErrMessageContentOrMediaReq = errors.New("message must contain either content or attachments")
)

// Message represents a user or system post within a channel.
type Message struct {
	id               MessageID
	channelID        ID
	authorID         *UserID
	replyToMessageID *MessageID
	content          *Content
	isPinned         bool
	createdAt        time.Time
	updatedAt        time.Time
	editedAt         *time.Time
}

// -----------------------------------------------------------------------------
// Getters
// -----------------------------------------------------------------------------

func (m *Message) ID() MessageID                { return m.id }
func (m *Message) ChannelID() ID                { return m.channelID }
func (m *Message) AuthorID() *UserID            { return m.authorID }
func (m *Message) ReplyToMessageID() *MessageID { return m.replyToMessageID }
func (m *Message) Content() *Content            { return m.content }
func (m *Message) IsPinned() bool               { return m.isPinned }
func (m *Message) CreatedAt() time.Time         { return m.createdAt }
func (m *Message) UpdatedAt() time.Time         { return m.updatedAt }
func (m *Message) EditedAt() *time.Time         { return m.editedAt }

// -----------------------------------------------------------------------------
// Constructors / Factory Methods
// -----------------------------------------------------------------------------

// NewMessage creates a fresh Message domain entity using UUIDv7.
func NewMessage(
	rawChannelID uuid.UUID,
	rawAuthorID *uuid.UUID,
	rawReplyToID *uuid.UUID,
	content *Content,
) (*Message, error) {
	chID, err := NewID(rawChannelID)
	if err != nil {
		return nil, ErrMessageChannelIDRequired
	}

	authorID, err := NewUserIDPtr(rawAuthorID)
	if err != nil {
		return nil, err
	}

	replyToID, err := NewMessageIDPtr(rawReplyToID)
	if err != nil {
		return nil, err
	}

	rawID := uuid.Must(uuid.NewV7())
	msgID, err := NewMessageID(rawID)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()

	return &Message{
		id:               msgID,
		channelID:        chID,
		authorID:         authorID,
		replyToMessageID: replyToID,
		content:          content,
		isPinned:         false,
		createdAt:        now,
		updatedAt:        now,
		editedAt:         nil,
	}, nil
}

// ReconstituteMessage restores an existing Message entity from persistence.
func ReconstituteMessage(
	rawID, rawChannelID uuid.UUID,
	rawAuthorID, rawReplyToMessageID *uuid.UUID,
	rawContent *string,
	isPinned bool,
	createdAt, updatedAt time.Time,
	rawEditedAt *time.Time,
) (*Message, error) {
	msgID, err := NewMessageID(rawID)
	if err != nil {
		return nil, err
	}

	chID, err := NewID(rawChannelID)
	if err != nil {
		return nil, err
	}

	authorID, err := NewUserIDPtr(rawAuthorID)
	if err != nil {
		return nil, err
	}

	replyToID, err := NewMessageIDPtr(rawReplyToMessageID)
	if err != nil {
		return nil, err
	}

	content, err := NewContentPtr(rawContent)
	if err != nil {
		return nil, err
	}

	var editedAt *time.Time
	if rawEditedAt != nil {
		t := rawEditedAt.UTC()
		editedAt = &t
	}

	return &Message{
		id:               msgID,
		channelID:        chID,
		authorID:         authorID,
		replyToMessageID: replyToID,
		content:          content,
		isPinned:         isPinned,
		createdAt:        createdAt.UTC(),
		updatedAt:        updatedAt.UTC(),
		editedAt:         editedAt,
	}, nil
}

// -----------------------------------------------------------------------------
// Domain Mutations
// -----------------------------------------------------------------------------

// EditContent updates the message content, sets editedAt, and touches updatedAt.
func (m *Message) EditContent(newContent *Content) {
	if m.content.Equals(newContent) {
		return
	}

	now := time.Now().UTC()
	m.content = newContent
	m.editedAt = &now
	m.touchWith(now)
}

// SetPinned updates the pinned status of the message.
func (m *Message) SetPinned(pinned bool) {
	if m.isPinned == pinned {
		return
	}
	m.isPinned = pinned
	m.touch()
}

func (m *Message) touch() {
	m.updatedAt = time.Now().UTC()
}

func (m *Message) touchWith(t time.Time) {
	m.updatedAt = t
}
