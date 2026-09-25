package relation

import (
	"bonfire-api/internal/channel"
	"bonfire-api/internal/presence"
	"bonfire-api/internal/user"
	"context"
	"time"

	"github.com/google/uuid"
)

type Cache interface {
	AddFriendID(ctx context.Context, userID uuid.UUID, friendID uuid.UUID) error
	AddFriendPair(ctx context.Context, userA uuid.UUID, userB uuid.UUID) error
	AddIncomingBlockID(ctx context.Context, userID uuid.UUID, blockerUserID uuid.UUID) error
	AddIncomingPendingID(ctx context.Context, userID uuid.UUID, senderID uuid.UUID) error
	AddOutgoingBlockID(ctx context.Context, userID uuid.UUID, blockedUserID uuid.UUID) error
	AddOutgoingPendingID(ctx context.Context, userID uuid.UUID, targetID uuid.UUID) error
	AddPendingPair(ctx context.Context, actorID uuid.UUID, peerID uuid.UUID) error
	BlockUser(ctx context.Context, blockerID uuid.UUID, targetID uuid.UUID) error
	DeleteBlocksPair(ctx context.Context, blockerID uuid.UUID, targetID uuid.UUID) error
	DeleteFriendsIndex(ctx context.Context, userID uuid.UUID) error
	DeleteFriendsPair(ctx context.Context, userA uuid.UUID, userB uuid.UUID) error
	DeleteIncomingBlocksIndex(ctx context.Context, userID uuid.UUID) error
	DeleteIncomingPendingsIndex(ctx context.Context, userID uuid.UUID) error
	DeleteOutgoingBlocksIndex(ctx context.Context, userID uuid.UUID) error
	DeleteOutgoingPendingsIndex(ctx context.Context, userID uuid.UUID) error
	DeletePendingPair(ctx context.Context, actorID uuid.UUID, peerID uuid.UUID) error
	GetFriendIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error)
	GetIncomingBlockIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error)
	GetIncomingPendingIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error)
	GetOutgoingBlockIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error)
	GetOutgoingPendingIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error)
	RemoveFriendID(ctx context.Context, userID uuid.UUID, friendID uuid.UUID) error
	RemoveFriendPair(ctx context.Context, userA uuid.UUID, userB uuid.UUID) error
	RemoveIncomingBlockID(ctx context.Context, userID uuid.UUID, blockerUserID uuid.UUID) error
	RemoveIncomingPendingID(ctx context.Context, userID uuid.UUID, senderID uuid.UUID) error
	RemoveOutgoingBlockID(ctx context.Context, userID uuid.UUID, blockedUserID uuid.UUID) error
	RemoveOutgoingPendingID(ctx context.Context, userID uuid.UUID, targetID uuid.UUID) error
	RemovePendingPair(ctx context.Context, actorID uuid.UUID, peerID uuid.UUID) error
	SetFriendIDs(ctx context.Context, userID uuid.UUID, friendIDs []uuid.UUID) error
	SetIncomingBlockIDs(ctx context.Context, userID uuid.UUID, blockIDs []uuid.UUID) error
	SetIncomingPendingIDs(ctx context.Context, userID uuid.UUID, pendingIDs []uuid.UUID) error
	SetOutgoingBlockIDs(ctx context.Context, userID uuid.UUID, blockIDs []uuid.UUID) error
	SetOutgoingPendingIDs(ctx context.Context, userID uuid.UUID, pendingIDs []uuid.UUID) error
	UnblockUser(ctx context.Context, blockerID uuid.UUID, targetID uuid.UUID) error
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
	GetVisibleMembersByUserID(ctx context.Context, userID uuid.UUID, limit int) ([]*channel.Member, bool, error)
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
