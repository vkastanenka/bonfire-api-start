package channel

import (
	"bytes"
	"errors"
	"fmt"
)

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
