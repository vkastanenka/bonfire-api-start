package user

import (
	"bonfire-api/internal/outbox"
	"bonfire-api/internal/presence"
	"context"
	"time"

	"github.com/google/uuid"
)

type Cache interface {
	Delete(ctx context.Context, id uuid.UUID) error
	DeleteBatch(ctx context.Context, ids []uuid.UUID) error
}

type Repository interface {
	Get(ctx context.Context, id uuid.UUID) (*User, error)
	GetBatch(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]*User, error)
	ListDeleteScheduled(ctx context.Context, currentTime time.Time, limitVal int) ([]*User, error)
	SetDeleteSchedule(ctx context.Context, id uuid.UUID, deleteScheduledAt *time.Time, disabledAt *time.Time, updatedAt time.Time) (*User, error)
	SetDisabled(ctx context.Context, id uuid.UUID, disabledAt *time.Time, updatedAt time.Time) (*User, error)
	UpdateBatch(ctx context.Context, users []*User) ([]*User, error)
	UpdateEmail(ctx context.Context, id uuid.UUID, email string, updatedAt time.Time) (*User, error)
	UpdatePasswordHash(ctx context.Context, id uuid.UUID, passwordHash string, updatedAt time.Time) (*User, error)
	UpdatePresence(ctx context.Context, id uuid.UUID, presence *presence.Presence, presenceUntil *time.Time, updatedAt time.Time) (*User, error)
	UpdateProfile(ctx context.Context, id uuid.UUID, displayName string, bio *string, avatarURL *string, bannerColor *string, updatedAt time.Time) (*User, error)
	UpdateUsername(ctx context.Context, id uuid.UUID, username string, updatedAt time.Time) (*User, error)
}

type CachedRepository interface {
	Get(ctx context.Context, id uuid.UUID) (*User, error)
	GetBatch(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]*User, error)
}

type CachedRelationRepository interface {
	GetFriendIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error)
}

type OutboxRepository interface {
	Create(ctx context.Context, e *outbox.Event) error
}

type SessionRepository interface {
	RevokeAll(ctx context.Context, userID uuid.UUID, now time.Time) ([]uuid.UUID, error)
}

type TX interface {
	ExecTx(ctx context.Context, fn func(txCtx context.Context) error) error
}
