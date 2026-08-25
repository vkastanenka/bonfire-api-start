package token

import (
	"errors"
)

func ErrTypeInvalid() error {
	return errors.New("invalid token variant")
}

var (
	ErrTokenExpired          = errors.New("token has expired")
	ErrTokenMalformed        = errors.New("token is malformed")
	ErrTokenSignatureInvalid = errors.New("token signature is invalid")
	ErrTokenInvalid          = errors.New("token is invalid")
	ErrIssuerMismatch        = errors.New("token issuer is invalid")
	ErrVariantMismatch       = errors.New("token type mismatch")
	ErrInternal              = errors.New("internal cryptographic error")
)
