package session

import (
	"bonfire-api/internal/outbox"
	"context"
	"time"

	"github.com/google/uuid"
)

type Cache interface {
	Delete(ctx context.Context, id uuid.UUID) error
	DeleteBatch(ctx context.Context, ids []uuid.UUID) error
	DeleteUserSessionsIndex(ctx context.Context, userID uuid.UUID) error
	RemoveUserSessionID(ctx context.Context, userID uuid.UUID, sessionID uuid.UUID) error
}

type Repository interface {
	Revoke(ctx context.Context, id uuid.UUID, userID uuid.UUID, now time.Time) error
	RevokeAll(ctx context.Context, userID uuid.UUID, now time.Time) ([]uuid.UUID, error)
}

type CachedRepository interface {
	ListValidByUserID(ctx context.Context, userID uuid.UUID, now time.Time, limit int) ([]*Session, error)
}

type OutboxRepository interface {
	Create(ctx context.Context, e *outbox.Event) error
}

type TX interface {
	ExecTx(ctx context.Context, fn func(txCtx context.Context) error) error
}
