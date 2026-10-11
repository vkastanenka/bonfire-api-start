package repository

import "bonfire-api/internal/pkg/errs"

const (
	ErrReasonJSONMarshalFailed     = "JSON_MARSHAL_FAILED"
	ErrReasonDomainPayloadRequired = "DOMAIN_PAYLOAD_REQUIRED"
)

func ErrJSONMarshalFailed() *errs.Error {
	return errs.Internal("Failed to marshal json payload.").
		Reason(ErrReasonJSONMarshalFailed)
}

func ErrDomainPayloadRequired() *errs.Error {
	return errs.Internal("Domain payload is required.").
		Reason(ErrReasonDomainPayloadRequired)
}
