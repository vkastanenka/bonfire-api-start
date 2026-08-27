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
	ListTypeByUserID(ctx context.Context, userID fields.ID, relType Type, limit int) ([]*Relation, error)
	Save(ctx context.Context, rel *Relation) (*Relation, error)
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

type MemberRepository interface {
	CountByChannelID(ctx context.Context, channelID fields.ID) (int, error)
	CreateBatch(ctx context.Context, members []*channel.Member) ([]*channel.Member, error)
	Delete(ctx context.Context, channelID fields.ID, userID fields.ID) error
	Get(ctx context.Context, channelID fields.ID, userID fields.ID) (*channel.Member, error)
	GetBatchByChannelID(ctx context.Context, channelID fields.ID) ([]*channel.Member, error)
	GetBatchByChannelIDs(ctx context.Context, channelIDs []fields.ID) (map[fields.ID][]*channel.Member, error)
	IncrementPeersMentionCountByChannelID(ctx context.Context, channelID fields.ID, userID fields.ID, incrementAmount int, updatedAt fields.Timestamp) error
	ListVisibleByUserID(ctx context.Context, userID fields.ID, limit int) ([]*channel.Member, error)
	Require(ctx context.Context, channelID fields.ID, userID fields.ID) (*channel.Member, error)
	UpdateIsVisible(ctx context.Context, channelID fields.ID, userID fields.ID, isVisible bool, updatedAt fields.Timestamp) (*channel.Member, error)
	UpdateLastReadMessage(ctx context.Context, channelID fields.ID, userID fields.ID, lastReadMessageID fields.ID, lastReadMessageAt fields.Timestamp, updatedAt fields.Timestamp, mentionCount *int) (*channel.Member, error)
	UpdateMutedUntil(ctx context.Context, channelID fields.ID, userID fields.ID, mutedUntil fields.Timestamp, updatedAt fields.Timestamp) (*channel.Member, error)
	UpdatePinnedAt(ctx context.Context, channelID fields.ID, userID fields.ID, pinnedAt fields.Timestamp, updatedAt fields.Timestamp) (*channel.Member, error)
}

type OutboxRepository interface {
	Publish(ctx context.Context, variant string, payload any) error
}

type UserRepository interface {
	Get(ctx context.Context, id fields.ID) (*user.User, error)
	GetBatch(ctx context.Context, ids []fields.ID) (map[fields.ID]*user.User, error)
}

type UserCache interface {
	GetPresence(ctx context.Context, userID fields.ID) (user.Presence, error)
	GetBatchPresence(ctx context.Context, userIDs []fields.ID) (map[fields.ID]user.Presence, error)
	SetPresence(ctx context.Context, userID fields.ID, p user.Presence) error
	SetBatchPresence(ctx context.Context, items map[fields.ID]user.Presence) error
}

type TX interface {
	ExecTx(ctx context.Context, fn func(txCtx context.Context) error) error
}
