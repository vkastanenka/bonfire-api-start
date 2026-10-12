package repository

import "bonfire-api/internal/pkg/errs"

const (
	ErrReasonJSONMarshalFailed     = "JSON_MARSHAL_FAILED"
	ErrReasonDomainPayloadRequired = "DOMAIN_PAYLOAD_REQUIRED"
	ErrReasonOutboxEventNotFound   = "OUTBOX_EVENT_NOT_FOUND"
	ErrReasonOutboxEventLeaseLost  = "OUTBOX_EVENT_LEASE_LOST"
)

func ErrJSONMarshalFailed() *errs.Error {
	return errs.Internal("Failed to marshal json payload.").
		Reason(ErrReasonJSONMarshalFailed)
}

func ErrDomainPayloadRequired() *errs.Error {
	return errs.Internal("Domain payload is required.").
		Reason(ErrReasonDomainPayloadRequired)
}

func ErrOutboxEventNotFound() *errs.Error {
	return errs.NotFound("Outbox event not found; worker lease was lost or event already processed.").
		Reason(ErrReasonOutboxEventNotFound)
}

func ErrOutboxEventLeaseLost() *errs.Error {
	return errs.NotFound("Outbox event lease renewal failed; lock lost or event reclaimed.").
		Reason(ErrReasonOutboxEventLeaseLost)
}
