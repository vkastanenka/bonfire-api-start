package parse

import "bonfire-api/internal/pkg/errs"

const (
	// Error reasons
	ErrReasonEnumValueInvalid = "ENUM_VALUE_INVALID"
	ErrReasonFieldRequired    = "FIELD_REQUIRED"
	ErrReasonIntegerInvalid   = "INTEGER_INVALID"

	// Field violation code
	ViolationCodeEnumValueInvalid = "ENUM_VALUE_INVALID"
	ViolationCodeFieldRequired    = "FIELD_REQUIRED"
	ViolationCodeFormatInvalid    = "FORMAT_INVALID"
)

// ErrValueInvalid returns an InvalidArgument error when a field value does not match supported enum bounds/values.
func ErrValueInvalid(fieldName, val string) *errs.Error {
	return errs.InvalidArgument("invalid "+fieldName).
		Reason(ErrReasonEnumValueInvalid).
		Meta("value", val).
		FieldViolation(fieldName, "Provided value is not supported or out of bounds.", ViolationCodeEnumValueInvalid)
}

// ErrFieldRequired returns an InvalidArgument error when a mandatory field is empty or missing.
func ErrFieldRequired(fieldName, val string) *errs.Error {
	return errs.InvalidArgument(fieldName+" is required").
		Reason(ErrReasonFieldRequired).
		Meta("value", val).
		FieldViolation(fieldName, "Value cannot be empty.", ViolationCodeFieldRequired)
}

// ErrIntegerInvalid returns an InvalidArgument error when a raw value cannot be parsed as an integer.
func ErrIntegerInvalid(fieldName, val string) *errs.Error {
	return errs.InvalidArgument("invalid "+fieldName).
		Reason(ErrReasonIntegerInvalid).
		Meta("value", val).
		FieldViolation(fieldName, "Provided byte value is not a valid integer.", ViolationCodeFormatInvalid)
}
