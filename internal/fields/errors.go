package fields

import (
	"strings"

	"bonfire-api/internal/errs"
)

func ErrRequired(field string) *errs.Error {
	return errs.InvalidArgument("Invalid value.").
		Reason(strings.ToUpper(field)+"_REQUIRED").
		FieldViolation(field, field+" cannot be empty.", "REQUIRED")
}

func ErrTooShort(field string, message string) *errs.Error {
	return errs.InvalidArgument("Invalid value.").
		Reason(strings.ToUpper(field)+"_TOO_SHORT").
		FieldViolation(field, message, "MIN_LENGTH_NOT_MET")
}

func ErrTooLong(field string, message string) *errs.Error {
	return errs.InvalidArgument("Invalid value.").
		Reason(strings.ToUpper(field)+"_TOO_LONG").
		FieldViolation(field, message, "MAX_LENGTH_EXCEEDED")
}

func ErrInvalidFormat(field, message string) *errs.Error {
	return errs.InvalidArgument("Invalid value.").
		Reason(strings.ToUpper(field)+"_INVALID_FORMAT").
		FieldViolation(field, message, "INVALID_FORMAT")
}
