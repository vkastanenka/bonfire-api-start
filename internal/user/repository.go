// internal/user/repository.go
package user

import (
	"context"

	"github.com/google/uuid"
)

type Repository interface {
	CheckAvailability(ctx context.Context, email, username string) (bool, bool, error)
	GetByID(ctx context.Context, id uuid.UUID) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	GetByUsername(ctx context.Context, username string) (*User, error)
	Save(ctx context.Context, u *User) error
}
