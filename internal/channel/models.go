package channel

import (
	"context"

	"bonfire-api/internal/fields"
	"bonfire-api/internal/outbox"
	"bonfire-api/internal/presence"
	"bonfire-api/internal/user"
)

type ChannelRepository interface {
	Create(ctx context.Context, ch *Channel) (*Channel, error)
	Delete(ctx context.Context, id fields.ID) error
	Get(ctx context.Context, id fields.ID) (*Channel, error)
	GetBatch(ctx context.Context, ids []fields.ID) (map[fields.ID]*Channel, error)
	GetForUpdate(ctx context.Context, id fields.ID) (*Channel, error)
	UpdateGroup(ctx context.Context, id fields.ID, name ChannelName, iconURL fields.URL, updatedAt fields.Timestamp) (*Channel, error)
	UpdateLastMessage(ctx context.Context, id fields.ID, lastMessageID fields.ID, lastMessageAt fields.Timestamp, updatedAt fields.Timestamp) (*Channel, error)
}

type ChannelCache interface {
	Delete(ctx context.Context, key fields.ID) error
	DeleteBatch(ctx context.Context, keys []fields.ID) error
	Get(ctx context.Context, id fields.ID) (*Channel, error)
	GetBatch(ctx context.Context, ids []fields.ID) (map[fields.ID]*Channel, []fields.ID, error)
	Set(ctx context.Context, ch *Channel) error
	SetBatch(ctx context.Context, channels []*Channel) error
}

type MemberRepository interface {
	CountByChannel(ctx context.Context, channelID fields.ID) (int64, error)
	CreateBatch(ctx context.Context, members []*Member) ([]*Member, error)
	Delete(ctx context.Context, channelID fields.ID, userID fields.ID) error
	Get(ctx context.Context, channelID fields.ID, userID fields.ID) (*Member, error)
	GetBatchByChannelIDs(ctx context.Context, channelIDs []fields.ID) (map[fields.ID][]*Member, error)
	IncrementPeersMentionCountByChannelID(ctx context.Context, channelID fields.ID, userID fields.ID, updatedAt fields.Timestamp) error
	ListVisibleByUserID(ctx context.Context, userID fields.ID, limit int32) ([]*Member, error)
	UpdateIsVisible(ctx context.Context, channelID fields.ID, userID fields.ID, isVisible bool, updatedAt fields.Timestamp) (*Member, error)
	UpdateLastReadMessage(ctx context.Context, channelID fields.ID, userID fields.ID, lastReadMessageID fields.ID, lastReadMessageAt fields.Timestamp, updatedAt fields.Timestamp, mentionCount *int32) (*Member, error)
	UpdateMutedUntil(ctx context.Context, channelID fields.ID, userID fields.ID, mutedUntil fields.Timestamp, updatedAt fields.Timestamp) (*Member, error)
	UpdatePinnedAt(ctx context.Context, channelID fields.ID, userID fields.ID, pinnedAt fields.Timestamp, updatedAt fields.Timestamp) (*Member, error)
}

type MemberCache interface {
	Delete(ctx context.Context, channelID fields.ID, userID fields.ID) error
	Get(ctx context.Context, channelID fields.ID, userID fields.ID) (*Member, error)
	GetBatchByChannelIDs(ctx context.Context, channelIDs []fields.ID) (found map[fields.ID][]*Member, missingChannelIDs []fields.ID, err error)
	GetVisibleByUserID(ctx context.Context, userID fields.ID) ([]*Member, []fields.ID, error)
	Set(ctx context.Context, member *Member) error
	SetBatch(ctx context.Context, members []*Member) error
}

type MessageRepository interface {
	Create(ctx context.Context, msg *Message) (*Message, error)
	CreateBatch(ctx context.Context, messages []*Message) ([]*Message, error)
	Delete(ctx context.Context, id fields.ID) error
	Get(ctx context.Context, id fields.ID) (*Message, error)
	ListAfterByChannelID(ctx context.Context, channelID fields.ID, cursorID fields.ID, limit int32) ([]*Message, error)
	ListAroundByChannelID(ctx context.Context, channelID fields.ID, lastReadMessageID fields.ID, beforeLimit int32, afterLimit int32) ([]*Message, error)
	ListBeforeByChannelID(ctx context.Context, channelID fields.ID, cursorID fields.ID, limit int32) ([]*Message, error)
	ListPinnedByChannelID(ctx context.Context, channelID fields.ID, cursorID fields.ID, cursorPinnedAt fields.Timestamp, limit int32) ([]*Message, error)
	UpdateContent(ctx context.Context, id fields.ID, content MessageContent, editedAt fields.Timestamp, updatedAt fields.Timestamp) (*Message, error)
	UpdatePinnedAt(ctx context.Context, id fields.ID, pinnedAt fields.Timestamp, updatedAt fields.Timestamp) (*Message, error)
}

type MessageCache interface {
	Delete(ctx context.Context, channelID fields.ID, messageID fields.ID) error
	DeleteBatch(ctx context.Context, channelID fields.ID, keys []fields.ID) error
	Get(ctx context.Context, id fields.ID) (*Message, error)
	GetBatch(ctx context.Context, ids []fields.ID) (map[fields.ID]*Message, []fields.ID, error)
	IsTimelineComplete(ctx context.Context, channelID fields.ID) bool
	ListAfterByChannelID(ctx context.Context, channelID fields.ID, cursorID fields.ID, limit int32) ([]*Message, error)
	ListAroundByChannelID(ctx context.Context, channelID fields.ID, anchorMessageID fields.ID, beforeLimit int32, afterLimit int32) ([]*Message, error)
	ListBeforeByChannelID(ctx context.Context, channelID fields.ID, cursorID fields.ID, limit int32) ([]*Message, error)
	Set(ctx context.Context, msg *Message) error
	SetBatch(ctx context.Context, messages []*Message) error
	SetTimelineComplete(ctx context.Context, channelID fields.ID) error
}

type ReactionRepository interface {
	CountByEmoji(ctx context.Context, messageID fields.ID, emoji ReactionEmoji) (int, error)
	Create(ctx context.Context, rx *Reaction) (*Reaction, error)
	Delete(ctx context.Context, messageID fields.ID, userID fields.ID, emoji ReactionEmoji) error
	Get(ctx context.Context, messageID fields.ID, userID fields.ID, emoji ReactionEmoji) (*Reaction, error)
	GetBatchSummaryByMessageIDs(ctx context.Context, userID fields.ID, messageIDs []fields.ID) (map[fields.ID]*ReactionSummary, error)
}

type ReactionCache interface {
	DecrementEmoji(ctx context.Context, messageID fields.ID, _ string) error
	Delete(ctx context.Context, messageID fields.ID) error
	Get(ctx context.Context, messageID fields.ID) (map[string]int, bool, error)
	GetBatch(ctx context.Context, messageIDs []fields.ID) (hits map[fields.ID]map[string]int, misses []fields.ID, err error)
	IncrementEmoji(ctx context.Context, messageID fields.ID, _ string) error
	Set(ctx context.Context, messageID fields.ID, counts map[string]int) error
	SetBatch(ctx context.Context, countsMap map[fields.ID]map[string]int, missedIDs []fields.ID) error
}

type OutboxRepository interface {
	Publish(ctx context.Context, variant string, payload any) (*outbox.Event, error)
}

type RelationRepository interface {
	HasIncomingBlock(ctx context.Context, actorID fields.ID, peerIDs []fields.ID) (bool, error)
}

type UserRepository interface {
	Get(ctx context.Context, id fields.ID) (*user.User, error)
	GetBatch(ctx context.Context, ids []fields.ID) (map[fields.ID]*user.User, error)
}

type UserCache interface {
	SetBatch(ctx context.Context, users []*user.User) error
}

type PresenceCache interface {
	GetBatch(ctx context.Context, userIDs []fields.ID) (map[fields.ID]presence.Presence, error)
}

type TX interface {
	ExecTx(ctx context.Context, fn func(txCtx context.Context) error) error
}
