package id

import "bonfire-api/internal/pkg/errs"

const (
	ErrReasonGenerateIDFailed = "GENERATE_ID_FAILED"
)

func ErrGenerateIDFailed() *errs.Error {
	return errs.Internal("Failed to generate ID.").
		Reason(ErrReasonGenerateIDFailed)
}
