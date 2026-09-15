package token

import (
	"bytes"
	"fmt"
	"strconv"

	"bonfire-api/internal/sanitize"
)

type Type int

const (
	TypeUnknown Type = iota
	TypeAccess
	TypeRefresh
	TypeEmailVerify
	TypePasswordReset
	typeMax
)

var typeNames = [...]string{
	TypeUnknown:       "UNKNOWN",
	TypeAccess:        "ACCESS",
	TypeRefresh:       "REFRESH",
	TypeEmailVerify:   "EMAIL_VERIFY",
	TypePasswordReset: "PASSWORD_RESET",
}

func ParseType(raw int) (Type, error) {
	t := Type(raw)
	if !t.IsValid() {
		return TypeUnknown, fmt.Errorf("invalid token type value: %d", raw)
	}
	return t, nil
}

func ParseTypeString(s string) (Type, error) {
	for i, name := range typeNames {
		if name == s {
			return Type(i), nil
		}
	}
	return TypeUnknown, fmt.Errorf("invalid token type string: %q", s)
}

func ParseTypeBytes(raw []byte) (Type, error) {
	cleaned := sanitize.Bytes(raw)
	if len(cleaned) == 0 {
		return TypeUnknown, nil
	}
	return ParseTypeString(string(cleaned))
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

func (t Type) IsAccess() bool        { return t == TypeAccess }
func (t Type) IsRefresh() bool       { return t == TypeRefresh }
func (t Type) IsEmailVerify() bool   { return t == TypeEmailVerify }
func (t Type) IsPasswordReset() bool { return t == TypePasswordReset }

func (t Type) MarshalText() ([]byte, error) {
	return []byte(t.String()), nil
}

func (t *Type) UnmarshalText(text []byte) error {
	parsed, err := ParseTypeString(string(text))
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

	parsed, err := ParseTypeBytes(data)
	if err != nil {
		return err
	}

	*t = parsed
	return nil
}
