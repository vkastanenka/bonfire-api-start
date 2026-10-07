package token

import "bonfire-api/internal/pkg/parse"

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

var typeNamesSlice = typeNames[:]

func ParseTypeInt(i int) (Type, error) {
	return parse.EnumInt(i, typeMax, "token_type")
}

func ParseTypeString(s string) (Type, error) {
	return parse.EnumString[Type](s, typeNamesSlice, "token_type")
}

func ParseTypeStringBytes(b []byte) (Type, error) {
	return parse.EnumStringBytes[Type](b, typeNamesSlice, "token_type")
}

func (t Type) IsValid() bool {
	return t > TypeUnknown && t < typeMax
}

func (t Type) Int() int {
	if t.IsValid() {
		return int(t)
	}
	return int(TypeUnknown)
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
	parsed, err := ParseTypeStringBytes(text)
	if err != nil {
		return err
	}
	*t = parsed
	return nil
}
