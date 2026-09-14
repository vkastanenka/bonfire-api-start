package session

import (
	"context"
	"net/netip"
	"time"

	"github.com/google/uuid"
)

type Repository interface {
	Create(ctx context.Context, s *Session) (*Session, error)
	DeleteBatchExpired(ctx context.Context, now time.Time, limitVal int) error
	Get(ctx context.Context, id uuid.UUID) (*Session, error)
	ListValidByUserID(ctx context.Context, userID uuid.UUID, now time.Time, limit int) ([]*Session, error)
	Revoke(ctx context.Context, id uuid.UUID, userID uuid.UUID, now time.Time) error
	RevokeAll(ctx context.Context, userID uuid.UUID, now time.Time) ([]uuid.UUID, error)
	RotateRefreshTokenHash(ctx context.Context, id uuid.UUID, oldHash string, newHash string, clientIP netip.Addr, userAgent string, expiresAt time.Time, now time.Time) (*Session, error)
}

type OutboxRepository interface {
	Publish(ctx context.Context, eventType string, payload any, now time.Time) error
}

type TX interface {
	ExecTx(ctx context.Context, fn func(txCtx context.Context) error) error
}
