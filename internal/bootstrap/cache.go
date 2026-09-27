package bootstrap

import (
	"bonfire-api/internal/presence"
	"context"

	"github.com/google/uuid"
)

type PresenceCache interface {
	Delete(ctx context.Context, id uuid.UUID) error
	Get(ctx context.Context, userID uuid.UUID) (presence.Presence, error)
	GetBatch(ctx context.Context, userIDs []uuid.UUID) (map[uuid.UUID]presence.Presence, error)
	Set(ctx context.Context, userID uuid.UUID, p presence.Presence) error
}
