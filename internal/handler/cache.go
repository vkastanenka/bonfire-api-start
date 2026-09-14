package handler

import (
	"bonfire-api/internal/fields"
	"context"
)

type TicketCache interface {
	Print(ctx context.Context, ticketID fields.ID, userID fields.ID, sessionID fields.ID) error
	Punch(ctx context.Context, ticketID fields.ID) (fields.ID, fields.ID, error)
}
