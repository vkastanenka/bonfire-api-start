package user

import (
	"bonfire-api/internal/outbox"
	"context"
	"time"

	"github.com/google/uuid"
)

type Cache interface {
	Delete(ctx context.Context, id uuid.UUID) error
	DeleteBatch(ctx context.Context, ids []uuid.UUID) error
}

type CachedRepository interface {
	Get(ctx context.Context, id uuid.UUID) (*User, error)
	GetBatch(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]*User, error)
}

type CachedRelationRepository interface {
	GetFriendIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error)
}

type OutboxRepository interface {
	Create(ctx context.Context, e *outbox.Event) error
}

type SessionCache interface {
	DeleteBatch(ctx context.Context, ids []uuid.UUID) error
}

type SessionRepository interface {
	RevokeAll(ctx context.Context, userID uuid.UUID, now time.Time) ([]uuid.UUID, error)
}

type TX interface {
	ExecTx(ctx context.Context, fn func(txCtx context.Context) error) error
}
