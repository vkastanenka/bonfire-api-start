package outbox

import (
	"bonfire-api/internal/pkg/errs"
	"errors"
)

var (
	ErrFatal          = errors.New("outbox: fatal event execution error")
	ErrLeaseLost      = errors.New("outbox: lease renewal failed")
	ErrInvalidPayload = errors.New("outbox: invalid event payload schema")
)

const (
	ErrReasonIDGenerateFailed  = "ID_GENERATE_FAILED"
	ErrReasonJSONMarshalFailed = "JSON_MARSHAL_FAILED"
)

func ErrIDGenerateFailed() *errs.Error {
	return errs.Internal("Failed to generate ID.").
		Reason(ErrReasonIDGenerateFailed)
}

func ErrJSONMarshalFailed() *errs.Error {
	return errs.Internal("Failed to marshal json payload.").
		Reason(ErrReasonJSONMarshalFailed)
}
