package cache

import (
	"bonfire-api/internal/pkg/errs"
	"errors"
)

const (
	ErrReasonJSONMarshalFailed = "JSON_MARSHAL_FAILED"
)

func ErrJSONMarshalFailed() *errs.Error {
	return errs.Internal("Failed to marshal json payload.").
		Reason(ErrReasonJSONMarshalFailed)
}

func ErrInvalidWSTicketLength() *errs.Error {
	return errs.Internal("").
		Wrap(errors.New("invalid websocket ticket payload length"))
}

func ErrTokenAlreadyUsed() *errs.Error {
	return errs.InvalidArgument("Token already used.").
		FieldViolation("token", "Token already used.", "INVALID").
		Wrap(errors.New("valid token is required"))
}
