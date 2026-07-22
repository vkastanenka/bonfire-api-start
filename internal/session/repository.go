// internal/session/repository.go
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
	DeleteAllExcept(ctx context.Context, userID uuid.UUID, exceptSessionID uuid.UUID) error
}
