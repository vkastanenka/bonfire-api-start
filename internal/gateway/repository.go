package gateway

import (
	"bonfire-api/internal/fields"
	"bonfire-api/internal/user"
	"context"
)

type UserCache interface {
	GetBatchPresence(ctx context.Context, userIDs []fields.ID) (map[fields.ID]user.Presence, error)
	GetPresence(ctx context.Context, userID fields.ID) (user.Presence, error)
	SetBatchPresence(ctx context.Context, items map[fields.ID]user.Presence) error
	SetPresence(ctx context.Context, userID fields.ID, p user.Presence) error
}

type TicketCache interface {
	Print(ctx context.Context, ticketID fields.ID, userID fields.ID, sessionID fields.ID) error
	Punch(ctx context.Context, ticketID fields.ID) (fields.ID, fields.ID, error)
}
