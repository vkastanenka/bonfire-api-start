package relation

import (
	"bytes"
	"fmt"
	"strconv"

	"bonfire-api/internal/sanitize"
)

type Type int

const (
	TypeUnknown Type = iota
	TypePending
	TypeFriends
	TypeBlocked
	typeMax
)

var typeNames = [...]string{
	TypeUnknown: "UNKNOWN",
	TypePending: "PENDING",
	TypeFriends: "FRIENDS",
	TypeBlocked: "BLOCKED",
}

func Parse(raw int) (Type, error) {
	t := Type(raw)
	if !t.IsValid() {
		return TypeUnknown, fmt.Errorf("invalid relation type value: %d", raw)
	}
	return t, nil
}

func ParseString(s string) (Type, error) {
	for i, name := range typeNames {
		if name == s {
			return Type(i), nil
		}
	}
	return TypeUnknown, fmt.Errorf("invalid relation type string: %q", s)
}

func ParseBytes(raw []byte) (Type, error) {
	cleaned := sanitize.Bytes(raw)
	if len(cleaned) == 0 {
		return TypeUnknown, nil
	}
	return ParseString(string(cleaned))
}

func (t Type) IsValid() bool {
	return t > TypeUnknown && t < typeMax
}

func (t Type) String() string {
	if uint(t) < uint(len(typeNames)) {
		return typeNames[t]
	}
	return typeNames[TypeUnknown]
}

func (t Type) IsPending() bool { return t == TypePending }
func (t Type) IsFriends() bool { return t == TypeFriends }
func (t Type) IsBlocked() bool { return t == TypeBlocked }

func (t Type) MarshalText() ([]byte, error) {
	return []byte(t.String()), nil
}

func (t *Type) UnmarshalText(text []byte) error {
	parsed, err := ParseString(string(text))
	if err != nil {
		return err
	}
	*t = parsed
	return nil
}

func (t Type) MarshalJSON() ([]byte, error) {
	return strconv.AppendQuote(nil, t.String()), nil
}

func (t *Type) UnmarshalJSON(data []byte) error {
	if len(data) == 0 || bytes.Equal(data, []byte("null")) {
		*t = TypeUnknown
		return nil
	}

	parsed, err := ParseBytes(data)
	if err != nil {
		return err
	}

	*t = parsed
	return nil
}
