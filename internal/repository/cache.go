package repository

import (
	"bonfire-api/internal/channel"
	"bonfire-api/internal/fields"
	"bonfire-api/internal/user"
	"context"
	"time"
)

type ChannelCache interface {
	AddMembers(ctx context.Context, channelID fields.ID, members []*channel.Member) error
	CreateGroup(ctx context.Context, ch *channel.Channel, members []*channel.Member) error
	Delete(ctx context.Context, id fields.ID) error
	Get(ctx context.Context, id fields.ID) (*channel.Channel, error)
	GetBatchMembersByChannelIDs(ctx context.Context, channelIDs []fields.ID) (map[fields.ID][]*channel.Member, []fields.ID, error)
	InvalidateMember(ctx context.Context, channelID fields.ID, userID fields.ID) error
	InvalidateMembers(ctx context.Context, channelID fields.ID) error
	Set(ctx context.Context, ch *channel.Channel) error
	SetBatchMembers(ctx context.Context, channelMembersMap map[fields.ID][]*channel.Member) error
	GetMember(ctx context.Context, channelID fields.ID, userID fields.ID) (*channel.Member, error)
	SetBatch(ctx context.Context, channels map[fields.ID]*channel.Channel) error
	GetBatch(ctx context.Context, ids []fields.ID) (map[fields.ID]*channel.Channel, []fields.ID, error)
}

type MessageCache interface {
	Delete(ctx context.Context, channelID fields.ID, msgID fields.ID) error
	Get(ctx context.Context, id fields.ID) (*channel.Message, error)
	GetRecentByChannelID(ctx context.Context, channelID fields.ID, limit int) ([]*channel.Message, bool, error)
	Set(ctx context.Context, msg *channel.Message) error
	SetBatch(ctx context.Context, channelID fields.ID, messages []*channel.Message) error
	GetAroundByChannelID(ctx context.Context, channelID fields.ID, cursorID fields.ID, beforeLimit int, afterLimit int) ([]*channel.Message, bool, bool, bool, error)
	GetAfterByChannelID(ctx context.Context, channelID fields.ID, cursorID fields.ID, limit int) ([]*channel.Message, bool, bool, error)
	GetBeforeByChannelID(ctx context.Context, channelID fields.ID, cursorID fields.ID, limit int) ([]*channel.Message, bool, bool, error)
}

type UserCache interface {
	AddChannelID(ctx context.Context, userID fields.ID, channelID fields.ID) error
	AddFriend(ctx context.Context, userID fields.ID, friendID fields.ID, channelID fields.ID) error
	AddFriendPair(ctx context.Context, userA fields.ID, userB fields.ID, channelID fields.ID) error
	AddPendingID(ctx context.Context, userID fields.ID, pendingUserID fields.ID) error
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
	GetPendingIDs(ctx context.Context, userID fields.ID) ([]fields.ID, error)
	GetVisibleMembersByUserID(ctx context.Context, userID fields.ID, limit int) ([]*channel.Member, bool, error)
	RemoveChannelID(ctx context.Context, userID fields.ID, channelID fields.ID) error
	RemoveFriendPair(ctx context.Context, userA fields.ID, userB fields.ID) error
	RemovePendingID(ctx context.Context, userID fields.ID, pendingUserID fields.ID) error
	RemovePendingPair(ctx context.Context, userA fields.ID, userB fields.ID) error
	Set(ctx context.Context, usr *user.User) error
	SetBatch(ctx context.Context, users map[fields.ID]*user.User) error
	SetBlockedByIDs(ctx context.Context, userID fields.ID, blockerIDs []fields.ID) error
	SetBlocklistIDs(ctx context.Context, userID fields.ID, blockedIDs []fields.ID) error
	SetChannelIDs(ctx context.Context, userID fields.ID, channelIDs []fields.ID) error
	SetFriends(ctx context.Context, userID fields.ID, friendsMap map[fields.ID]fields.ID) error
	SetPendingIDs(ctx context.Context, userID fields.ID, pendingIDs []fields.ID) error
	UnblockUser(ctx context.Context, blockerID fields.ID, targetID fields.ID) error
	addRelationID(ctx context.Context, key string, targetID fields.ID, ttl time.Duration) error
	getRelationIDs(ctx context.Context, key string) ([]fields.ID, error)
	setRelationIDs(ctx context.Context, key string, ids []fields.ID, ttl time.Duration) error
}
