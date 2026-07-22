package user

import (
	"context"

	"github.com/google/uuid"
)

type Repository interface {
	Create(ctx context.Context, u *User) error
	Save(ctx context.Context, u *User) error
	SaveProfile(ctx context.Context, u *User) error
	Get(ctx context.Context, id uuid.UUID) (*User, error)
	GetByEmail(ctx context.Context, email Email) (*User, error)
	GetByUsername(ctx context.Context, username Username) (*User, error)
	CheckAvailability(ctx context.Context, email Email, username Username) (bool, bool, error)
}
