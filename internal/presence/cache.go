package presence

import (
	"context"

	"github.com/google/uuid"
)

type UserCache interface {
	GetPeerIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error)
}

type PresenceCache interface {
	GetBatchNodeUsers(ctx context.Context, userIDs []uuid.UUID) (map[uuid.UUID][]uuid.UUID, error)
	GetBatchPresence(ctx context.Context, userIDs []uuid.UUID) (map[uuid.UUID]Presence, error)
	GetPresence(ctx context.Context, userID uuid.UUID) (Presence, error)
	Heartbeat(ctx context.Context, nodeID uuid.UUID, userID uuid.UUID, sessionID uuid.UUID) error
	RegisterNodeSession(ctx context.Context, nodeID uuid.UUID, userID uuid.UUID, sessionID uuid.UUID, p Presence) (bool, Presence, error)
	RemoveBatchNodeUsers(ctx context.Context, nodeID uuid.UUID, userIDs []uuid.UUID) error
	SetPresence(ctx context.Context, userID uuid.UUID, p Presence) error
	UnregisterNodeSession(ctx context.Context, nodeID uuid.UUID, userID uuid.UUID, sessionID uuid.UUID) (bool, error)
}

type TicketCache interface {
	Print(ctx context.Context, ticketID uuid.UUID, userID uuid.UUID, sessionID uuid.UUID) error
	Punch(ctx context.Context, ticketID uuid.UUID) (uuid.UUID, uuid.UUID, error)
}
