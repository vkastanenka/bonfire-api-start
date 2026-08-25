package relation

import (
	"bonfire-api/internal/fields"
)

type TypeValue int

const (
	TypeUnknown TypeValue = iota
	TypePending
	TypeFriends
	TypeBlocked
	typeMax
)

var typeSpec = &fields.EnumSpec{
	Domain: "RELATIONSHIP_TYPE",
	Max:    int(typeMax),
	Names:  []string{"unknown", "pending", "friends", "blocked"},
	Bytes:  [][]byte{[]byte("unknown"), []byte("pending"), []byte("friends"), []byte("blocked")},
}

type Type struct {
	fields.Enum[TypeValue]
}

func NewType(val TypeValue) Type {
	return Type{Enum: fields.NewEnum(val, typeSpec)}
}

func NewTypePending() Type {
	return NewType(TypePending)
}

func NewTypeFriends() Type {
	return NewType(TypeFriends)
}

func NewTypeBlocked() Type {
	return NewType(TypeBlocked)
}

func Parse(raw int) (Type, error) {
	val, ok := fields.ParseEnumInt[TypeValue](raw, typeSpec)
	if !ok || val <= TypeUnknown {
		return Type{}, ErrTypeInvalid()
	}
	return NewType(val), nil
}

func ParseString(raw string) (Type, error) {
	val, ok := fields.ParseEnumString[TypeValue](raw, typeSpec)
	if !ok || val <= TypeUnknown {
		return Type{}, ErrTypeInvalid()
	}
	return NewType(val), nil
}

func (t Type) IsPending() bool { return t.Is(TypePending) }
func (t Type) IsFriends() bool { return t.Is(TypeFriends) }
func (t Type) IsBlocked() bool { return t.Is(TypeBlocked) }
