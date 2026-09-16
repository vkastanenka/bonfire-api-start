package token

import (
	"bytes"
	"strconv"

	"bonfire-api/internal/pkg/sanitize"
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
	return sanitize.ParseEnumInt(raw, typeMax, "token type")
}

func ParseTypeString(s string) (Type, error) {
	return sanitize.ParseEnumString[Type](s, typeNames[:], "token type")
}

func ParseTypeBytes(b []byte) (Type, error) {
	return sanitize.ParseEnumBytes[Type](b, typeNames[:], "token type")
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
	parsed, err := ParseTypeBytes(text)
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
