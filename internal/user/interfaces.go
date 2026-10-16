package user

import (
	"bonfire-api/internal/outbox"
	"context"
	"time"

	"github.com/google/uuid"
)

type AuthCache interface {
	DeleteUserAndRevokeSessions(ctx context.Context, userID uuid.UUID, revokedSessionIDs []uuid.UUID) error
}

type Broadcaster interface {
	ToUser(ctx context.Context, recipientID uuid.UUID, messageType string, payload any, excludeSessionIDs ...uuid.UUID) error
	ToUserSession(ctx context.Context, userID uuid.UUID, sessionID uuid.UUID, messageType string, payload any) error
	ToUserSessions(ctx context.Context, userID uuid.UUID, sessionIDs []uuid.UUID, messageType string, payload any) error
	ToUsers(ctx context.Context, recipientIDs []uuid.UUID, messageType string, payload any, excludeSessionIDs ...uuid.UUID) error
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
	RevokeAllExcept(ctx context.Context, userID uuid.UUID, exceptID uuid.UUID, now time.Time) ([]uuid.UUID, error)
}

type TX interface {
	ExecTx(ctx context.Context, fn func(txCtx context.Context) error) error
}

type Worker interface {
	RegisterHandler(eventType string, handler outbox.Handler)
}
