package user

import (
	"bonfire-api/internal/fields"
	"context"
)

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

type Cache interface {
	AddChannelID(ctx context.Context, userID fields.ID, channelID fields.ID) error
	AddFriendID(ctx context.Context, userID fields.ID, friendID fields.ID) error
	Delete(ctx context.Context, id fields.ID) error
	DeleteBatch(ctx context.Context, ids []fields.ID) error
	Get(ctx context.Context, id fields.ID) (*User, error)
	GetBatch(ctx context.Context, ids []fields.ID) (map[fields.ID]*User, []fields.ID, error)
	GetChannelIDs(ctx context.Context, userID fields.ID) ([]fields.ID, error)
	GetFriendIDs(ctx context.Context, userID fields.ID) ([]fields.ID, error)
	GetPeerIDs(ctx context.Context, userID fields.ID) ([]fields.ID, error)
	RemoveChannelID(ctx context.Context, userID fields.ID, channelID fields.ID) error
	RemoveFriendID(ctx context.Context, userID fields.ID, friendID fields.ID) error
	Set(ctx context.Context, usr *User) error
	SetBatch(ctx context.Context, users map[fields.ID]*User) error
	SetChannelIDs(ctx context.Context, userID fields.ID, channelIDs []fields.ID) error
	SetFriendIDs(ctx context.Context, userID fields.ID, friendIDs []fields.ID) error
}

type OutboxRepository interface {
	Publish(ctx context.Context, eventType string, payload any, now fields.Timestamp) error
}

type TX interface {
	ExecTx(ctx context.Context, fn func(txCtx context.Context) error) error
}

type Broadcaster interface {
	BroadcastToPeers(ctx context.Context, actorID fields.ID, eventType string, payload interface{}) error
	BroadcastToUser(ctx context.Context, actorID, targetUserID fields.ID, eventType string, payload interface{}) error
}
