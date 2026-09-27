package cache

import (
	"bonfire-api/internal/pkg/errs"
	"errors"
)

func ErrInvalidWSTicketLength() *errs.Error {
	return errs.Internal("").
		Wrap(errors.New("invalid websocket ticket payload length"))
}

func ErrTokenAlreadyUsed() *errs.Error {
	return errs.InvalidArgument("Token already used.").
		FieldViolation("token", "Token already used.", "INVALID").
		Wrap(errors.New("valid token is required"))
}
