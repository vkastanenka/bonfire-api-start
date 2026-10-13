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
	ErrReasonIDGenerateFailed     = "ID_GENERATE_FAILED"
	ErrReasonJSONMarshalFailed    = "JSON_MARSHAL_FAILED"
	ErrReasonSessionEventRequired = "OUTBOX_EVENT_PAYLOAD_REQUIRED"
	ErrReasonEventNotFound        = "OUTBOX_EVENT_NOT_FOUND"
	ErrReasonLeaseLost            = "OUTBOX_EVENT_LEASE_LOST"
)

func ErrIDGenerateFailed() *errs.Error {
	return errs.Internal("Failed to generate ID.").
		Reason(ErrReasonIDGenerateFailed)
}

func ErrJSONMarshalFailed() *errs.Error {
	return errs.Internal("Failed to marshal json payload.").
		Reason(ErrReasonJSONMarshalFailed)
}

func ErrPayloadRequired() *errs.Error {
	return errs.Internal("Outbox event payload required.").
		Reason(ErrReasonSessionEventRequired)
}

func ErrOutboxEventNotFound() *errs.Error {
	return errs.NotFound("Outbox event not found.").
		Reason(ErrReasonEventNotFound)
}

func ErrOutboxEventLeaseLost() *errs.Error {
	return errs.Aborted("Outbox event lease lost.").
		Reason(ErrReasonLeaseLost)
}
