package user

import (
	"context"

	"github.com/google/uuid"
)

type CheckAvailabilityParams struct {
	Email    string
	Username string
}

type CheckAvailabilityResult struct {
	EmailAvailable    bool
	UsernameAvailable bool
}

type CreateParams struct {
	ID           uuid.UUID
	Email        string
	Username     string
	PasswordHash string
}

type UpdatePasswordParams struct {
	ID           uuid.UUID
	PasswordHash string
}

type CreateProfileParams struct {
	UserID      uuid.UUID
	DisplayName string
}

type Repository interface {
	CheckAvailability(ctx context.Context, p CheckAvailabilityParams) (CheckAvailabilityResult, error)
	Create(ctx context.Context, p CreateParams) (User, error)
	Get(ctx context.Context, id uuid.UUID) (User, error)
	GetByEmail(ctx context.Context, email string) (User, error)
	GetByUsername(ctx context.Context, username string) (User, error)
	UpdatePassword(ctx context.Context, p UpdatePasswordParams) (User, error)
	MarkVerified(ctx context.Context, id uuid.UUID) (User, error)
	CreateProfile(ctx context.Context, p CreateProfileParams) (UserProfile, error)
	GetProfile(ctx context.Context, userID uuid.UUID) (UserProfile, error)
}
