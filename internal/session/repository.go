// internal/session/repository.go
package session

import (
	"context"
	"net/netip"
	"time"

	"github.com/google/uuid"
)

type CreateParams struct {
	ID               uuid.UUID
	UserID           uuid.UUID
	RefreshTokenHash []byte
	ExpiresAt        time.Time
	ClientIP         netip.Addr
	UserAgent        string
	OS               string
	Browser          string
}

type UpdateRefreshTokenParams struct {
	ID               uuid.UUID
	RefreshTokenHash []byte
	ExpiresAt        time.Time
}

type DeleteAllExceptParams struct {
	UserID    uuid.UUID
	SessionID uuid.UUID
}

type Repository interface {
	Create(ctx context.Context, p CreateParams) (Session, error)
	GetByID(ctx context.Context, id uuid.UUID) (Session, error)
	UpdateRefreshToken(ctx context.Context, p UpdateRefreshTokenParams) (Session, error)
	UpdateLastSeen(ctx context.Context, id uuid.UUID) (Session, error)
	Revoke(ctx context.Context, id uuid.UUID) (Session, error)
	Delete(ctx context.Context, id uuid.UUID) error
	DeleteAllExcept(ctx context.Context, p DeleteAllExceptParams) error
}
