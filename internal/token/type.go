package token

import (
	"bonfire-api/internal/fields"
)

type TypeValue int

const (
	TypeUnknown TypeValue = iota
	TypeAccess
	TypeRefresh
	TypeEmailVerify
	TypePasswordReset
	typeMax
)

var typeSpec = &fields.EnumSpec{
	Domain: "TOKEN_VARIANT",
	Max:    int(typeMax),
	Names: []string{
		"UNKNOWN",
		"ACCESS",
		"REFRESH",
		"EMAIL_VERIFY",
		"PASSWORD_RESET",
	},
	Bytes: [][]byte{
		[]byte(""),
		[]byte("access"),
		[]byte("refresh"),
		[]byte("email-verify"),
		[]byte("password-reset"),
	},
}

type Type struct {
	fields.Enum[TypeValue]
}

func NewType(val TypeValue) Type {
	return Type{Enum: fields.NewEnum(val, typeSpec)}
}

func NewTypeAccess() Type        { return NewType(TypeAccess) }
func NewTypeRefresh() Type       { return NewType(TypeRefresh) }
func NewTypeEmailVerify() Type   { return NewType(TypeEmailVerify) }
func NewTypePasswordReset() Type { return NewType(TypePasswordReset) }

func ParseType[T fields.IntegerType](raw T) (Type, error) {
	val := TypeValue(raw)
	if val <= TypeUnknown || int(val) >= typeSpec.Max {
		return Type{}, ErrTypeInvalid()
	}
	return NewType(val), nil
}

func ParseTypeString(s string) (Type, error) {
	val, ok := fields.ParseEnumString[TypeValue](s, typeSpec)
	if !ok || val <= TypeUnknown {
		return Type{}, ErrTypeInvalid()
	}
	return NewType(val), nil
}

func (v Type) IsAccess() bool        { return v.Is(TypeAccess) }
func (v Type) IsRefresh() bool       { return v.Is(TypeRefresh) }
func (v Type) IsEmailVerify() bool   { return v.Is(TypeEmailVerify) }
func (v Type) IsPasswordReset() bool { return v.Is(TypePasswordReset) }
