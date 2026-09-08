package channel

import (
	"bonfire-api/internal/fields"
	"bonfire-api/internal/presence"
	"bonfire-api/internal/user"
	"context"
)

type ChannelCache interface {
	AddMembers(ctx context.Context, channelID fields.ID, members []*Member) error
	CreateGroup(ctx context.Context, ch *Channel, members []*Member) error
	Delete(ctx context.Context, id fields.ID) error
	Get(ctx context.Context, id fields.ID) (*Channel, error)
	GetBatchMembersByChannelIDs(ctx context.Context, channelIDs []fields.ID) (map[fields.ID][]*Member, []fields.ID, error)
	GetMember(ctx context.Context, channelID fields.ID, userID fields.ID) (*Member, error)
	InvalidateMember(ctx context.Context, channelID fields.ID, userID fields.ID) error
	InvalidateMembers(ctx context.Context, channelID fields.ID) error
	Set(ctx context.Context, ch *Channel) error
	SetBatchMembers(ctx context.Context, channelMembersMap map[fields.ID][]*Member) error
}

type UserCache interface {
	AddChannelID(ctx context.Context, userID fields.ID, channelID fields.ID) error
	AddFriendID(ctx context.Context, userID fields.ID, friendID fields.ID) error
	Delete(ctx context.Context, id fields.ID) error
	DeleteBatch(ctx context.Context, ids []fields.ID) error
	Get(ctx context.Context, id fields.ID) (*user.User, error)
	GetBatch(ctx context.Context, ids []fields.ID) (map[fields.ID]*user.User, []fields.ID, error)
	GetChannelIDs(ctx context.Context, userID fields.ID) ([]fields.ID, error)
	GetFriendIDs(ctx context.Context, userID fields.ID) ([]fields.ID, error)
	GetPeerIDs(ctx context.Context, userID fields.ID) ([]fields.ID, error)
	GetVisibleMembersByUserID(ctx context.Context, userID fields.ID, limit int) ([]*Member, bool, error)
	RemoveChannelID(ctx context.Context, userID fields.ID, channelID fields.ID) error
	RemoveFriendID(ctx context.Context, userID fields.ID, friendID fields.ID) error
	RemoveFriendPair(ctx context.Context, userA fields.ID, userB fields.ID) error
	Set(ctx context.Context, usr *user.User) error
	SetBatch(ctx context.Context, users map[fields.ID]*user.User) error
	SetChannelIDs(ctx context.Context, userID fields.ID, channelIDs []fields.ID) error
	SetFriendIDs(ctx context.Context, userID fields.ID, friendIDs []fields.ID) error
}

type PresenceCache interface {
	GetPresence(ctx context.Context, userID fields.ID) (presence.Presence, error)
	GetBatchPresence(ctx context.Context, userIDs []fields.ID) (map[fields.ID]presence.Presence, error)
	SetPresence(ctx context.Context, userID fields.ID, p presence.Presence) error
}
