package outbox

import (
	"fmt"

	"bonfire-api/internal/errs"
)

func ErrTypeRequired() *errs.Error {
	return errs.InvalidArgument("Event type is required.").
		Reason("EVENT_TYPE_REQUIRED").
		FieldViolation("event_type", "Field is required.", "REQUIRED")
}

func ErrTypeTooLong() *errs.Error {
	return errs.InvalidArgument("Event type exceeds maximum length.").
		Reason("EVENT_TYPE_TOO_LONG").
		FieldViolation("event_type", fmt.Sprintf("Must not exceed %d characters.", typeMaxLength), "MAX_LENGTH_EXCEEDED")
}

func ErrPayloadEmpty() *errs.Error {
	return errs.InvalidArgument("Payload cannot be empty.").
		Reason("PAYLOAD_EMPTY").
		FieldViolation("payload", "Payload cannot be an empty JSON object or array.", "INVALID_VALUE")
}

func ErrPayloadInvalidJSON() *errs.Error {
	return errs.InvalidArgument("Invalid JSON payload.").
		Reason("PAYLOAD_INVALID_JSON").
		FieldViolation("payload", "Must be valid JSON.", "MALFORMED_JSON")
}

func ErrPayloadRequired() *errs.Error {
	return errs.InvalidArgument("Payload is required.").
		Reason("PAYLOAD_REQUIRED").
		FieldViolation("payload", "Field is required.", "REQUIRED")
}

func ErrPayloadTooLarge() *errs.Error {
	return errs.InvalidArgument("Payload size exceeds maximum allowed size.").
		Reason("PAYLOAD_TOO_LARGE").
		FieldViolation("payload", fmt.Sprintf("Payload size must be under %d bytes.", maxPayloadByteSize), "MAX_SIZE_EXCEEDED")
}
