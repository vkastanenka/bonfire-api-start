package repository

import "bonfire-api/internal/pkg/errs"

const (
	ErrReasonJSONMarshalFailed = "JSON_MARSHAL_FAILED"
)

func ErrJSONMarshalFailed() *errs.Error {
	return errs.Internal("Failed to marshal json payload.").
		Reason(ErrReasonJSONMarshalFailed)
}
