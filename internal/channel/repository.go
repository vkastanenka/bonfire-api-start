package channel

import (
	"context"

	"bonfire-api/internal/fields"
	"bonfire-api/internal/presence"
	"bonfire-api/internal/user"
)

type ChannelCache interface {
	Get(ctx context.Context, id fields.ID) (*Channel, error)
	Set(ctx context.Context, ch *Channel) error
	Delete(ctx context.Context, id fields.ID) error
	AddMembers(ctx context.Context, channelID fields.ID, members []*Member) error
	CreateGroup(ctx context.Context, ch *Channel, members []*Member) error
	GetBatchMembersByChannelIDs(ctx context.Context, channelIDs []fields.ID) (map[fields.ID][]*Member, []fields.ID, error)
	SetBatchMembers(ctx context.Context, channelMembersMap map[fields.ID][]*Member) error
	InvalidateMembers(ctx context.Context, channelID fields.ID) error
	InvalidateMember(ctx context.Context, channelID fields.ID, userID fields.ID) error
}

type ChannelRepository interface {
	Create(ctx context.Context, ch *Channel) (*Channel, error)
	Delete(ctx context.Context, id fields.ID) error
	Get(ctx context.Context, id fields.ID) (*Channel, error)
	GetBatch(ctx context.Context, ids []fields.ID) (map[fields.ID]*Channel, error)
	GetForUpdate(ctx context.Context, id fields.ID) (*Channel, error)
	UpdateGroup(ctx context.Context, id fields.ID, name ChannelName, iconURL fields.URL, updatedAt fields.Timestamp) (*Channel, error)
	UpdateLastMessage(ctx context.Context, id fields.ID, lastMessageID fields.ID, lastMessageAt fields.Timestamp, updatedAt fields.Timestamp) (*Channel, error)
}

type CachedChannelRepository interface {
	Get(ctx context.Context, id fields.ID) (*Channel, error)
}

type MemberRepository interface {
	CountByChannelID(ctx context.Context, channelID fields.ID) (int, error)
	CreateBatch(ctx context.Context, members []*Member) ([]*Member, error)
	Delete(ctx context.Context, channelID fields.ID, userID fields.ID) error
	Get(ctx context.Context, channelID fields.ID, userID fields.ID) (*Member, error)
	GetBatchByChannelID(ctx context.Context, channelID fields.ID) ([]*Member, error)
	GetBatchByChannelIDs(ctx context.Context, channelIDs []fields.ID) (map[fields.ID][]*Member, error)
	IncrementPeersMentionCountByChannelID(ctx context.Context, channelID fields.ID, userID fields.ID, incrementAmount int, updatedAt fields.Timestamp) error
	ListVisibleByUserID(ctx context.Context, userID fields.ID, limit int) ([]*Member, error)
	Require(ctx context.Context, channelID fields.ID, userID fields.ID) (*Member, error)
	UpdateIsVisible(ctx context.Context, channelID fields.ID, userID fields.ID, isVisible bool, updatedAt fields.Timestamp) (*Member, error)
	UpdateLastReadMessage(ctx context.Context, channelID fields.ID, userID fields.ID, lastReadMessageID fields.ID, lastReadMessageAt fields.Timestamp, updatedAt fields.Timestamp, mentionCount *int) (*Member, error)
	UpdateMutedUntil(ctx context.Context, channelID fields.ID, userID fields.ID, mutedUntil fields.Timestamp, updatedAt fields.Timestamp) (*Member, error)
	UpdatePinnedAt(ctx context.Context, channelID fields.ID, userID fields.ID, pinnedAt fields.Timestamp, updatedAt fields.Timestamp) (*Member, error)
}

type MessageRepository interface {
	CountByChannelID(ctx context.Context, channelID fields.ID) (int, error)
	Create(ctx context.Context, msg *Message) (*Message, error)
	CreateAndMention(ctx context.Context, msg *Message, channelID fields.ID, userID fields.ID, updatedAt fields.Timestamp) (*Message, error)
	CreateBatch(ctx context.Context, messages []*Message) ([]*Message, error)
	CreateBatchAndMention(ctx context.Context, messages []*Message, channelID fields.ID, userID fields.ID, updatedAt fields.Timestamp) ([]*Message, error)
	Delete(ctx context.Context, id fields.ID) error
	Get(ctx context.Context, id fields.ID) (*Message, error)
	ListAfterByChannelID(ctx context.Context, channelID fields.ID, cursorID fields.ID, limit int) ([]*Message, bool, error)
	ListAroundByChannelID(ctx context.Context, channelID fields.ID, cursorID fields.ID, beforeLimit int, afterLimit int) ([]*Message, bool, bool, error)
	ListBeforeByChannelID(ctx context.Context, channelID fields.ID, cursorID fields.ID, limit int) ([]*Message, bool, error)
	ListPinnedByChannelID(ctx context.Context, channelID fields.ID, cursorID fields.ID, cursorPinnedAt fields.Timestamp, limit int) ([]*Message, bool, error)
	UpdateContent(ctx context.Context, id fields.ID, content MessageContent, editedAt fields.Timestamp, updatedAt fields.Timestamp) (*Message, error)
	UpdatePinnedAt(ctx context.Context, id fields.ID, pinnedAt fields.Timestamp, updatedAt fields.Timestamp) (*Message, error)
}

type ReactionRepository interface {
	CountByEmoji(ctx context.Context, messageID fields.ID, emoji ReactionEmoji) (int, error)
	Create(ctx context.Context, rx *Reaction) (*Reaction, error)
	Delete(ctx context.Context, messageID fields.ID, userID fields.ID, emoji ReactionEmoji) error
	Get(ctx context.Context, messageID fields.ID, userID fields.ID, emoji ReactionEmoji) (*Reaction, error)
	GetBatchSummaryByMessageIDs(ctx context.Context, userID fields.ID, messageIDs []fields.ID) (map[fields.ID]*ReactionSummary, error)
}

type OutboxRepository interface {
	Publish(ctx context.Context, eventType string, payload any, now fields.Timestamp) error
}

type RelationRepository interface {
	HasIncomingBlock(ctx context.Context, actorID fields.ID, peerIDs []fields.ID) error
}

type PresenceCache interface {
	GetPresence(ctx context.Context, userID fields.ID) (presence.Presence, error)
	GetBatchPresence(ctx context.Context, userIDs []fields.ID) (map[fields.ID]presence.Presence, error)
	SetPresence(ctx context.Context, userID fields.ID, p presence.Presence) error
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
	RemoveChannelID(ctx context.Context, userID fields.ID, channelID fields.ID) error
	RemoveFriendID(ctx context.Context, userID fields.ID, friendID fields.ID) error
	RemoveFriendPair(ctx context.Context, userA fields.ID, userB fields.ID) error
	Set(ctx context.Context, usr *user.User) error
	SetBatch(ctx context.Context, users map[fields.ID]*user.User) error
	SetChannelIDs(ctx context.Context, userID fields.ID, channelIDs []fields.ID) error
	SetFriendIDs(ctx context.Context, userID fields.ID, friendIDs []fields.ID) error
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

type Broadcaster interface {
	BroadcastToUser(ctx context.Context, userID fields.ID, excludeSessionIDs []fields.ID, eventType string, payload interface{}) error
	BroadcastToUsers(ctx context.Context, recipientIDs []fields.ID, excludeSessionIDs []fields.ID, eventType string, payload interface{}) error
}
