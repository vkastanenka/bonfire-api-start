package channel

import (
	"errors"
	"net/url"
	"strings"
	"unicode/utf8"

	"bonfire-api/internal/sanitize"

	"github.com/google/uuid"
)

// -----------------------------------------------------------------------------
// Strongly Typed Domain Identifier Value Objects
// -----------------------------------------------------------------------------

var (
	ErrIDNil     = errors.New("id cannot be nil or zero-value")
	ErrIDInvalid = errors.New("invalid uuid format")
)

// ID represents the primary Channel identifier (channel.ID).
type ID uuid.UUID

func NewID(raw uuid.UUID) (ID, error) {
	if raw == uuid.Nil {
		return ID{}, ErrIDNil
	}
	return ID(raw), nil
}

func ParseID(raw string) (ID, error) {
	parsed, err := uuid.Parse(strings.TrimSpace(raw))
	if err != nil {
		return ID{}, ErrIDInvalid
	}
	return NewID(parsed)
}

func (id ID) UUID() uuid.UUID      { return uuid.UUID(id) }
func (id ID) String() string       { return uuid.UUID(id).String() }
func (id ID) IsValid() bool        { return uuid.UUID(id) != uuid.Nil }
func (id ID) Equals(other ID) bool { return id == other }

// NewIDs validates a slice of raw UUIDs for batch operations.
func NewIDs(raws []uuid.UUID) ([]ID, error) {
	if len(raws) == 0 {
		return nil, nil
	}
	ids := make([]ID, 0, len(raws))
	for _, raw := range raws {
		id, err := NewID(raw)
		if err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}

// -----------------------------------------------------------------------------
// UserID (External Domain Reference)
// -----------------------------------------------------------------------------

type UserID uuid.UUID

func NewUserID(raw uuid.UUID) (UserID, error) {
	if raw == uuid.Nil {
		return UserID{}, ErrIDNil
	}
	return UserID(raw), nil
}

func NewUserIDPtr(raw *uuid.UUID) (*UserID, error) {
	if raw == nil || *raw == uuid.Nil {
		return nil, nil
	}
	id, err := NewUserID(*raw)
	if err != nil {
		return nil, err
	}
	return &id, nil
}

func ParseUserID(raw string) (UserID, error) {
	parsed, err := uuid.Parse(strings.TrimSpace(raw))
	if err != nil {
		return UserID{}, ErrIDInvalid
	}
	return NewUserID(parsed)
}

func (id UserID) UUID() uuid.UUID          { return uuid.UUID(id) }
func (id UserID) String() string           { return uuid.UUID(id).String() }
func (id UserID) IsValid() bool            { return uuid.UUID(id) != uuid.Nil }
func (id UserID) Equals(other UserID) bool { return id == other }

// -----------------------------------------------------------------------------
// MessageID (External Domain Reference)
// -----------------------------------------------------------------------------

type MessageID uuid.UUID

func NewMessageID(raw uuid.UUID) (MessageID, error) {
	if raw == uuid.Nil {
		return MessageID{}, ErrIDNil
	}
	return MessageID(raw), nil
}

func NewMessageIDPtr(raw *uuid.UUID) (*MessageID, error) {
	if raw == nil || *raw == uuid.Nil {
		return nil, nil
	}
	id, err := NewMessageID(*raw)
	if err != nil {
		return nil, err
	}
	return &id, nil
}

func ParseMessageID(raw string) (MessageID, error) {
	parsed, err := uuid.Parse(strings.TrimSpace(raw))
	if err != nil {
		return MessageID{}, ErrIDInvalid
	}
	return NewMessageID(parsed)
}

func (id MessageID) UUID() uuid.UUID             { return uuid.UUID(id) }
func (id MessageID) String() string              { return uuid.UUID(id).String() }
func (id MessageID) IsValid() bool               { return uuid.UUID(id) != uuid.Nil }
func (id MessageID) Equals(other MessageID) bool { return id == other }

// -----------------------------------------------------------------------------
// Name (Optional / Pointer Value Object)
// -----------------------------------------------------------------------------

var (
	ErrNameEmpty          = errors.New("channel name cannot be empty")
	ErrNameTooLong        = errors.New("channel name cannot exceed 100 characters")
	ErrInvalidChannelName = errors.New("channel name is invalid")
)

type Name struct {
	value string
}

// NewName sanitizes input and validates character length.
// Returns nil, nil if the raw string pointer is nil or reduces to empty whitespace.
func NewName(raw *string) (*Name, error) {
	if raw == nil {
		return nil, nil
	}

	cleaned := sanitize.Text(*raw)
	if cleaned == "" {
		return nil, ErrNameEmpty
	}

	if utf8.RuneCountInString(cleaned) > 100 {
		return nil, ErrNameTooLong
	}

	return &Name{value: cleaned}, nil
}

func (n *Name) String() string {
	if n == nil {
		return ""
	}
	return n.value
}

func (n *Name) StringPtr() *string {
	if n == nil {
		return nil
	}
	s := n.value
	return &s
}

func (n *Name) Equals(other *Name) bool {
	if n == nil && other == nil {
		return true
	}
	if n == nil || other == nil {
		return false
	}
	return n.value == other.value
}

// -----------------------------------------------------------------------------
// IconURL (Optional / Pointer Value Object)
// -----------------------------------------------------------------------------

var (
	ErrIconURLEmpty    = errors.New("icon url cannot be empty")
	ErrIconURLTooShort = errors.New("icon url must be at least 3 characters")
	ErrIconURLTooLong  = errors.New("icon url cannot exceed 2048 characters")
	ErrIconURLInvalid  = errors.New("icon url must be a valid http or https URL")
)

type IconURL struct {
	value string
}

// NewIconURL validates and constructs an IconURL value object.
// Returns nil, nil if the raw string pointer is nil or empty whitespace.
func NewIconURL(raw *string) (*IconURL, error) {
	if raw == nil {
		return nil, nil
	}

	s := strings.TrimSpace(*raw)
	if s == "" {
		return nil, nil
	}

	if len(s) < 3 {
		return nil, ErrIconURLTooShort
	}
	if len(s) > 2048 {
		return nil, ErrIconURLTooLong
	}

	parsed, err := url.ParseRequestURI(s)
	if err != nil || parsed.Host == "" {
		return nil, ErrIconURLInvalid
	}

	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "http" && scheme != "https" {
		return nil, ErrIconURLInvalid
	}

	return &IconURL{value: s}, nil
}

func (i *IconURL) String() string {
	if i == nil {
		return ""
	}
	return i.value
}

func (i *IconURL) StringPtr() *string {
	if i == nil {
		return nil
	}
	s := i.value
	return &s
}

func (i *IconURL) Equals(other *IconURL) bool {
	if i == nil && other == nil {
		return true
	}
	if i == nil || other == nil {
		return false
	}
	return i.value == other.value
}

// -----------------------------------------------------------------------------
// Content (Required Value Object)
// -----------------------------------------------------------------------------

var (
	ErrContentEmpty   = errors.New("message content cannot be empty")
	ErrContentTooLong = errors.New("message content cannot exceed 4000 characters")
)

type Content struct {
	value string
}

func NewContent(raw string) (Content, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return Content{}, ErrContentEmpty
	}
	if utf8.RuneCountInString(s) > 4000 {
		return Content{}, ErrContentTooLong
	}
	return Content{value: s}, nil
}

func NewContentPtr(raw *string) (*Content, error) {
	if raw == nil {
		return nil, nil
	}

	s := strings.TrimSpace(*raw)
	if s == "" {
		return nil, nil
	}

	content, err := NewContent(s)
	if err != nil {
		return nil, err
	}
	return &content, nil
}

func (c Content) String() string { return c.value }
func (c Content) IsValid() bool  { return c.value != "" }

func (c *Content) Equals(other *Content) bool {
	if c == nil && other == nil {
		return true
	}
	if c == nil || other == nil {
		return false
	}
	return c.value == other.value
}

// -----------------------------------------------------------------------------
// Attachment Specs (FileName, ContentType, AttachmentURL)
// -----------------------------------------------------------------------------

var (
	ErrFileNameEmpty   = errors.New("file name cannot be empty")
	ErrFileNameTooLong = errors.New("file name cannot exceed 255 characters")
)

type FileName struct {
	value string
}

func NewFileName(raw string) (FileName, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return FileName{}, ErrFileNameEmpty
	}
	if utf8.RuneCountInString(s) > 255 {
		return FileName{}, ErrFileNameTooLong
	}
	return FileName{value: s}, nil
}

func (fn FileName) String() string { return fn.value }
func (fn FileName) IsValid() bool  { return fn.value != "" }

var (
	ErrContentTypeEmpty   = errors.New("content type cannot be empty")
	ErrContentTypeTooLong = errors.New("content type cannot exceed 128 characters")
)

type ContentType struct {
	value string
}

func NewContentType(raw string) (ContentType, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return ContentType{}, ErrContentTypeEmpty
	}
	if len(s) > 128 {
		return ContentType{}, ErrContentTypeTooLong
	}
	return ContentType{value: s}, nil
}

func (ct ContentType) String() string { return ct.value }
func (ct ContentType) IsValid() bool  { return ct.value != "" }

var (
	ErrAttachmentURLEmpty    = errors.New("attachment url cannot be empty")
	ErrAttachmentURLTooShort = errors.New("attachment url must be at least 3 characters")
	ErrAttachmentURLTooLong  = errors.New("attachment url cannot exceed 2048 characters")
	ErrAttachmentURLInvalid  = errors.New("attachment url must be a valid http or https URL")
)

type AttachmentURL struct {
	value string
}

func NewAttachmentURL(raw string) (AttachmentURL, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return AttachmentURL{}, ErrAttachmentURLEmpty
	}
	if len(s) < 3 {
		return AttachmentURL{}, ErrAttachmentURLTooShort
	}
	if len(s) > 2048 {
		return AttachmentURL{}, ErrAttachmentURLTooLong
	}

	parsed, err := url.ParseRequestURI(s)
	if err != nil || parsed.Host == "" {
		return AttachmentURL{}, ErrAttachmentURLInvalid
	}

	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "http" && scheme != "https" {
		return AttachmentURL{}, ErrAttachmentURLInvalid
	}

	return AttachmentURL{value: s}, nil
}

func (au AttachmentURL) String() string { return au.value }
func (au AttachmentURL) IsValid() bool  { return au.value != "" }

// -----------------------------------------------------------------------------
// Emoji (Reaction / Expression Value Object)
// -----------------------------------------------------------------------------

var (
	ErrEmojiEmpty   = errors.New("emoji cannot be empty")
	ErrEmojiTooLong = errors.New("emoji cannot exceed 32 characters")
)

type Emoji struct {
	value string
}

func NewEmoji(raw string) (Emoji, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return Emoji{}, ErrEmojiEmpty
	}
	if utf8.RuneCountInString(s) > 32 {
		return Emoji{}, ErrEmojiTooLong
	}
	return Emoji{value: s}, nil
}

func (e Emoji) String() string { return e.value }
func (e Emoji) IsValid() bool  { return e.value != "" }
