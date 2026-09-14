package relation

import (
	"context"
	"time"

	"bonfire-api/internal/channel"
	"bonfire-api/internal/user"

	"github.com/google/uuid"
)

type Repository interface {
	DeleteByUserID(ctx context.Context, user1ID uuid.UUID, user2ID uuid.UUID, actorID uuid.UUID) error
	Get(ctx context.Context, user1ID uuid.UUID, user2ID uuid.UUID) (*Relation, error)
	GetForUpdate(ctx context.Context, user1ID uuid.UUID, user2ID uuid.UUID) (*Relation, error)
	HasIncomingBlock(ctx context.Context, actorID uuid.UUID, peerIDs []uuid.UUID) error
	ListFriendsByUserID(ctx context.Context, userID uuid.UUID, limit int) ([]*Relation, error)
	ListIncomingBlocksByUserID(ctx context.Context, userID uuid.UUID, limit int) ([]*Relation, error)
	ListIncomingPendingByUserID(ctx context.Context, userID uuid.UUID, limit int) ([]*Relation, error)
	ListOutgoingBlocksByUserID(ctx context.Context, userID uuid.UUID, limit int) ([]*Relation, error)
	ListTypeByUserID(ctx context.Context, userID uuid.UUID, relType Type, limit int) ([]*Relation, error)
	Save(ctx context.Context, rel *Relation) (*Relation, error)
}

type CachedRepository interface {
	GetBlockedByIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error)
	GetBlocklistIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error)
	GetFriends(ctx context.Context, userID uuid.UUID) (map[uuid.UUID]uuid.UUID, error)
	GetPendingIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error)
}

type ChannelRepository interface {
	Create(ctx context.Context, ch *channel.Channel) (*channel.Channel, error)
	Delete(ctx context.Context, id uuid.UUID) error
	Get(ctx context.Context, id uuid.UUID) (*channel.Channel, error)
	GetBatch(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]*channel.Channel, error)
	GetForUpdate(ctx context.Context, id uuid.UUID) (*channel.Channel, error)
	UpdateGroup(ctx context.Context, id uuid.UUID, name channel.ChannelName, iconURL string, updatedAt time.Time) (*channel.Channel, error)
	UpdateLastMessage(ctx context.Context, id uuid.UUID, lastMessageID uuid.UUID, lastMessageAt time.Time, updatedAt time.Time) (*channel.Channel, error)
}

type CachedChannelRepository interface {
	Get(ctx context.Context, id uuid.UUID) (*channel.Channel, error)
	GetBatch(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]*channel.Channel, error)
}

type MemberRepository interface {
	CountByChannelID(ctx context.Context, channelID uuid.UUID) (int, error)
	CreateBatch(ctx context.Context, members []*channel.Member) ([]*channel.Member, error)
	Delete(ctx context.Context, channelID uuid.UUID, userID uuid.UUID) error
	Get(ctx context.Context, channelID uuid.UUID, userID uuid.UUID) (*channel.Member, error)
	GetBatchByChannelID(ctx context.Context, channelID uuid.UUID) ([]*channel.Member, error)
	GetBatchByChannelIDs(ctx context.Context, channelIDs []uuid.UUID) (map[uuid.UUID][]*channel.Member, error)
	IncrementPeersMentionCountByChannelID(ctx context.Context, channelID uuid.UUID, userID uuid.UUID, incrementAmount int, updatedAt time.Time) error
	ListVisibleByUserID(ctx context.Context, userID uuid.UUID, limit int) ([]*channel.Member, error)
	UpdateIsVisible(ctx context.Context, channelID uuid.UUID, userID uuid.UUID, isVisible bool, updatedAt time.Time) (*channel.Member, error)
	UpdateLastReadMessage(ctx context.Context, channelID uuid.UUID, userID uuid.UUID, lastReadMessageID uuid.UUID, lastReadMessageAt time.Time, updatedAt time.Time, mentionCount *int) (*channel.Member, error)
	UpdateMutedUntil(ctx context.Context, channelID uuid.UUID, userID uuid.UUID, mutedUntil time.Time, updatedAt time.Time) (*channel.Member, error)
	UpdatePinnedAt(ctx context.Context, channelID uuid.UUID, userID uuid.UUID, pinnedAt time.Time, updatedAt time.Time) (*channel.Member, error)
}

type CachedMemberRepository interface {
	Get(ctx context.Context, channelID uuid.UUID, userID uuid.UUID) (*channel.Member, error)
	GetBatchByChannelID(ctx context.Context, channelID uuid.UUID) ([]*channel.Member, error)
	GetBatchByChannelIDs(ctx context.Context, channelIDs []uuid.UUID) (map[uuid.UUID][]*channel.Member, error)
	ListVisibleByUserID(ctx context.Context, userID uuid.UUID, limit int) ([]*channel.Member, error)
}

type OutboxRepository interface {
	Publish(ctx context.Context, eventType string, payload any, now time.Time) error
}

type UserRepository interface {
	Get(ctx context.Context, id uuid.UUID) (*user.User, error)
	GetBatch(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]*user.User, error)
}

type CachedUserRepository interface {
	Get(ctx context.Context, id uuid.UUID) (*user.User, error)
	GetBatch(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]*user.User, error)
}

type TX interface {
	ExecTx(ctx context.Context, fn func(txCtx context.Context) error) error
}
