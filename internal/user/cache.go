package user

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type Cache interface {
	AddChannelID(ctx context.Context, userID uuid.UUID, channelID uuid.UUID) error
	AddFriend(ctx context.Context, userID uuid.UUID, friendID uuid.UUID, channelID uuid.UUID) error
	AddFriendPair(ctx context.Context, userA uuid.UUID, userB uuid.UUID, channelID uuid.UUID) error
	AddPendingID(ctx context.Context, userID uuid.UUID, pendingUserID uuid.UUID) error
	BlockUser(ctx context.Context, blockerID uuid.UUID, targetID uuid.UUID) error
	Delete(ctx context.Context, id uuid.UUID) error
	DeleteBatch(ctx context.Context, ids []uuid.UUID) error
	Get(ctx context.Context, id uuid.UUID) (*User, error)
	GetBatch(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]*User, []uuid.UUID, error)
	GetBlockedByIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error)
	GetBlocklistIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error)
	GetChannelIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error)
	GetFriends(ctx context.Context, userID uuid.UUID) (map[uuid.UUID]uuid.UUID, error)
	GetPeerIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error)
	GetPendingIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error)
	// GetVisibleMembersByUserID(ctx context.Context, userID uuid.UUID, limit int) ([]*channel.Member, bool, error)
	RemoveChannelID(ctx context.Context, userID uuid.UUID, channelID uuid.UUID) error
	RemoveFriendPair(ctx context.Context, userA uuid.UUID, userB uuid.UUID) error
	RemovePendingID(ctx context.Context, userID uuid.UUID, pendingUserID uuid.UUID) error
	RemovePendingPair(ctx context.Context, userA uuid.UUID, userB uuid.UUID) error
	Set(ctx context.Context, usr *User) error
	SetBatch(ctx context.Context, users map[uuid.UUID]*User) error
	SetBlockedByIDs(ctx context.Context, userID uuid.UUID, blockerIDs []uuid.UUID) error
	SetBlocklistIDs(ctx context.Context, userID uuid.UUID, blockedIDs []uuid.UUID) error
	SetChannelIDs(ctx context.Context, userID uuid.UUID, channelIDs []uuid.UUID) error
	SetFriends(ctx context.Context, userID uuid.UUID, friendsMap map[uuid.UUID]uuid.UUID) error
	SetPendingIDs(ctx context.Context, userID uuid.UUID, pendingIDs []uuid.UUID) error
	UnblockUser(ctx context.Context, blockerID uuid.UUID, targetID uuid.UUID) error
	addRelationID(ctx context.Context, key string, targetID uuid.UUID, ttl time.Duration) error
	getRelationIDs(ctx context.Context, key string) ([]uuid.UUID, error)
	setRelationIDs(ctx context.Context, key string, ids []uuid.UUID, ttl time.Duration) error
}
