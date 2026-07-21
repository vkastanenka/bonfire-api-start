package relationship

import (
	"fmt"
	"strconv"
)

type Type int16

const (
	TypeUnknown Type = iota
	TypePending
	TypeFriends
	TypeBlocked
	typeMax
)

var typeNames = [...]string{
	TypeUnknown: "unknown",
	TypePending: "pending",
	TypeFriends: "friends",
	TypeBlocked: "blocked",
}

var typeValues = map[string]Type{
	"pending": TypePending,
	"friends": TypeFriends,
	"blocked": TypeBlocked,
}

func (t Type) Valid() bool {
	return t > TypeUnknown && t < typeMax
}

func (t Type) String() string {
	if int(t) >= 0 && int(t) < len(typeNames) {
		return typeNames[t]
	}
	return "unknown"
}

func Parse(s string) Type {
	if t, ok := typeValues[s]; ok {
		return t
	}
	return TypeUnknown
}

func (t Type) MarshalJSON() ([]byte, error) {
	return []byte(strconv.Quote(t.String())), nil
}

func (t *Type) UnmarshalJSON(data []byte) error {
	unquoted, err := strconv.Unquote(string(data))
	if err != nil {
		return fmt.Errorf("invalid relationship type string: %w", err)
	}

	parsed := Parse(unquoted)
	if !parsed.Valid() {
		return fmt.Errorf("invalid relationship type: %q", unquoted)
	}

	*t = parsed
	return nil
}
