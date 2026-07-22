package token

import "fmt"

type Variant string

const (
	VariantUnknown       Variant = ""
	VariantAccess        Variant = "access"
	VariantRefresh       Variant = "refresh"
	VariantEmailVerify   Variant = "email-verify"
	VariantPasswordReset Variant = "password-reset"
)

func (v Variant) IsValid() bool {
	switch v {
	case VariantAccess, VariantRefresh, VariantEmailVerify, VariantPasswordReset:
		return true
	default:
		return false
	}
}

func (v Variant) String() string {
	return string(v)
}

func ParseVariant(raw string) (Variant, error) {
	v := Variant(raw)
	if !v.IsValid() {
		return VariantUnknown, fmt.Errorf("invalid token variant: %q", raw)
	}
	return v, nil
}
