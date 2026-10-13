package user

import (
	"bonfire-api/internal/outbox"
	"context"
	"time"

	"github.com/google/uuid"
)

type Broadcaster interface {
	BroadcastToUser(ctx context.Context, recipientID uuid.UUID, eventType string, payload any, excludeSessionIDs ...uuid.UUID) error
	BroadcastToUsers(ctx context.Context, recipientIDs []uuid.UUID, eventType string, payload any, excludeSessionIDs ...uuid.UUID) error
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

type Worker interface {
	RegisterHandler(eventType string, handler outbox.Handler)
}
