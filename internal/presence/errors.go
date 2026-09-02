package presence

import "bonfire-api/internal/errs"

func ErrPresenceInvalid() *errs.Error {
	return errs.InvalidArgument("Invalid presence.").
		Reason("PRESENCE_INVALID").
		FieldViolation("presence", "Must be one of ONLINE, OFFLINE, IDLE, BUSY, DND, or INVISIBLE.", "INVALID_ENUM_VALUE").
		Meta("domain", "presence")
}
