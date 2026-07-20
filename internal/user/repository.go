package user

import (
	"context"

	"github.com/google/uuid"
)

type Repository interface {
	CheckAvailability(ctx context.Context, email, username string) (bool, bool, error)
	Create(ctx context.Context, id uuid.UUID, email, username, password string) (User, error)
	Get(ctx context.Context, id uuid.UUID) (User, error)
	GetByEmail(ctx context.Context, email string) (User, error)
	GetByUsername(ctx context.Context, username string) (User, error)
	UpdatePassword(ctx context.Context, id uuid.UUID, passwordHash string) (User, error)
	MarkVerified(ctx context.Context, id uuid.UUID) (User, error)
	CreateProfile(ctx context.Context, userID uuid.UUID, displayName string) (UserProfile, error)
	GetProfile(ctx context.Context, userID uuid.UUID) (UserProfile, error)
}
