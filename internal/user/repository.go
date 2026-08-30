package user

import (
	"bonfire-api/internal/fields"
	"bonfire-api/internal/outbox"
	"context"
)

type Cache interface {
	GetBatchPresence(ctx context.Context, userIDs []fields.ID) (map[fields.ID]Presence, error)
	GetPresence(ctx context.Context, userID fields.ID) (Presence, error)
	SetBatchPresence(ctx context.Context, items map[fields.ID]Presence) error
	SetPresence(ctx context.Context, userID fields.ID, p Presence) error
}

type Repository interface {
	Availability(ctx context.Context, email *Email, username *Username) (bool, bool, error)
	Create(ctx context.Context, u *User) (*User, error)
	Get(ctx context.Context, id fields.ID) (*User, error)
	GetBatch(ctx context.Context, ids []fields.ID) (map[fields.ID]*User, error)
	GetByEmail(ctx context.Context, email Email) (*User, error)
	ListDeleteScheduled(ctx context.Context, currentTime fields.Timestamp, limitVal int) ([]*User, error)
	SetDeleteSchedule(ctx context.Context, id fields.ID, deleteScheduledAt fields.Timestamp, disabledAt fields.Timestamp, updatedAt fields.Timestamp) (*User, error)
	SetDisabled(ctx context.Context, id fields.ID, disabledAt fields.Timestamp, updatedAt fields.Timestamp) (*User, error)
	Update(ctx context.Context, u *User) (*User, error)
	UpdateBatch(ctx context.Context, users []*User) ([]*User, error)
	UpdateEmail(ctx context.Context, id fields.ID, email Email, updatedAt fields.Timestamp) (*User, error)
	UpdatePasswordHash(ctx context.Context, id fields.ID, passwordHash PasswordHash, updatedAt fields.Timestamp) (*User, error)
	UpdatePhone(ctx context.Context, id fields.ID, phone Phone, updatedAt fields.Timestamp) (*User, error)
	UpdatePresence(ctx context.Context, id fields.ID, presence PreferredPresence, presenceUntil fields.Timestamp, updatedAt fields.Timestamp) (*User, error)
	UpdateProfile(ctx context.Context, id fields.ID, displayName DisplayName, bio Bio, avatarURL fields.URL, bannerColor fields.HexColor, updatedAt fields.Timestamp) (*User, error)
	UpdateUsername(ctx context.Context, id fields.ID, username Username, updatedAt fields.Timestamp) (*User, error)
	Verify(ctx context.Context, id fields.ID, verifiedAt fields.Timestamp, updatedAt fields.Timestamp) (*User, error)
}

type OutboxRepository interface {
	Publish(ctx context.Context, eventType outbox.Type, payload outbox.Payload, now fields.Timestamp) error
}

type TX interface {
	ExecTx(ctx context.Context, fn func(txCtx context.Context) error) error
}
