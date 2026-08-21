package fields

import (
	"errors"
	"fmt"

	"bonfire-api/internal/errs"
)

func ErrLimitInvalid(fieldName string) error {
	return errs.InvalidArgument("Limit must be a non-negative integer.").
		Reason("INVALID_LIMIT").
		FieldViolation(fieldName, "Must be a non-negative integer.", "INVALID_FORMAT")
}

func ErrLimitExceeded(fieldName string) error {
	msg := fmt.Sprintf("Limit cannot exceed %d.", MaxCursorLimit)
	return errs.InvalidArgument(msg).
		Reason("LIMIT_EXCEEDED").
		FieldViolation(fieldName, msg, "MAX_LIMIT_EXCEEDED")
}

func ErrHexColorInvalid(fieldName string) error {
	return errs.InvalidArgument("Invalid hex color format.").
		Reason("INVALID_HEX_COLOR").
		FieldViolation(fieldName, "Must be a 6-character hex color (e.g. #FF0000).", "INVALID_FORMAT")
}

func ErrHexColorRequired(fieldName string) error {
	return errs.InvalidArgument("Hex color is required.").
		Reason("HEX_COLOR_REQUIRED").
		FieldViolation(fieldName, "Field is required.", "REQUIRED")
}

func ErrIDRequired(fieldName string) error {
	return errs.InvalidArgument(fieldName+" is required.").
		Reason("ID_REQUIRED").
		FieldViolation(fieldName, fieldName+" is required.", "REQUIRED")
}

func ErrIDInvalid(fieldName string) error {
	return errs.InvalidArgument("Invalid ID format.").
		Reason("INVALID_ID_FORMAT").
		FieldViolation(fieldName, "Must be a valid UUID.", "INVALID_FORMAT")
}

func ErrIntInvalid(fieldName string) error {
	return errs.InvalidArgument("Invalid integer format.").
		Reason("INVALID_INT_FORMAT").
		FieldViolation(fieldName, "Must be a valid integer.", "INVALID_FORMAT")
}

func ErrIntRequired(fieldName string) error {
	return errs.InvalidArgument(fieldName+" is required.").
		Reason("INT_REQUIRED").
		FieldViolation(fieldName, "Field is required.", "REQUIRED")
}

func ErrIntJSONInvalid(fieldName string) error {
	return errs.InvalidArgument("Invalid integer payload.").
		Reason("MALFORMED_JSON").
		FieldViolation(fieldName, "Must be a valid numeric integer.", "INVALID_TYPE")
}

func ErrJSONTooLarge(fieldName string) error {
	return errs.InvalidArgument("JSON payload is too large.").
		Reason("METADATA_TOO_LARGE").
		FieldViolation(fieldName, fmt.Sprintf("Payload must be %d bytes or fewer.", MaxJSONBytes), "MAX_SIZE_EXCEEDED")
}

func ErrJSONInvalidFormat(fieldName string) error {
	return errs.InvalidArgument("Invalid JSON format.").
		Reason("INVALID_METADATA_FORMAT").
		FieldViolation(fieldName, "Payload must be a JSON object.", "INVALID_TYPE")
}

func ErrJSONMalformed(fieldName string) error {
	return errs.InvalidArgument("Invalid JSON payload.").
		Reason("MALFORMED_JSON").
		FieldViolation(fieldName, "Payload must be valid JSON.", "MALFORMED_JSON")
}

func ErrJSONRequired(fieldName string) error {
	return errs.InvalidArgument("JSON payload is required.").
		Reason("METADATA_REQUIRED").
		FieldViolation(fieldName, "Field is required.", "REQUIRED")
}

func ErrJSONMapInvalid(fieldName string) error {
	return errs.InvalidArgument("Invalid JSON structure.").
		Reason("MALFORMED_JSON").
		FieldViolation(fieldName, "Failed to process JSON map.", "INVALID_STRUCTURE")
}

func ErrJSONNestingExceeded(fieldName string) error {
	return errs.InvalidArgument("JSON nesting is too deep.").
		Reason("METADATA_NESTING_EXCEEDED").
		FieldViolation(fieldName, fmt.Sprintf("Nesting level cannot exceed %d.", MaxMetadataDepth), "MAX_DEPTH_EXCEEDED")
}

func ErrTimestampInvalid(fieldName string) error {
	return errs.InvalidArgument("Invalid timestamp format.").
		Reason("INVALID_TIMESTAMP_FORMAT").
		FieldViolation(fieldName, "Timestamp must be a valid RFC 3339 date-time format.", "INVALID_FORMAT")
}

func ErrTimestampRequired(fieldName string) error {
	return errs.InvalidArgument("Timestamp is required.").
		Reason("TIMESTAMP_REQUIRED").
		FieldViolation(fieldName, "Field is required.", "REQUIRED")
}

func ErrURLTooLong(fieldName string) error {
	return errs.InvalidArgument("URL exceeds maximum length.").
		Reason("URL_TOO_LONG").
		FieldViolation(fieldName, fmt.Sprintf("Must not exceed %d characters.", URLMaxLength), "MAX_LENGTH_EXCEEDED")
}

func ErrURLInvalid(fieldName string) error {
	return errs.InvalidArgument("Invalid URL format.").
		Reason("INVALID_URL_FORMAT").
		FieldViolation(fieldName, "URL must be a valid HTTP or HTTPS address.", "INVALID_FORMAT")
}

func ErrURLRequired(fieldName string) error {
	return errs.InvalidArgument("URL is required.").
		Reason("URL_REQUIRED").
		FieldViolation(fieldName, "Field is required.", "REQUIRED")
}

var ErrEnumNilDomain = errors.New("enum descriptor is nil")

// ErrEnumInvalidDomain returns an error when the EnumSpec descriptor is missing.
func ErrEnumInvalidDomain() error {
	return ErrEnumNilDomain
}

// ErrEnumInvalidValue returns a formatted error when the string cannot be parsed into the enum.
func ErrEnumInvalidValue(val string) error {
	return fmt.Errorf("invalid enum value: %q", val)
}
