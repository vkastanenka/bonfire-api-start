package fields

import (
	"bonfire-api/internal/errs"
	"strings"
)

func ErrRequired(field, message string) *errs.Error {
	return errs.InvalidArgument("Invalid value.").
		Reason(strings.ToUpper(field)+"_REQUIRED").
		FieldViolation(field, message, "REQUIRED")
}

func ErrTooShort(field string, min int, message string) *errs.Error {
	return errs.InvalidArgument("Invalid value.").
		Reason(strings.ToUpper(field)+"_TOO_SHORT").
		FieldViolation(field, message, "MIN_LENGTH_NOT_MET")
}

func ErrTooLong(field string, max int, message string) *errs.Error {
	return errs.InvalidArgument("Invalid value.").
		Reason(strings.ToUpper(field)+"_TOO_LONG").
		FieldViolation(field, message, "MAX_LENGTH_EXCEEDED")
}

func ErrInvalidFormat(field, message string) *errs.Error {
	return errs.InvalidArgument("Invalid value.").
		Reason(strings.ToUpper(field)+"_INVALID_FORMAT").
		FieldViolation(field, message, "INVALID_FORMAT")
}
