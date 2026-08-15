package channel

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"bonfire-api/internal/errs"
	"bonfire-api/internal/fields"
	"bonfire-api/internal/pkg/ptr"
	"bonfire-api/internal/sanitize"
)

// -----------------------------------------------------------------------------
// Name
// -----------------------------------------------------------------------------

type Name struct {
	value fields.Text
}

func ParseName(raw *string) (*Name, error) {
	if raw == nil {
		return nil, nil
	}

	cleaned := sanitize.Text(ptr.From(raw))
	if cleaned == "" {
		return nil, nil
	}

	if utf8.RuneCountInString(cleaned) > 100 {
		return nil, errs.InvalidArgument("Name too long.").
			Reason("NAME_TOO_LONG").
			FieldViolation("name", "Name must be 100 characters or fewer.", "MAX_LENGTH_EXCEEDED").
			Meta("domain", "channels")
	}

	return &Name{value: fields.NewText(cleaned)}, nil
}

// -----------------------------------------------------------------------------
// Message Content
// -----------------------------------------------------------------------------

type MessageContent struct {
	value fields.Text
}

func ParseMessageContent(raw *string) (*MessageContent, error) {
	if raw == nil {
		return nil, nil
	}

	cleaned := sanitize.Text(ptr.From(raw))
	if cleaned == "" {
		return nil, nil
	}

	if utf8.RuneCountInString(cleaned) > 4000 {
		return nil, errs.InvalidArgument("Content too long.").
			Reason("CONTENT_TOO_LONG").
			FieldViolation("content", "Content must be 4000 characters or fewer.", "MAX_LENGTH_EXCEEDED").
			Meta("domain", "messages")
	}

	return &MessageContent{value: fields.NewText(cleaned)}, nil
}

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

// -----------------------------------------------------------------------------
// Type
// -----------------------------------------------------------------------------

var ErrInvalidType = errors.New("invalid channel type")

type Type int16

const (
	TypeUnknown Type = 0
	TypeDirect  Type = 1
	TypeGroup   Type = 2
	typeMax
)

var typeNames = [...]string{
	TypeUnknown: "UNKNOWN",
	TypeDirect:  "DIRECT",
	TypeGroup:   "GROUP",
}

var typeBytes = [...][]byte{
	TypeUnknown: []byte("UNKNOWN"),
	TypeDirect:  []byte("DIRECT"),
	TypeGroup:   []byte("GROUP"),
}

func ParseType(raw int16) (Type, error) {
	t := Type(raw)
	if !t.IsValid() {
		return TypeUnknown, ErrInvalidType
	}
	return t, nil
}

func ParseTypeBytes(b []byte) (Type, error) {
	b = bytes.TrimSpace(b)
	if len(b) == 0 {
		return TypeUnknown, ErrInvalidType
	}

	for i := 1; i < int(typeMax); i++ {
		if bytes.EqualFold(typeBytes[i], b) {
			return Type(i), nil
		}
	}
	return TypeUnknown, ErrInvalidType
}

func (t Type) IsValid() bool {
	return t > TypeUnknown && t < typeMax
}

func (t Type) String() string {
	if t.IsValid() {
		return typeNames[t]
	}
	return fmt.Sprintf("TYPE_%d", t)
}

func (t Type) MarshalText() ([]byte, error) {
	if t.IsValid() {
		return typeBytes[t], nil
	}
	return typeBytes[TypeUnknown], nil
}

func (t *Type) UnmarshalText(text []byte) error {
	if len(text) == 0 {
		*t = TypeUnknown
		return nil
	}
	parsed, err := ParseTypeBytes(text)
	if err != nil {
		return err
	}
	*t = parsed
	return nil
}
