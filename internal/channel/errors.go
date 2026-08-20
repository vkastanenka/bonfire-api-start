package channel

import "bonfire-api/internal/errs"

func ErrMembersNotFound() *errs.Error {
	return errs.NotFound("Channel members not found.")
}
