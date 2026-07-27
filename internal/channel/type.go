package channel

import (
	"errors"
	"fmt"
)

type Type int16

const (
	TypeUnknown Type = 0
	TypeDirect  Type = 1
	TypeGroup   Type = 2
)

var typeNames = map[Type]string{
	TypeDirect: "DIRECT",
	TypeGroup:  "GROUP",
}

func (t Type) IsValid() bool {
	return int(t) > int(TypeUnknown) && int(t) < len(typeNames)
}

func (t Type) String() string {
	if name, ok := typeNames[t]; ok {
		return name
	}
	return fmt.Sprintf("TYPE_%d", t)
}

func ParseType(raw int16) (Type, error) {
	t := Type(raw)
	if !t.IsValid() {
		return TypeUnknown, errors.New("invalid channel type")
	}
	return t, nil
}
