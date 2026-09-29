package repository

import (
	"bonfire-api/internal/channel"
	"bonfire-api/internal/presence"
	"bonfire-api/internal/session"
	"bonfire-api/internal/user"
	"context"

	"github.com/google/uuid"
)

type ChannelCache interface {
	CreateGroup(ctx context.Context, ch *channel.Channel, members []*channel.Member) error
	DeleteGroup(ctx context.Context, channelID uuid.UUID, memberIDs []uuid.UUID) error
	Get(ctx context.Context, id uuid.UUID) (*channel.Channel, error)
	GetBatch(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]*channel.Channel, []uuid.UUID, error)
	GetUserChannelIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, bool, error)
	RemoveUserChannelID(ctx context.Context, userID uuid.UUID, channelID uuid.UUID) error
	Set(ctx context.Context, ch *channel.Channel) error
	SetBatch(ctx context.Context, channels map[uuid.UUID]*channel.Channel) error
	SetUserChannelIDs(ctx context.Context, userID uuid.UUID, members []*channel.Member) error
}

type GatewayCache interface {
	GetBatchUsers(ctx context.Context, userIDs []uuid.UUID) (map[uuid.UUID][]uuid.UUID, error)
	GetUserSessions(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, []uuid.UUID, map[uuid.UUID]uuid.UUID, error)
	Heartbeat(ctx context.Context, nodeID uuid.UUID, userID uuid.UUID, sessionID uuid.UUID) error
	RegisterSession(ctx context.Context, nodeID uuid.UUID, userID uuid.UUID, sessionID uuid.UUID, p presence.Presence) (bool, error)
	RemoveBatchUsers(ctx context.Context, nodeID uuid.UUID, userIDs []uuid.UUID) error
	SetUserSessions(ctx context.Context, userID uuid.UUID, sessions map[uuid.UUID]uuid.UUID) error
	UnregisterSession(ctx context.Context, nodeID uuid.UUID, userID uuid.UUID, sessionID uuid.UUID) (bool, error)
}

type MemberCache interface {
	Add(ctx context.Context, channelID uuid.UUID, members []*channel.Member) error
	Get(ctx context.Context, channelID uuid.UUID, userID uuid.UUID) (*channel.Member, error)
	GetBatchByChannelIDs(ctx context.Context, channelIDs []uuid.UUID) (map[uuid.UUID][]*channel.Member, []uuid.UUID, error)
	Invalidate(ctx context.Context, channelID uuid.UUID, userID uuid.UUID) error
	InvalidateChannel(ctx context.Context, channelID uuid.UUID) error
	Remove(ctx context.Context, channelID uuid.UUID, userID uuid.UUID) error
	SetBatchByChannelIDs(ctx context.Context, channelMembersMap map[uuid.UUID][]*channel.Member) error
	SetUserMembers(ctx context.Context, userID uuid.UUID, members []*channel.Member) error
}

type MessageCache interface {
	Delete(ctx context.Context, channelID uuid.UUID, msgID uuid.UUID) error
	Get(ctx context.Context, id uuid.UUID) (*channel.Message, error)
	GetAfterByChannelID(ctx context.Context, channelID uuid.UUID, cursorID uuid.UUID, limit int) ([]*channel.Message, bool, bool, error)
	GetAroundByChannelID(ctx context.Context, channelID uuid.UUID, cursorID uuid.UUID, beforeLimit int, afterLimit int) ([]*channel.Message, bool, bool, bool, error)
	GetBeforeByChannelID(ctx context.Context, channelID uuid.UUID, cursorID uuid.UUID, limit int) ([]*channel.Message, bool, bool, error)
	GetRecentByChannelID(ctx context.Context, channelID uuid.UUID, limit int) ([]*channel.Message, bool, error)
	Set(ctx context.Context, msg *channel.Message) error
	SetBatch(ctx context.Context, channelID uuid.UUID, messages []*channel.Message) error
}

type RelationCache interface {
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

type SessionCache interface {
	Delete(ctx context.Context, id uuid.UUID) error
	DeleteBatch(ctx context.Context, ids []uuid.UUID) error
	Get(ctx context.Context, id uuid.UUID) (*session.Session, error)
	GetBatch(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]*session.Session, []uuid.UUID, error)
	Set(ctx context.Context, sess *session.Session) error
	SetBatch(ctx context.Context, sessions map[uuid.UUID]*session.Session) error
}

type UserCache interface {
	Delete(ctx context.Context, id uuid.UUID) error
	DeleteBatch(ctx context.Context, ids []uuid.UUID) error
	Get(ctx context.Context, id uuid.UUID) (*user.User, error)
	GetBatch(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]*user.User, []uuid.UUID, error)
	Set(ctx context.Context, usr *user.User) error
	SetBatch(ctx context.Context, users map[uuid.UUID]*user.User) error
}
