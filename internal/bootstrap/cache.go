package bootstrap

import (
	"bonfire-api/internal/fields"
	"bonfire-api/internal/presence"
	"context"
)

type PresenceCache interface {
	GetPresence(ctx context.Context, userID fields.ID) (presence.Presence, error)
	GetBatchPresence(ctx context.Context, userIDs []fields.ID) (map[fields.ID]presence.Presence, error)
	SetPresence(ctx context.Context, userID fields.ID, p presence.Presence) error
}
