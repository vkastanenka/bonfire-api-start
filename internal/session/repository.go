package session

import (
	"bonfire-api/internal/fields"
	"context"
	"time"
)

type Cache interface {
	Delete(ctx context.Context, id fields.ID) error
	DeleteBatch(ctx context.Context, ids []fields.ID) error
	Get(ctx context.Context, id fields.ID) (*Session, error)
	Set(ctx context.Context, sess *Session) error
}

type Repository interface {
	Create(ctx context.Context, s *Session) (*Session, error)
	DeleteBatchExpired(ctx context.Context, now time.Time, limitVal int) error
	Get(ctx context.Context, id fields.ID) (*Session, error)
	ListValidByUserID(ctx context.Context, userID fields.ID, now fields.Timestamp, limit int) ([]*Session, error)
	Revoke(ctx context.Context, id fields.ID, userID fields.ID, now fields.Timestamp) error
	RevokeAll(ctx context.Context, userID fields.ID, now fields.Timestamp) error
	RotateRefreshTokenHash(ctx context.Context, id fields.ID, oldHash fields.TokenHash, newHash fields.TokenHash, clientIP fields.IP, userAgent fields.UserAgent, expiresAt fields.Timestamp, now fields.Timestamp) (*Session, error)
}

type OutboxRepository interface {
	Publish(ctx context.Context, eventType string, payload any, now fields.Timestamp) error
}

type TX interface {
	ExecTx(ctx context.Context, fn func(txCtx context.Context) error) error
}
