package session

import (
	"context"

	"github.com/google/uuid"
)

type Cache interface {
	Delete(ctx context.Context, id uuid.UUID) error
	DeleteBatch(ctx context.Context, ids []uuid.UUID) error
	Get(ctx context.Context, id uuid.UUID) (*Session, error)
	Set(ctx context.Context, sess *Session) error
}
