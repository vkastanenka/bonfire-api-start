package relationship

import (
	"bonfire-api/internal/errs"
	"bytes"
	"strings"
	"unsafe"
)

func ErrTypeInvalid() *errs.Error {
	return errs.InvalidArgument("Invalid relationship type.").
		Reason("RELATIONSHIP_TYPE_INVALID").
		FieldViolation("type", "Must be one of: pending, friends, blocked.", "INVALID_ENUM_VALUE")
}

type Type uint8

const (
	TypeUnknown Type = iota
	TypePending
	TypeFriends
	TypeBlocked
	TypeMax
)

var typeNames = [...]string{
	TypeUnknown: "unknown",
	TypePending: "pending",
	TypeFriends: "friends",
	TypeBlocked: "blocked",
}

var typeBytes = [...][]byte{
	TypeUnknown: []byte("unknown"),
	TypePending: []byte("pending"),
	TypeFriends: []byte("friends"),
	TypeBlocked: []byte("blocked"),
}

func Parse(raw string) (Type, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return TypeUnknown, ErrTypeInvalid()
	}

	switch s {
	case "pending":
		return TypePending, nil
	case "friends":
		return TypeFriends, nil
	case "blocked":
		return TypeBlocked, nil
	}

	switch strings.ToLower(s) {
	case "pending":
		return TypePending, nil
	case "friends":
		return TypeFriends, nil
	case "blocked":
		return TypeBlocked, nil
	default:
		return TypeUnknown, ErrTypeInvalid()
	}
}

func ParseBytes(b []byte) (Type, error) {
	b = bytes.TrimSpace(b)
	if len(b) == 0 {
		return TypeUnknown, ErrTypeInvalid()
	}

	s := unsafe.String(unsafe.SliceData(b), len(b))
	return Parse(s)
}

func (t Type) IsValid() bool {
	return t > TypeUnknown && t < TypeMax
}

func (t Type) String() string {
	if t.IsValid() {
		return typeNames[t]
	}
	return typeNames[TypeUnknown]
}

func (t Type) Int16() int16 {
	return int16(t)
}

func (t Type) Int16Ptr() *int16 {
	if !t.IsValid() {
		return nil
	}
	v := t.Int16()
	return &v
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

	parsed, err := ParseBytes(text)
	if err != nil {
		return err
	}
	*t = parsed
	return nil
}
