package user

import (
	"bonfire-api/internal/presence"
	"context"
	"time"

	"github.com/google/uuid"
)

type Repository interface {
	Availability(ctx context.Context, email *string, username *string) (bool, bool, error)
	Create(ctx context.Context, u *User) (*User, error)
	Get(ctx context.Context, id uuid.UUID) (*User, error)
	GetBatch(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	ListDeleteScheduled(ctx context.Context, currentTime time.Time, limitVal int) ([]*User, error)
	SetDeleteSchedule(ctx context.Context, id uuid.UUID, deleteScheduledAt *time.Time, disabledAt *time.Time, updatedAt time.Time) (*User, error)
	SetDisabled(ctx context.Context, id uuid.UUID, disabledAt *time.Time, updatedAt time.Time) (*User, error)
	Update(ctx context.Context, u *User) (*User, error)
	UpdateBatch(ctx context.Context, users []*User) ([]*User, error)
	UpdateEmail(ctx context.Context, id uuid.UUID, email string, updatedAt time.Time) (*User, error)
	UpdatePasswordHash(ctx context.Context, id uuid.UUID, passwordHash string, updatedAt time.Time) (*User, error)
	UpdatePhone(ctx context.Context, id uuid.UUID, phone *string, updatedAt time.Time) (*User, error)
	UpdatePresence(ctx context.Context, id uuid.UUID, presence *presence.Presence, presenceUntil *time.Time, updatedAt time.Time) (*User, error)
	UpdateProfile(ctx context.Context, id uuid.UUID, displayName string, bio *string, avatarURL *string, bannerColor *string, updatedAt time.Time) (*User, error)
	UpdateUsername(ctx context.Context, id uuid.UUID, username string, updatedAt time.Time) (*User, error)
	Verify(ctx context.Context, id uuid.UUID, verifiedAt *time.Time, updatedAt time.Time) (*User, error)
}

type CachedRepository interface {
	Get(ctx context.Context, id uuid.UUID) (*User, error)
	GetBatch(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]*User, error)
}

type OutboxRepository interface {
	Publish(ctx context.Context, eventType string, payload any, now time.Time) error
}

type TX interface {
	ExecTx(ctx context.Context, fn func(txCtx context.Context) error) error
}
