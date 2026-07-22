package relationship

import (
	"bytes"
	"errors"
	"unsafe"
)

var ErrInvalidVariant = errors.New("invalid relationship variant")

type Variant uint8

const (
	VariantUnknown Variant = iota
	VariantPending
	VariantFriends
	VariantBlocked
	variantMax
)

var variantNames = [...]string{
	VariantUnknown: "unknown",
	VariantPending: "pending",
	VariantFriends: "friends",
	VariantBlocked: "blocked",
}

var variantBytes = [...][]byte{
	VariantUnknown: []byte("unknown"),
	VariantPending: []byte("pending"),
	VariantFriends: []byte("friends"),
	VariantBlocked: []byte("blocked"),
}

func NewVariant(raw string) (Variant, error) {
	if raw == "" {
		return VariantUnknown, ErrInvalidVariant
	}
	b := unsafe.Slice(unsafe.StringData(raw), len(raw))
	return ParseBytes(b)
}

func ParseBytes(b []byte) (Variant, error) {
	b = bytes.TrimSpace(b)
	if len(b) == 0 {
		return VariantUnknown, ErrInvalidVariant
	}

	for i := 1; i < int(variantMax); i++ {
		if bytes.EqualFold(variantBytes[i], b) {
			return Variant(i), nil
		}
	}
	return VariantUnknown, ErrInvalidVariant
}

func (t Variant) IsValid() bool {
	return t > VariantUnknown && t < variantMax
}

func (t Variant) String() string {
	if t.IsValid() {
		return variantNames[t]
	}
	return variantNames[VariantUnknown]
}

func (t Variant) MarshalText() ([]byte, error) {
	if t.IsValid() {
		return variantBytes[t], nil
	}
	return variantBytes[VariantUnknown], nil
}

func (t *Variant) UnmarshalText(text []byte) error {
	if len(text) == 0 {
		*t = VariantUnknown
		return nil
	}

	parsed, err := ParseBytes(text)
	if err != nil {
		return err
	}
	*t = parsed
	return nil
}
