package channel

import (
	"context"
	"time"

	"bonfire-api/internal/outbox"
	"bonfire-api/internal/presence"
	"bonfire-api/internal/relation"
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
	CountBatchByUserID(ctx context.Context, userIDs []uuid.UUID) (map[uuid.UUID]int, error)
	CountByChannelID(ctx context.Context, channelID uuid.UUID) (int, error)
	CountByUserID(ctx context.Context, userID uuid.UUID) (int, error)
	CreateBatch(ctx context.Context, members []*Member) ([]*Member, error)
	Delete(ctx context.Context, channelID uuid.UUID, userID uuid.UUID) error
	Get(ctx context.Context, channelID uuid.UUID, userID uuid.UUID) (*Member, error)
	GetBatchByChannelID(ctx context.Context, channelID uuid.UUID) ([]*Member, error)
	GetBatchByChannelIDs(ctx context.Context, channelIDs []uuid.UUID) (map[uuid.UUID][]*Member, error)
	IncrementPeersMentionCountByChannelID(ctx context.Context, channelID uuid.UUID, userID uuid.UUID, incrementAmount int, updatedAt time.Time) ([]*Member, error)
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
	CreateBatch(ctx context.Context, messages []*Message) ([]*Message, error)
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
	Create(ctx context.Context, e *outbox.Event) error
}

type RelationRepository interface {
	DeleteByUserID(ctx context.Context, user1ID uuid.UUID, user2ID uuid.UUID, actorID uuid.UUID) error
	Get(ctx context.Context, user1ID uuid.UUID, user2ID uuid.UUID) (*relation.Relation, error)
	GetForUpdate(ctx context.Context, user1ID uuid.UUID, user2ID uuid.UUID) (*relation.Relation, error)
	HasIncomingBlock(ctx context.Context, actorID uuid.UUID, peerIDs []uuid.UUID) error
	ListFriendsByUserID(ctx context.Context, userID uuid.UUID, limit int) ([]*relation.Relation, error)
	ListIncomingBlocksByUserID(ctx context.Context, userID uuid.UUID, limit int) ([]*relation.Relation, error)
	ListIncomingPendingsByUserID(ctx context.Context, userID uuid.UUID, limit int) ([]*relation.Relation, error)
	ListOutgoingBlocksByUserID(ctx context.Context, userID uuid.UUID, limit int) ([]*relation.Relation, error)
	ListOutgoingPendingsByUserID(ctx context.Context, userID uuid.UUID, limit int) ([]*relation.Relation, error)
	ListTypeByUserID(ctx context.Context, userID uuid.UUID, relType relation.Type, limit int) ([]*relation.Relation, error)
	Save(ctx context.Context, rel *relation.Relation) (*relation.Relation, error)
}

type CachedRelationRepository interface {
	GetFriendIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error)
	GetIncomingBlockIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error)
	GetIncomingPendingIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error)
	GetOutgoingBlockIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error)
	GetOutgoingPendingIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error)
}

type UserRepository interface {
	Availability(ctx context.Context, email *string, username *string) (bool, bool, error)
	Create(ctx context.Context, u *user.User) (*user.User, error)
	Get(ctx context.Context, id uuid.UUID) (*user.User, error)
	GetBatch(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]*user.User, error)
	GetByEmail(ctx context.Context, email string) (*user.User, error)
	ListDeleteScheduled(ctx context.Context, currentTime time.Time, limitVal int) ([]*user.User, error)
	SetDeleteSchedule(ctx context.Context, id uuid.UUID, deleteScheduledAt *time.Time, disabledAt *time.Time, updatedAt time.Time) (*user.User, error)
	SetDisabled(ctx context.Context, id uuid.UUID, disabledAt *time.Time, updatedAt time.Time) (*user.User, error)
	Update(ctx context.Context, u *user.User) (*user.User, error)
	UpdateBatch(ctx context.Context, users []*user.User) ([]*user.User, error)
	UpdateEmail(ctx context.Context, id uuid.UUID, email string, updatedAt time.Time) (*user.User, error)
	UpdatePasswordHash(ctx context.Context, id uuid.UUID, passwordHash string, updatedAt time.Time) (*user.User, error)
	UpdatePhone(ctx context.Context, id uuid.UUID, phone *string, updatedAt time.Time) (*user.User, error)
	UpdatePresence(ctx context.Context, id uuid.UUID, presence *presence.Presence, presenceUntil *time.Time, updatedAt time.Time) (*user.User, error)
	UpdateProfile(ctx context.Context, id uuid.UUID, displayName string, bio *string, avatarURL *string, bannerColor *string, updatedAt time.Time) (*user.User, error)
	UpdateUsername(ctx context.Context, id uuid.UUID, username string, updatedAt time.Time) (*user.User, error)
	Verify(ctx context.Context, id uuid.UUID, verifiedAt *time.Time, updatedAt time.Time) (*user.User, error)
}

type CachedUserRepository interface {
	Get(ctx context.Context, id uuid.UUID) (*user.User, error)
	GetBatch(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]*user.User, error)
}

type TX interface {
	ExecTx(ctx context.Context, fn func(txCtx context.Context) error) error
}
