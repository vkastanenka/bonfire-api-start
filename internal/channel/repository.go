package channel

import (
	"context"
	"time"

	"bonfire-api/internal/user"

	"github.com/google/uuid"
)

type ChannelRepository interface {
	Create(ctx context.Context, ch *Channel) (*Channel, error)
	Delete(ctx context.Context, id uuid.UUID) error
	Get(ctx context.Context, id uuid.UUID) (*Channel, error)
	GetBatch(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]*Channel, error)
	GetForUpdate(ctx context.Context, id uuid.UUID) (*Channel, error)
	UpdateGroup(ctx context.Context, id uuid.UUID, name *string, iconURL *string, updatedAt time.Time) (*Channel, error)
	UpdateLastMessage(ctx context.Context, id uuid.UUID, lastMessageID *uuid.UUID, lastMessageAt *time.Time, updatedAt time.Time) (*Channel, error)
}

type CachedChannelRepository interface {
	Get(ctx context.Context, id uuid.UUID) (*Channel, error)
	GetBatch(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]*Channel, error)
}

type MemberRepository interface {
	CountByChannelID(ctx context.Context, channelID uuid.UUID) (int, error)
	CreateBatch(ctx context.Context, members []*Member) ([]*Member, error)
	Delete(ctx context.Context, channelID uuid.UUID, userID uuid.UUID) error
	Get(ctx context.Context, channelID uuid.UUID, userID uuid.UUID) (*Member, error)
	GetBatchByChannelID(ctx context.Context, channelID uuid.UUID) ([]*Member, error)
	GetBatchByChannelIDs(ctx context.Context, channelIDs []uuid.UUID) (map[uuid.UUID][]*Member, error)
	IncrementPeersMentionCountByChannelID(ctx context.Context, channelID uuid.UUID, userID uuid.UUID, incrementAmount int, updatedAt time.Time) error
	ListVisibleByUserID(ctx context.Context, userID uuid.UUID, limit int) ([]*Member, error)
	UpdateIsVisible(ctx context.Context, channelID uuid.UUID, userID uuid.UUID, isVisible bool, updatedAt time.Time) (*Member, error)
	UpdateLastReadMessage(ctx context.Context, channelID uuid.UUID, userID uuid.UUID, lastReadMessageID *uuid.UUID, lastReadMessageAt time.Time, updatedAt time.Time, mentionCount *int) (*Member, error)
	UpdateMutedUntil(ctx context.Context, channelID uuid.UUID, userID uuid.UUID, mutedUntil *time.Time, updatedAt time.Time) (*Member, error)
	UpdatePinnedAt(ctx context.Context, channelID uuid.UUID, userID uuid.UUID, pinnedAt *time.Time, updatedAt time.Time) (*Member, error)
}

type CachedMemberRepository interface {
	Get(ctx context.Context, channelID uuid.UUID, userID uuid.UUID) (*Member, error)
	GetBatchByChannelID(ctx context.Context, channelID uuid.UUID) ([]*Member, error)
	GetBatchByChannelIDs(ctx context.Context, channelIDs []uuid.UUID) (map[uuid.UUID][]*Member, error)
	ListVisibleByUserID(ctx context.Context, userID uuid.UUID, limit int) ([]*Member, error)
}

type MessageRepository interface {
	CountByChannelID(ctx context.Context, channelID uuid.UUID) (int, error)
	Create(ctx context.Context, msg *Message) (*Message, error)
	CreateAndMention(ctx context.Context, msg *Message, channelID uuid.UUID, userID uuid.UUID, updatedAt time.Time) (*Message, error)
	CreateBatch(ctx context.Context, messages []*Message) ([]*Message, error)
	CreateBatchAndMention(ctx context.Context, messages []*Message, channelID uuid.UUID, userID uuid.UUID, updatedAt time.Time) ([]*Message, error)
	Delete(ctx context.Context, id uuid.UUID) error
	Get(ctx context.Context, id uuid.UUID) (*Message, error)
	ListAfterByChannelID(ctx context.Context, channelID uuid.UUID, cursorID uuid.UUID, limit int) ([]*Message, bool, error)
	ListAroundByChannelID(ctx context.Context, channelID uuid.UUID, cursorID uuid.UUID, beforeLimit int, afterLimit int) ([]*Message, bool, bool, error)
	ListBeforeByChannelID(ctx context.Context, channelID uuid.UUID, cursorID uuid.UUID, limit int) ([]*Message, bool, error)
	ListPinnedByChannelID(ctx context.Context, channelID uuid.UUID, cursorID *uuid.UUID, cursorPinnedAt *time.Time, limit int) ([]*Message, bool, error)
	UpdateContent(ctx context.Context, id uuid.UUID, content string, editedAt time.Time, updatedAt time.Time) (*Message, error)
	UpdatePinnedAt(ctx context.Context, id uuid.UUID, pinnedAt *time.Time, updatedAt time.Time) (*Message, error)
}

type CachedMessageRepository interface {
	Get(ctx context.Context, id uuid.UUID) (*Message, error)
	ListAfterByChannelID(ctx context.Context, channelID uuid.UUID, cursorID uuid.UUID, limit int) ([]*Message, bool, error)
	ListAroundByChannelID(ctx context.Context, channelID uuid.UUID, cursorID uuid.UUID, beforeLimit int, afterLimit int) ([]*Message, bool, bool, error)
	ListBeforeByChannelID(ctx context.Context, channelID uuid.UUID, cursorID uuid.UUID, limit int) ([]*Message, bool, error)
}

type ReactionRepository interface {
	CountByEmoji(ctx context.Context, messageID uuid.UUID, emoji string) (int, error)
	Create(ctx context.Context, rx *Reaction) (*Reaction, error)
	Delete(ctx context.Context, messageID uuid.UUID, userID uuid.UUID, emoji string) error
	Get(ctx context.Context, messageID uuid.UUID, userID uuid.UUID, emoji string) (*Reaction, error)
	GetBatchSummaryByMessageIDs(ctx context.Context, userID uuid.UUID, messageIDs []uuid.UUID) (map[uuid.UUID]*ReactionSummary, error)
}

type OutboxRepository interface {
	Publish(ctx context.Context, eventType string, payload any, now time.Time) error
}

type RelationRepository interface {
	HasIncomingBlock(ctx context.Context, actorID uuid.UUID, peerIDs []uuid.UUID) error
}

type UserRepository interface {
	Get(ctx context.Context, id uuid.UUID) (*user.User, error)
	GetBatch(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]*user.User, error)
}

type CachedUserRepository interface {
	Get(ctx context.Context, id uuid.UUID) (*user.User, error)
	GetBatch(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]*user.User, error)
	GetBatchValid(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]*user.User, error)
	GetValid(ctx context.Context, id uuid.UUID) (*user.User, error)
}

type TX interface {
	ExecTx(ctx context.Context, fn func(txCtx context.Context) error) error
}
