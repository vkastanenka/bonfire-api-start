package channel

import (
	"errors"
	"net/url"
	"strings"

	"bonfire-api/internal/sanitize"
)

var (
	ErrNameEmpty          = errors.New("channel name cannot be empty")
	ErrNameTooLong        = errors.New("channel name cannot exceed 100 characters")
	ErrInvalidChannelName = errors.New("channel name is invalid")
)

type Name struct {
	value string
}

// NewName sanitizes input and validates length rules.
func NewName(raw *string) (*Name, error) {
	if raw == nil {
		return nil, nil
	}

	cleaned := sanitize.Text(*raw)
	if cleaned == "" {
		return nil, nil // "" or "   " becomes nil
	}

	if len(cleaned) > 100 {
		return nil, ErrInvalidChannelName
	}

	return &Name{value: cleaned}, nil
}

func (n *Name) String() string {
	if n == nil {
		return ""
	}
	return n.value
}

func (n *Name) IsNil() bool {
	return n == nil
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

var (
	ErrIconURLEmpty    = errors.New("icon url cannot be empty")
	ErrIconURLTooShort = errors.New("icon url must be at least 3 characters")
	ErrIconURLTooLong  = errors.New("icon url cannot exceed 2048 characters")
	ErrIconURLInvalid  = errors.New("icon url must be a valid http or https URL")
)

type IconURL struct {
	value string
}

func NewIconURL(s string) (IconURL, error) {
	if s == "" {
		return IconURL{}, nil
	}
	if len(s) < 3 {
		return IconURL{}, ErrIconURLTooShort
	}
	if len(s) > 2048 {
		return IconURL{}, ErrIconURLTooLong
	}

	parsed, err := url.ParseRequestURI(s)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return IconURL{}, ErrIconURLInvalid
	}

	return IconURL{value: s}, nil
}

func (n *IconURL) String() string {
	if n == nil {
		return ""
	}
	return n.value
}

func (n *IconURL) IsNil() bool {
	return n == nil
}

func (n *IconURL) Equals(other *IconURL) bool {
	if n == nil && other == nil {
		return true
	}
	if n == nil || other == nil {
		return false
	}
	return n.value == other.value
}

func (i IconURL) String() string { return i.value }
func (i IconURL) IsValid() bool  { return i.value != "" }

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
	if len(s) > 4000 {
		return Content{}, ErrContentTooLong
	}
	return Content{value: s}, nil
}

func (c Content) String() string { return c.value }
func (c Content) IsValid() bool  { return c.value != "" }

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
	if len(s) > 255 {
		return FileName{}, ErrFileNameTooLong
	}
	return FileName{value: s}, nil
}

func (fn FileName) String() string { return fn.value }
func (fn FileName) IsValid() bool  { return fn.value != "" }

var (
	ErrFileSizeInvalid = errors.New("file size must be greater than zero")
)

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
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return AttachmentURL{}, ErrAttachmentURLInvalid
	}

	return AttachmentURL{value: s}, nil
}

func (au AttachmentURL) String() string { return au.value }
func (au AttachmentURL) IsValid() bool  { return au.value != "" }

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
	if len(s) > 32 {
		return Emoji{}, ErrEmojiTooLong
	}
	return Emoji{value: s}, nil
}

func (e Emoji) String() string { return e.value }
func (e Emoji) IsValid() bool  { return e.value != "" }
