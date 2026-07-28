package channel

import (
	"errors"
	"net/url"
	"strings"

	"bonfire-api/internal/sanitize"
)

var (
	ErrNameEmpty   = errors.New("channel name cannot be empty")
	ErrNameTooLong = errors.New("channel name cannot exceed 100 characters")
)

type Name struct {
	value string
}

func NewName(raw string) (Name, error) {
	s := sanitize.Text(raw)
	if s == "" {
		return Name{}, ErrNameEmpty
	}
	if len(s) > 100 {
		return Name{}, ErrNameTooLong
	}
	return Name{value: s}, nil
}

func (n Name) String() string { return n.value }
func (n Name) IsValid() bool  { return n.value != "" }

var (
	ErrIconURLEmpty    = errors.New("icon url cannot be empty")
	ErrIconURLTooShort = errors.New("icon url must be at least 3 characters")
	ErrIconURLTooLong  = errors.New("icon url cannot exceed 2048 characters")
	ErrIconURLInvalid  = errors.New("icon url must be a valid http or https URL")
)

type IconURL struct {
	value string
}

func NewIconURL(raw string) (IconURL, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return IconURL{}, ErrIconURLEmpty
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
