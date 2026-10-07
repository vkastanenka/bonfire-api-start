package id

import "bonfire-api/internal/pkg/errs"

const (
	ErrReasonIDGenerateFailed = "ID_GENERATE_FAILED"
)

func ErrIDGenerateFailed() *errs.Error {
	return errs.Internal("Failed to generate ID.").
		Reason(ErrReasonIDGenerateFailed)
}
