package user

import "bonfire-api/internal/pkg/errs"

const (
	ErrReasonJSONMarshalFailed                = "JSON_MARSHAL_FAILED"
	ErrReasonPasswordInvalid                  = "PASSWORD_INVALID"
	ErrReasonPasswordMismatch                 = "PASSWORD_MISMATCH"
	ErrReasonPreferredPresenceDurationInvalid = "PREFERRED_PRESENCE_DURATION_INVALID"
	ErrReasonPayloadRequired                  = "USER_PAYLOAD_REQUIRED"
	ErrReasonUserDisabled                     = "USER_DISABLED"
	ErrReasonUserScheduledDeletion            = "USER_SCHEDULED_DELETION"
)

func ErrPasswordMismatch() *errs.Error {
	return errs.InvalidArgument("Passwords must match.").
		Reason("PASSWORD_MISMATCH").
		FieldViolation("currentPassword", "Passwords do not match.", "PASSWORD_MISMATCH")
}

func ErrPreferredPresenceDurationInvalid() *errs.Error {
	return errs.Internal("Preferred presence duration is invalid.").
		Reason(ErrReasonPreferredPresenceDurationInvalid)
}

func ErrPayloadRequired() *errs.Error {
	return errs.Internal("User payload required.").
		Reason(ErrReasonPayloadRequired)
}

func ErrJSONMarshalFailed() *errs.Error {
	return errs.Internal("Failed to marshal json payload.").
		Reason(ErrReasonJSONMarshalFailed)
}

func ErrUserDisabled() *errs.Error {
	return errs.FailedPrecondition("User account is disabled.").
		Reason(ErrReasonUserDisabled)
}

func ErrUserScheduledDeletion() *errs.Error {
	return errs.FailedPrecondition("User account is scheduled for deletion.").
		Reason(ErrReasonUserScheduledDeletion)
}
