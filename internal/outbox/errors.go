package outbox

import (
	"fmt"

	"bonfire-api/internal/errs"
)

func ErrAggregateTypeRequired() *errs.Error {
	return errs.InvalidArgument("Aggregate type is required.").
		Reason("AGGREGATE_TYPE_REQUIRED").
		FieldViolation("aggregate_type", "Field is required.", "REQUIRED")
}

func ErrAggregateTypeTooLong() *errs.Error {
	return errs.InvalidArgument("Aggregate type exceeds maximum length.").
		Reason("AGGREGATE_TYPE_TOO_LONG").
		FieldViolation("aggregate_type", fmt.Sprintf("Must not exceed %d characters.", aggregateTypeMaxLength), "MAX_LENGTH_EXCEEDED")
}

func ErrEventTypeRequired() *errs.Error {
	return errs.InvalidArgument("Event type is required.").
		Reason("EVENT_TYPE_REQUIRED").
		FieldViolation("event_type", "Field is required.", "REQUIRED")
}

func ErrEventTypeTooLong() *errs.Error {
	return errs.InvalidArgument("Event type exceeds maximum length.").
		Reason("EVENT_TYPE_TOO_LONG").
		FieldViolation("event_type", fmt.Sprintf("Must not exceed %d characters.", eventTypeMaxLength), "MAX_LENGTH_EXCEEDED")
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
