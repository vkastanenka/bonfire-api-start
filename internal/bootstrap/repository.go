package bootstrap

import (
	"context"

	"bonfire-api/internal/channel"
	"bonfire-api/internal/fields"
	"bonfire-api/internal/user"
)

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

type MessageRepository interface {
	CountByChannelID(ctx context.Context, channelID fields.ID) (int, error)
	Create(ctx context.Context, msg *channel.Message) (*channel.Message, error)
	CreateAndMention(ctx context.Context, msg *channel.Message, channelID fields.ID, userID fields.ID, updatedAt fields.Timestamp) (*channel.Message, error)
	CreateBatch(ctx context.Context, messages []*channel.Message) ([]*channel.Message, error)
	CreateBatchAndMention(ctx context.Context, messages []*channel.Message, channelID fields.ID, userID fields.ID, updatedAt fields.Timestamp) ([]*channel.Message, error)
	Delete(ctx context.Context, id fields.ID) error
	Get(ctx context.Context, id fields.ID) (*channel.Message, error)
	ListAfterByChannelID(ctx context.Context, channelID fields.ID, cursorID fields.ID, limit int) ([]*channel.Message, bool, error)
	ListAroundByChannelID(ctx context.Context, channelID fields.ID, cursorID fields.ID, beforeLimit int, afterLimit int) ([]*channel.Message, bool, bool, error)
	ListBeforeByChannelID(ctx context.Context, channelID fields.ID, cursorID fields.ID, limit int) ([]*channel.Message, bool, error)
	ListPinnedByChannelID(ctx context.Context, channelID fields.ID, cursorID fields.ID, cursorPinnedAt fields.Timestamp, limit int) ([]*channel.Message, bool, error)
	UpdateContent(ctx context.Context, id fields.ID, content channel.MessageContent, editedAt fields.Timestamp, updatedAt fields.Timestamp) (*channel.Message, error)
	UpdatePinnedAt(ctx context.Context, id fields.ID, pinnedAt fields.Timestamp, updatedAt fields.Timestamp) (*channel.Message, error)
}

type CachedMessageRepository interface {
	Get(ctx context.Context, id fields.ID) (*channel.Message, error)
	ListAfterByChannelID(ctx context.Context, channelID fields.ID, cursorID fields.ID, limit int) ([]*channel.Message, bool, error)
	ListAroundByChannelID(ctx context.Context, channelID fields.ID, cursorID fields.ID, beforeLimit int, afterLimit int) ([]*channel.Message, bool, bool, error)
	ListBeforeByChannelID(ctx context.Context, channelID fields.ID, cursorID fields.ID, limit int) ([]*channel.Message, bool, error)
}

type ReactionRepository interface {
	CountByEmoji(ctx context.Context, messageID fields.ID, emoji channel.ReactionEmoji) (int, error)
	Create(ctx context.Context, rx *channel.Reaction) (*channel.Reaction, error)
	Delete(ctx context.Context, messageID fields.ID, userID fields.ID, emoji channel.ReactionEmoji) error
	Get(ctx context.Context, messageID fields.ID, userID fields.ID, emoji channel.ReactionEmoji) (*channel.Reaction, error)
	GetBatchSummaryByMessageIDs(ctx context.Context, userID fields.ID, messageIDs []fields.ID) (map[fields.ID]*channel.ReactionSummary, error)
}

type OutboxRepository interface {
	Publish(ctx context.Context, eventType string, payload any, now fields.Timestamp) error
}

type RelationRepository interface {
	HasIncomingBlock(ctx context.Context, actorID fields.ID, peerIDs []fields.ID) error
}

type CachedRelationRepository interface {
	GetBlockedByIDs(ctx context.Context, userID fields.ID) ([]fields.ID, error)
	GetBlocklistIDs(ctx context.Context, userID fields.ID) ([]fields.ID, error)
	GetFriends(ctx context.Context, userID fields.ID) (map[fields.ID]fields.ID, error)
	GetPendingIDs(ctx context.Context, userID fields.ID) ([]fields.ID, error)
}

type UserRepository interface {
	Get(ctx context.Context, id fields.ID) (*user.User, error)
	GetBatch(ctx context.Context, ids []fields.ID) (map[fields.ID]*user.User, error)
}

type CachedUserRepository interface {
	Get(ctx context.Context, id fields.ID) (*user.User, error)
	GetBatch(ctx context.Context, ids []fields.ID) (map[fields.ID]*user.User, error)
	GetBatchValid(ctx context.Context, ids []fields.ID) (map[fields.ID]*user.User, error)
	GetValid(ctx context.Context, id fields.ID) (*user.User, error)
}

type TX interface {
	ExecTx(ctx context.Context, fn func(txCtx context.Context) error) error
}
