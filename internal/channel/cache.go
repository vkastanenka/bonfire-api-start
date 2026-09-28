package channel

import (
	"bonfire-api/internal/presence"
	"bonfire-api/internal/user"
	"context"

	"github.com/google/uuid"
)

type ChannelCache interface {
	CreateGroup(ctx context.Context, ch *Channel, members []*Member) error
	Delete(ctx context.Context, id uuid.UUID) error
	DeleteGroup(ctx context.Context, channelID uuid.UUID, memberIDs []uuid.UUID) error
	Get(ctx context.Context, id uuid.UUID) (*Channel, error)
	GetBatch(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]*Channel, []uuid.UUID, error)
	GetUserChannelIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, bool, error)
	RemoveUserChannelID(ctx context.Context, userID uuid.UUID, channelID uuid.UUID) error
	Set(ctx context.Context, ch *Channel) error
	SetBatch(ctx context.Context, channels map[uuid.UUID]*Channel) error
	SetUserChannelIDs(ctx context.Context, userID uuid.UUID, members []*Member) error
}

type MemberCache interface {
	Add(ctx context.Context, channelID uuid.UUID, members []*Member) error
	Get(ctx context.Context, channelID uuid.UUID, userID uuid.UUID) (*Member, error)
	GetBatchByChannelIDs(ctx context.Context, channelIDs []uuid.UUID) (map[uuid.UUID][]*Member, []uuid.UUID, error)
	Invalidate(ctx context.Context, channelID uuid.UUID, userID uuid.UUID) error
	InvalidateBatch(ctx context.Context, channelID uuid.UUID, userIDs []uuid.UUID) error
	InvalidateChannel(ctx context.Context, channelID uuid.UUID) error
	Remove(ctx context.Context, channelID uuid.UUID, userID uuid.UUID) error
	SetBatchByChannelIDs(ctx context.Context, channelMembersMap map[uuid.UUID][]*Member) error
	SetUserMembers(ctx context.Context, userID uuid.UUID, members []*Member) error
}

type MessageCache interface {
	Delete(ctx context.Context, channelID uuid.UUID, msgID uuid.UUID) error
	Get(ctx context.Context, id uuid.UUID) (*Message, error)
	GetAfterByChannelID(ctx context.Context, channelID uuid.UUID, cursorID uuid.UUID, limit int) ([]*Message, bool, bool, error)
	GetAroundByChannelID(ctx context.Context, channelID uuid.UUID, cursorID uuid.UUID, beforeLimit int, afterLimit int) ([]*Message, bool, bool, bool, error)
	GetBeforeByChannelID(ctx context.Context, channelID uuid.UUID, cursorID uuid.UUID, limit int) ([]*Message, bool, bool, error)
	GetRecentByChannelID(ctx context.Context, channelID uuid.UUID, limit int) ([]*Message, bool, error)
	Set(ctx context.Context, msg *Message) error
	SetBatch(ctx context.Context, channelID uuid.UUID, messages []*Message) error
}

type PresenceCache interface {
	Delete(ctx context.Context, id uuid.UUID) error
	Get(ctx context.Context, userID uuid.UUID) (presence.Presence, error)
	GetBatch(ctx context.Context, userIDs []uuid.UUID) (map[uuid.UUID]presence.Presence, error)
	Set(ctx context.Context, userID uuid.UUID, p presence.Presence) error
}

type UserCache interface {
	Delete(ctx context.Context, id uuid.UUID) error
	DeleteBatch(ctx context.Context, ids []uuid.UUID) error
	Get(ctx context.Context, id uuid.UUID) (*user.User, error)
	GetBatch(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]*user.User, []uuid.UUID, error)
	Set(ctx context.Context, usr *user.User) error
	SetBatch(ctx context.Context, users map[uuid.UUID]*user.User) error
}
