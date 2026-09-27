package gateway

import (
	"bonfire-api/internal/presence"
	"bonfire-api/internal/user"
	"context"

	"github.com/google/uuid"
)

type PresenceCache interface {
	Delete(ctx context.Context, id uuid.UUID) error
	Get(ctx context.Context, userID uuid.UUID) (presence.Presence, error)
	GetBatch(ctx context.Context, userIDs []uuid.UUID) (map[uuid.UUID]presence.Presence, error)
	Set(ctx context.Context, userID uuid.UUID, p presence.Presence) error
}

type UserCache interface {
	Delete(ctx context.Context, id uuid.UUID) error
	DeleteBatch(ctx context.Context, ids []uuid.UUID) error
	Get(ctx context.Context, id uuid.UUID) (*user.User, error)
	GetBatch(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]*user.User, []uuid.UUID, error)
	Set(ctx context.Context, usr *user.User) error
	SetBatch(ctx context.Context, users map[uuid.UUID]*user.User) error
}

type WSTicketCache interface {
	Print(ctx context.Context, ticketID uuid.UUID, userID uuid.UUID, sessionID uuid.UUID) error
	Punch(ctx context.Context, ticketID uuid.UUID) (uuid.UUID, uuid.UUID, error)
}
