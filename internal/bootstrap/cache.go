package bootstrap

import (
	"bonfire-api/internal/presence"
	"context"

	"github.com/google/uuid"
)

type PresenceCache interface {
	GetBatchNodeUsers(ctx context.Context, userIDs []uuid.UUID) (map[uuid.UUID][]uuid.UUID, error)
	GetBatchPresence(ctx context.Context, userIDs []uuid.UUID) (map[uuid.UUID]presence.Presence, error)
	GetPresence(ctx context.Context, userID uuid.UUID) (presence.Presence, error)
	GetSessionNode(ctx context.Context, userID uuid.UUID, sessionID uuid.UUID) (uuid.UUID, bool, error)
	Heartbeat(ctx context.Context, nodeID uuid.UUID, userID uuid.UUID, sessionID uuid.UUID) error
	RegisterNodeSession(ctx context.Context, nodeID uuid.UUID, userID uuid.UUID, sessionID uuid.UUID, p presence.Presence) (bool, presence.Presence, error)
	RemoveBatchNodeUsers(ctx context.Context, nodeID uuid.UUID, userIDs []uuid.UUID) error
	SetPresence(ctx context.Context, userID uuid.UUID, p presence.Presence) error
	UnregisterNodeSession(ctx context.Context, nodeID uuid.UUID, userID uuid.UUID, sessionID uuid.UUID) (bool, error)
}
