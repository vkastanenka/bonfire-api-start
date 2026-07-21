package presence

import (
	"context"

	"github.com/google/uuid"
)

type Repository interface {
	SetPresence(ctx context.Context, userID uuid.UUID, p Presence) error
	GetPresence(ctx context.Context, userID uuid.UUID) (Presence, error)
	GetPresenceBulk(ctx context.Context, userIDs []uuid.UUID) (map[uuid.UUID]Presence, error)
}
