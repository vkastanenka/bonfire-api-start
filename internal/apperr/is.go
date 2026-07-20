package apperr

import (
	"errors"
)

func IsCode(err error, code Code) bool {
	var appErr *Error
	if errors.As(err, &appErr) {
		if appErr == nil {
			return false
		}
		return appErr.Code == code
	}
	return false
}

func IsCancelled(err error) bool          { return IsCode(err, CodeCancelled) }
func IsUnknown(err error) bool            { return IsCode(err, CodeUnknown) }
func IsInvalidArgument(err error) bool    { return IsCode(err, CodeInvalidArgument) }
func IsDeadlineExceeded(err error) bool   { return IsCode(err, CodeDeadlineExceeded) }
func IsNotFound(err error) bool           { return IsCode(err, CodeNotFound) }
func IsAlreadyExists(err error) bool      { return IsCode(err, CodeAlreadyExists) }
func IsPermissionDenied(err error) bool   { return IsCode(err, CodePermissionDenied) }
func IsResourceExhausted(err error) bool  { return IsCode(err, CodeResourceExhausted) }
func IsFailedPrecondition(err error) bool { return IsCode(err, CodeFailedPrecondition) }
func IsAborted(err error) bool            { return IsCode(err, CodeAborted) }
func IsOutOfRange(err error) bool         { return IsCode(err, CodeOutOfRange) }
func IsUnimplemented(err error) bool      { return IsCode(err, CodeUnimplemented) }
func IsInternal(err error) bool           { return IsCode(err, CodeInternal) }
func IsUnavailable(err error) bool        { return IsCode(err, CodeUnavailable) }
func IsDataLoss(err error) bool           { return IsCode(err, CodeDataLoss) }
func IsUnauthenticated(err error) bool    { return IsCode(err, CodeUnauthenticated) }
