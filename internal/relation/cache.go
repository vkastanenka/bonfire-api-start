package relation

import (
	"bonfire-api/internal/channel"
	"bonfire-api/internal/fields"
	"bonfire-api/internal/presence"
	"bonfire-api/internal/user"
	"context"
	"time"
)

type PresenceCache interface {
	GetBatchNodeUsers(ctx context.Context, userIDs []fields.ID) (map[fields.ID][]fields.ID, error)
	GetBatchPresence(ctx context.Context, userIDs []fields.ID) (map[fields.ID]presence.Presence, error)
	GetPresence(ctx context.Context, userID fields.ID) (presence.Presence, error)
	GetSessionNode(ctx context.Context, userID fields.ID, sessionID fields.ID) (fields.ID, bool, error)
	Heartbeat(ctx context.Context, nodeID fields.ID, userID fields.ID, sessionID fields.ID) error
	RegisterNodeSession(ctx context.Context, nodeID fields.ID, userID fields.ID, sessionID fields.ID, p presence.Presence) (bool, presence.Presence, error)
	RemoveBatchNodeUsers(ctx context.Context, nodeID fields.ID, userIDs []fields.ID) error
	SetPresence(ctx context.Context, userID fields.ID, p presence.Presence) error
	UnregisterNodeSession(ctx context.Context, nodeID fields.ID, userID fields.ID, sessionID fields.ID) (bool, error)
}

type UserCache interface {
	AddChannelID(ctx context.Context, userID fields.ID, channelID fields.ID) error
	AddFriend(ctx context.Context, userID fields.ID, friendID fields.ID, channelID fields.ID) error
	AddFriendPair(ctx context.Context, userA fields.ID, userB fields.ID, channelID fields.ID) error
	BlockUser(ctx context.Context, blockerID fields.ID, targetID fields.ID) error
	Delete(ctx context.Context, id fields.ID) error
	DeleteBatch(ctx context.Context, ids []fields.ID) error
	Get(ctx context.Context, id fields.ID) (*user.User, error)
	GetBatch(ctx context.Context, ids []fields.ID) (map[fields.ID]*user.User, []fields.ID, error)
	GetBlockedByIDs(ctx context.Context, userID fields.ID) ([]fields.ID, error)
	GetBlocklistIDs(ctx context.Context, userID fields.ID) ([]fields.ID, error)
	GetChannelIDs(ctx context.Context, userID fields.ID) ([]fields.ID, error)
	GetFriends(ctx context.Context, userID fields.ID) (map[fields.ID]fields.ID, error)
	GetPeerIDs(ctx context.Context, userID fields.ID) ([]fields.ID, error)
	GetVisibleMembersByUserID(ctx context.Context, userID fields.ID, limit int) ([]*channel.Member, bool, error)
	RemoveChannelID(ctx context.Context, userID fields.ID, channelID fields.ID) error
	RemoveFriendPair(ctx context.Context, userA fields.ID, userB fields.ID) error
	Set(ctx context.Context, usr *user.User) error
	SetBatch(ctx context.Context, users map[fields.ID]*user.User) error
	SetBlockedByIDs(ctx context.Context, userID fields.ID, blockerIDs []fields.ID) error
	SetBlocklistIDs(ctx context.Context, userID fields.ID, blockedIDs []fields.ID) error
	SetChannelIDs(ctx context.Context, userID fields.ID, channelIDs []fields.ID) error
	SetFriends(ctx context.Context, userID fields.ID, friendsMap map[fields.ID]fields.ID) error
	UnblockUser(ctx context.Context, blockerID fields.ID, targetID fields.ID) error
	addRelationID(ctx context.Context, key string, targetID fields.ID, ttl time.Duration) error
	getRelationIDs(ctx context.Context, key string) ([]fields.ID, error)
	setRelationIDs(ctx context.Context, key string, ids []fields.ID, ttl time.Duration) error
}
