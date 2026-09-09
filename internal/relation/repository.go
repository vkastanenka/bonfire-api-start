package relation

import (
	"context"

	"bonfire-api/internal/channel"
	"bonfire-api/internal/fields"
	"bonfire-api/internal/user"
)

type Repository interface {
	DeleteByUserID(ctx context.Context, user1ID fields.ID, user2ID fields.ID, actorID fields.ID) error
	Get(ctx context.Context, user1ID fields.ID, user2ID fields.ID) (*Relation, error)
	GetForUpdate(ctx context.Context, user1ID fields.ID, user2ID fields.ID) (*Relation, error)
	HasIncomingBlock(ctx context.Context, actorID fields.ID, peerIDs []fields.ID) error
	ListFriendsByUserID(ctx context.Context, userID fields.ID, limit int) ([]*Relation, error)
	ListIncomingBlocksByUserID(ctx context.Context, userID fields.ID, limit int) ([]*Relation, error)
	ListIncomingPendingByUserID(ctx context.Context, userID fields.ID, limit int) ([]*Relation, error)
	ListOutgoingBlocksByUserID(ctx context.Context, userID fields.ID, limit int) ([]*Relation, error)
	ListTypeByUserID(ctx context.Context, userID fields.ID, relType Type, limit int) ([]*Relation, error)
	Save(ctx context.Context, rel *Relation) (*Relation, error)
}

type CachedRepository interface {
	GetBlockedByIDs(ctx context.Context, userID fields.ID) ([]fields.ID, error)
	GetBlocklistIDs(ctx context.Context, userID fields.ID) ([]fields.ID, error)
	GetFriends(ctx context.Context, userID fields.ID) (map[fields.ID]fields.ID, error)
}

type ChannelRepository interface {
	Create(ctx context.Context, ch *channel.Channel) (*channel.Channel, error)
	Delete(ctx context.Context, id fields.ID) error
	Get(ctx context.Context, id fields.ID) (*channel.Channel, error)
	GetBatch(ctx context.Context, ids []fields.ID) (map[fields.ID]*channel.Channel, error)
	GetForUpdate(ctx context.Context, id fields.ID) (*channel.Channel, error)
	UpdateGroup(ctx context.Context, id fields.ID, name channel.ChannelName, iconURL fields.URL, updatedAt fields.Timestamp) (*channel.Channel, error)
	UpdateLastMessage(ctx context.Context, id fields.ID, lastMessageID fields.ID, lastMessageAt fields.Timestamp, updatedAt fields.Timestamp) (*channel.Channel, error)
}

type CachedChannelRepository interface {
	Get(ctx context.Context, id fields.ID) (*channel.Channel, error)
	GetBatch(ctx context.Context, ids []fields.ID) (map[fields.ID]*channel.Channel, error)
}

type MemberRepository interface {
	CountByChannelID(ctx context.Context, channelID fields.ID) (int, error)
	CreateBatch(ctx context.Context, members []*channel.Member) ([]*channel.Member, error)
	Delete(ctx context.Context, channelID fields.ID, userID fields.ID) error
	Get(ctx context.Context, channelID fields.ID, userID fields.ID) (*channel.Member, error)
	GetBatchByChannelID(ctx context.Context, channelID fields.ID) ([]*channel.Member, error)
	GetBatchByChannelIDs(ctx context.Context, channelIDs []fields.ID) (map[fields.ID][]*channel.Member, error)
	IncrementPeersMentionCountByChannelID(ctx context.Context, channelID fields.ID, userID fields.ID, incrementAmount int, updatedAt fields.Timestamp) error
	ListVisibleByUserID(ctx context.Context, userID fields.ID, limit int) ([]*channel.Member, error)
	UpdateIsVisible(ctx context.Context, channelID fields.ID, userID fields.ID, isVisible bool, updatedAt fields.Timestamp) (*channel.Member, error)
	UpdateLastReadMessage(ctx context.Context, channelID fields.ID, userID fields.ID, lastReadMessageID fields.ID, lastReadMessageAt fields.Timestamp, updatedAt fields.Timestamp, mentionCount *int) (*channel.Member, error)
	UpdateMutedUntil(ctx context.Context, channelID fields.ID, userID fields.ID, mutedUntil fields.Timestamp, updatedAt fields.Timestamp) (*channel.Member, error)
	UpdatePinnedAt(ctx context.Context, channelID fields.ID, userID fields.ID, pinnedAt fields.Timestamp, updatedAt fields.Timestamp) (*channel.Member, error)
}

type CachedMemberRepository interface {
	Get(ctx context.Context, channelID fields.ID, userID fields.ID) (*channel.Member, error)
	GetBatchByChannelID(ctx context.Context, channelID fields.ID) ([]*channel.Member, error)
	GetBatchByChannelIDs(ctx context.Context, channelIDs []fields.ID) (map[fields.ID][]*channel.Member, error)
	ListVisibleByUserID(ctx context.Context, userID fields.ID, limit int) ([]*channel.Member, error)
}

type OutboxRepository interface {
	Publish(ctx context.Context, eventType string, payload any, now fields.Timestamp) error
}

type UserRepository interface {
	Get(ctx context.Context, id fields.ID) (*user.User, error)
	GetBatch(ctx context.Context, ids []fields.ID) (map[fields.ID]*user.User, error)
}

type CachedUserRepository interface {
	Get(ctx context.Context, id fields.ID) (*user.User, error)
	GetBatch(ctx context.Context, ids []fields.ID) (map[fields.ID]*user.User, error)
}

type TX interface {
	ExecTx(ctx context.Context, fn func(txCtx context.Context) error) error
}
