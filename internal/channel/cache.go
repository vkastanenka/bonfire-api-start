package channel

import (
	"bonfire-api/internal/presence"
	"bonfire-api/internal/user"
	"context"
	"time"

	"github.com/google/uuid"
)

type ChannelCache interface {
	AddMembers(ctx context.Context, channelID uuid.UUID, members []*Member) error
	CreateGroup(ctx context.Context, ch *Channel, members []*Member) error
	Delete(ctx context.Context, id uuid.UUID) error
	Get(ctx context.Context, id uuid.UUID) (*Channel, error)
	GetBatch(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]*Channel, []uuid.UUID, error)
	GetBatchMembersByChannelIDs(ctx context.Context, channelIDs []uuid.UUID) (map[uuid.UUID][]*Member, []uuid.UUID, error)
	GetMember(ctx context.Context, channelID uuid.UUID, userID uuid.UUID) (*Member, error)
	InvalidateMember(ctx context.Context, channelID uuid.UUID, userID uuid.UUID) error
	InvalidateMembers(ctx context.Context, channelID uuid.UUID) error
	Set(ctx context.Context, ch *Channel) error
	SetBatch(ctx context.Context, channels map[uuid.UUID]*Channel) error
	SetBatchMembers(ctx context.Context, channelMembersMap map[uuid.UUID][]*Member) error
}

type MessageCache interface {
	Delete(ctx context.Context, channelID uuid.UUID, msgID uuid.UUID) error
	Get(ctx context.Context, id uuid.UUID) (*Message, error)
	GetAfterByChannelID(ctx context.Context, channelID uuid.UUID, cursorID uuid.UUID, limit int) ([]*Message, bool, bool, error)
	GetAroundByChannelID(ctx context.Context, channelID uuid.UUID, cursorID uuid.UUID, beforeLimit int, afterLimit int) ([]*Message, bool, bool, bool, error)
	GetBeforeByChannelID(ctx context.Context, channelID uuid.UUID, cursorID uuid.UUID, limit int) ([]*Message, bool, bool, error)
	GetRecentByChannelID(ctx context.Context, channelID uuid.UUID, limit int) ([]*Message, bool, error)
	Set(ctx context.Context, msg *Message) error
	SetBatch(ctx context.Context, channelID uuid.UUID, messages []*Message) error
	fetchAndUnmarshalBatch(ctx context.Context, ids []string) ([]*Message, error)
}

type PresenceCache interface {
	GetBatchNodeUsers(ctx context.Context, userIDs []uuid.UUID) (map[uuid.UUID][]uuid.UUID, error)
	GetBatchPresence(ctx context.Context, userIDs []uuid.UUID) (map[uuid.UUID]presence.Presence, error)
	GetPresence(ctx context.Context, userID uuid.UUID) (presence.Presence, error)
	GetSessionNode(ctx context.Context, userID uuid.UUID, sessionID uuid.UUID) (uuid.UUID, bool, error)
	Heartbeat(ctx context.Context, nodeID uuid.UUID, userID uuid.UUID, sessionID uuid.UUID) error
	RegisterNodeSession(ctx context.Context, nodeID uuid.UUID, userID uuid.UUID, sessionID uuid.UUID, p presence.Presence) (bool, presence.Presence, error)
	RemoveBatchNodeUsers(ctx context.Context, nodeID uuid.UUID, userIDs []uuid.UUID) error
	SetPresence(ctx context.Context, userID uuid.UUID, p presence.Presence) error
	UnregisterNodeSession(ctx context.Context, nodeID uuid.UUID, userID uuid.UUID, sessionID uuid.UUID) (bool, error)
}

type UserCache interface {
	AddChannelID(ctx context.Context, userID uuid.UUID, channelID uuid.UUID) error
	AddFriend(ctx context.Context, userID uuid.UUID, friendID uuid.UUID, channelID uuid.UUID) error
	AddFriendPair(ctx context.Context, userA uuid.UUID, userB uuid.UUID, channelID uuid.UUID) error
	AddPendingID(ctx context.Context, userID uuid.UUID, pendingUserID uuid.UUID) error
	BlockUser(ctx context.Context, blockerID uuid.UUID, targetID uuid.UUID) error
	Delete(ctx context.Context, id uuid.UUID) error
	DeleteBatch(ctx context.Context, ids []uuid.UUID) error
	Get(ctx context.Context, id uuid.UUID) (*user.User, error)
	GetBatch(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]*user.User, []uuid.UUID, error)
	GetBlockedByIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error)
	GetBlocklistIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error)
	GetChannelIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error)
	GetFriends(ctx context.Context, userID uuid.UUID) (map[uuid.UUID]uuid.UUID, error)
	GetPeerIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error)
	GetPendingIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error)
	GetVisibleMembersByUserID(ctx context.Context, userID uuid.UUID, limit int) ([]*Member, bool, error)
	RemoveChannelID(ctx context.Context, userID uuid.UUID, channelID uuid.UUID) error
	RemoveFriendPair(ctx context.Context, userA uuid.UUID, userB uuid.UUID) error
	RemovePendingID(ctx context.Context, userID uuid.UUID, pendingUserID uuid.UUID) error
	RemovePendingPair(ctx context.Context, userA uuid.UUID, userB uuid.UUID) error
	Set(ctx context.Context, usr *user.User) error
	SetBatch(ctx context.Context, users map[uuid.UUID]*user.User) error
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
