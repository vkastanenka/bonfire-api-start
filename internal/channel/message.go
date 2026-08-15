package channel

import (
	"bonfire-api/internal/fields"
	"encoding/json"
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

type Message struct {
	id                 fields.ID
	channelID          fields.ID
	authorID           fields.ID
	msgType            MessageType
	content            MessageContent
	systemMetadata     fields.JSON
	replyToMessageID   fields.ID
	forwardedMessageID fields.ID
	forwardedChannelID fields.ID
	pinnedAt           fields.Timestamp
	createdAt          fields.Timestamp
	updatedAt          fields.Timestamp
	editedAt           fields.Timestamp
}

// -----------------------------------------------------------------------------
// Getters
// -----------------------------------------------------------------------------

func (m *Message) ID() fields.ID                 { return m.id }
func (m *Message) ChannelID() fields.ID          { return m.channelID }
func (m *Message) AuthorID() fields.ID           { return m.authorID }
func (m *Message) Type() MessageType             { return m.msgType }
func (m *Message) Content() MessageContent       { return m.content }
func (m *Message) SystemMetadata() fields.JSON   { return m.systemMetadata }
func (m *Message) ReplyToMessageID() fields.ID   { return m.replyToMessageID }
func (m *Message) ForwardedMessageID() fields.ID { return m.forwardedMessageID }
func (m *Message) ForwardedChannelID() fields.ID { return m.forwardedChannelID }
func (m *Message) PinnedAt() fields.Timestamp    { return m.pinnedAt }
func (m *Message) CreatedAt() fields.Timestamp   { return m.createdAt }
func (m *Message) UpdatedAt() fields.Timestamp   { return m.updatedAt }
func (m *Message) EditedAt() fields.Timestamp    { return m.editedAt }

func (m *Message) CreatedAtPtr() *time.Time {
	if m == nil {
		return nil
	}
	t := m.CreatedAt()
	return &t
}

func (m *Message) IDPtr() *uuid.UUID {
	if m == nil {
		return nil
	}
	u := m.ID().UUID()
	return &u
}

// -----------------------------------------------------------------------------
// Constructors / Factory Methods
// -----------------------------------------------------------------------------

// NewMessage creates a fresh Message domain entity using UUIDv7.
func NewMessage(
	rawChannelID uuid.UUID,
	rawAuthorID *uuid.UUID,
	rawReplyToID *uuid.UUID,
	rawContent *string,
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

	content, err := NewContentPtr(rawContent)
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
		pinnedAt:         nil,
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
	rawPinnedAt *time.Time,
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

	var pinnedAt *time.Time
	if rawPinnedAt != nil {
		t := rawPinnedAt.UTC()
		pinnedAt = &t
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
		pinnedAt:         pinnedAt,
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

// SetPinned sets or unsets the pinned status and timestamp of the message.
func (m *Message) SetPinned(pinned bool) {
	if (m.pinnedAt != nil) == pinned {
		return
	}

	now := time.Now().UTC()
	if pinned {
		m.pinnedAt = &now
	} else {
		m.pinnedAt = nil
	}
	m.touchWith(now)
}

// TogglePinned flips the message's pinned status.
func (m *Message) TogglePinned() {
	m.SetPinned(m.pinnedAt == nil)
}

func (m *Message) touch() {
	m.updatedAt = time.Now().UTC()
}

func (m *Message) touchWith(t time.Time) {
	m.updatedAt = t
}

// -----------------------------------------------------------------------------
// Aggregate & Author Summary
// -----------------------------------------------------------------------------

var (
	ErrMessageAggregateNilMessage = errors.New("message aggregate requires a base message")
)

// AuthorSummary represents non-authoritative, snapshot metadata of the message author.
type AuthorSummary struct {
	id          *UserID
	username    string
	displayName string
	avatarURL   *string
}

func (a AuthorSummary) ID() *UserID         { return a.id }
func (a AuthorSummary) Username() string    { return a.username }
func (a AuthorSummary) DisplayName() string { return a.displayName }
func (a AuthorSummary) AvatarURL() *string  { return a.avatarURL }

func ReconstituteAuthorSummary(
	rawID *uuid.UUID,
	username, displayName string,
	avatarURL *string,
) (AuthorSummary, error) {
	authorID, err := NewUserIDPtr(rawID)
	if err != nil {
		return AuthorSummary{}, err
	}

	return AuthorSummary{
		id:          authorID,
		username:    username,
		displayName: displayName,
		avatarURL:   avatarURL,
	}, nil
}

// -----------------------------------------------------------------------------
// Read Model / Aggregate
// -----------------------------------------------------------------------------

// MessageAggregate is a rich read-model used for feed rendering.
type MessageAggregate struct {
	message     *Message
	author      AuthorSummary
	attachments []*Attachment
	reactions   []ReactionSummary
}

// -----------------------------------------------------------------------------
// Getters
// -----------------------------------------------------------------------------

func (ma *MessageAggregate) Message() *Message            { return ma.message }
func (ma *MessageAggregate) Author() AuthorSummary        { return ma.author }
func (ma *MessageAggregate) Attachments() []*Attachment   { return ma.attachments }
func (ma *MessageAggregate) Reactions() []ReactionSummary { return ma.reactions }

// -----------------------------------------------------------------------------
// Persistence Reconstitution
// -----------------------------------------------------------------------------

// ReconstituteMessageAggregate constructs the domain read model from persistence data.
func ReconstituteMessageAggregate(
	message *Message,
	author AuthorSummary,
	attachments []*Attachment,
	reactions []ReactionSummary,
) (*MessageAggregate, error) {
	if message == nil {
		return nil, ErrMessageAggregateNilMessage
	}

	if attachments == nil {
		attachments = make([]*Attachment, 0)
	}
	if reactions == nil {
		reactions = make([]ReactionSummary, 0)
	}

	return &MessageAggregate{
		message:     message,
		author:      author,
		attachments: attachments,
		reactions:   reactions,
	}, nil
}

// -----------------------------------------------------------------------------
// Unmarshaling Helpers for JSON Aggregates from SQL
// -----------------------------------------------------------------------------

// AttachmentDTO represents the raw JSON structure returned by sqlc json_agg.
type AttachmentDTO struct {
	ID          uuid.UUID `json:"id"`
	FileName    string    `json:"file_name"`
	FileSize    int32     `json:"file_size"`
	ContentType string    `json:"content_type"`
	URL         string    `json:"url"`
	Width       *int32    `json:"width"`
	Height      *int32    `json:"height"`
	CreatedAt   time.Time `json:"created_at"`
}

// ReactionDTO represents the raw JSON structure returned by sqlc json_agg.
type ReactionDTO struct {
	MessageID uuid.UUID `json:"message_id"`
	UserID    uuid.UUID `json:"user_id"`
	Emoji     string    `json:"emoji"`
	CreatedAt time.Time `json:"created_at"`
}

// UnmarshalAttachmentsJSON parses the raw sqlc JSON byte slice into reconstituted domain Attachments.
func UnmarshalAttachmentsJSON(messageID uuid.UUID, rawJSON []byte) ([]*Attachment, error) {
	if len(rawJSON) == 0 || string(rawJSON) == "[]" {
		return []*Attachment{}, nil
	}

	var dtos []AttachmentDTO
	if err := json.Unmarshal(rawJSON, &dtos); err != nil {
		return nil, err
	}

	attachments := make([]*Attachment, 0, len(dtos))
	for _, dto := range dtos {
		att, err := ReconstituteAttachment(
			dto.ID,
			messageID,
			dto.FileName,
			dto.FileSize,
			dto.ContentType,
			dto.URL,
			dto.Width,
			dto.Height,
			dto.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		attachments = append(attachments, att)
	}

	return attachments, nil
}

// UnmarshalReactionsJSON parses and aggregates raw sqlc reaction JSON byte slices into ReactionSummaries.
func UnmarshalReactionsJSON(rawJSON []byte, currentUserID *uuid.UUID) ([]ReactionSummary, error) {
	if len(rawJSON) == 0 || string(rawJSON) == "[]" {
		return []ReactionSummary{}, nil
	}

	var dtos []ReactionDTO
	if err := json.Unmarshal(rawJSON, &dtos); err != nil {
		return nil, err
	}

	// Group reactions by Emoji to build ReactionSummaries
	type summaryGroup struct {
		count      int64
		hasReacted bool
	}
	grouped := make(map[string]*summaryGroup)

	for _, dto := range dtos {
		grp, exists := grouped[dto.Emoji]
		if !exists {
			grp = &summaryGroup{}
			grouped[dto.Emoji] = grp
		}
		grp.count++
		if currentUserID != nil && dto.UserID == *currentUserID {
			grp.hasReacted = true
		}
	}

	summaries := make([]ReactionSummary, 0, len(grouped))
	for emoji, grp := range grouped {
		summary, err := ReconstituteReactionSummary(emoji, grp.count, grp.hasReacted)
		if err != nil {
			return nil, err
		}
		summaries = append(summaries, summary)
	}

	return summaries, nil
}
