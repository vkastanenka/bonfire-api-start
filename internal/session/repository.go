package session

import (
	"bonfire-api/internal/outbox"
	"context"
	"net/netip"
	"time"

	"github.com/google/uuid"
)

type Cache interface {
	AddUserSessionID(ctx context.Context, userID uuid.UUID, sessionID uuid.UUID) error
	Delete(ctx context.Context, id uuid.UUID) error
	DeleteBatch(ctx context.Context, ids []uuid.UUID) error
	DeleteUserSessionsIndex(ctx context.Context, userID uuid.UUID) error
	Get(ctx context.Context, id uuid.UUID) (*Session, error)
	GetBatch(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]*Session, []uuid.UUID, error)
	GetUserSessionIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error)
	RemoveUserSessionID(ctx context.Context, userID uuid.UUID, sessionID uuid.UUID) error
	Set(ctx context.Context, sess *Session) error
	SetBatch(ctx context.Context, users map[uuid.UUID]*Session) error
	SetUserSessionIDs(ctx context.Context, userID uuid.UUID, sessionIDs []uuid.UUID) error
}

type Repository interface {
	Create(ctx context.Context, s *Session) (*Session, error)
	DeleteBatchExpired(ctx context.Context, now time.Time, limitVal int) error
	Get(ctx context.Context, id uuid.UUID) (*Session, error)
	ListValidByUserID(ctx context.Context, userID uuid.UUID, now time.Time, limit int) ([]*Session, error)
	Revoke(ctx context.Context, id uuid.UUID, userID uuid.UUID, now time.Time) error
	RevokeAll(ctx context.Context, userID uuid.UUID, now time.Time) ([]uuid.UUID, error)
	RotateRefreshTokenHash(ctx context.Context, id uuid.UUID, oldHash string, newHash string, clientIP netip.Addr, userAgent string, expiresAt time.Time, now time.Time) (*Session, error)
}

type CachedRepository interface {
	Get(ctx context.Context, id uuid.UUID) (*Session, error)
	ListValidByUserID(ctx context.Context, userID uuid.UUID, now time.Time, limit int) ([]*Session, error)
}

type OutboxRepository interface {
	Create(ctx context.Context, e *outbox.Event) error
}

type TX interface {
	ExecTx(ctx context.Context, fn func(txCtx context.Context) error) error
}
