package gateway

import (
	"bonfire-api/internal/pkg/errs"
)

func ErrInvalidWSTicketLength() *errs.Error {
	return errs.Internal("Invalid websocket ticket payload length.")
}
