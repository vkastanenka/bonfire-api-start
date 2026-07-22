package session

import (
	"context"

	"github.com/google/uuid"
)

type Repository interface {
	Create(ctx context.Context, s *Session) error
	Save(ctx context.Context, s *Session) error
	Get(ctx context.Context, id uuid.UUID) (*Session, error)
	Delete(ctx context.Context, id uuid.UUID) error
	DeleteAllByUserID(ctx context.Context, userID uuid.UUID) error
	DeleteAllExcept(ctx context.Context, userID uuid.UUID, currentSessionID uuid.UUID) error
	DeleteAllExpired(ctx context.Context) error
}
