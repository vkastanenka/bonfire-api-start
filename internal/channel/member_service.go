package channel

import (
	"bonfire-api/internal/fields"
	"context"
)

type MemberRepository interface {
	CreateBatch(ctx context.Context, members []*Member) ([]*Member, error)
	Delete(ctx context.Context, channelID fields.ID, userID fields.ID) error
	Get(ctx context.Context, channelID fields.ID, userID fields.ID) (*Member, error)
	GetBatchByChannelIDs(ctx context.Context, channelIDs []fields.ID) ([]*Member, error)
	IncrementBatchMentionCount(ctx context.Context, channelID fields.ID, userIDs []fields.ID, updatedAt fields.Timestamp) error
	ListVisibleByUserID(ctx context.Context, userID fields.ID, limit int32) ([]*Member, error)
	UpdateIsVisible(ctx context.Context, channelID fields.ID, userID fields.ID, isVisible bool, updatedAt fields.Timestamp) (*Member, error)
	UpdateLastReadMessage(ctx context.Context, channelID fields.ID, userID fields.ID, lastReadMessageID fields.ID, lastReadMessageAt fields.Timestamp, updatedAt fields.Timestamp, mentionCount *int32) (*Member, error)
	UpdateMutedUntil(ctx context.Context, channelID fields.ID, userID fields.ID, mutedUntil fields.Timestamp, updatedAt fields.Timestamp) (*Member, error)
	UpdatePinnedAt(ctx context.Context, channelID fields.ID, userID fields.ID, pinnedAt fields.Timestamp, updatedAt fields.Timestamp) (*Member, error)
}

type MemberCache interface {
	Delete(ctx context.Context, key fields.ID) error
	DeleteBatch(ctx context.Context, keys []fields.ID) error
	Get(ctx context.Context, id fields.ID) (*Member, error)
	GetBatch(ctx context.Context, ids []fields.ID) (map[fields.ID]*Member, []fields.ID, error)
	Set(ctx context.Context, ch *Member) error
	SetBatch(ctx context.Context, members []*Member) error
}

type MemberService struct {
	repo  MemberRepository
	cache MemberCache
}

func NewMemberService(repo MemberRepository, cache MemberCache) *MemberService {
	return &MemberService{repo: repo, cache: cache}
}

// Create batch

// Update is visible

// Update last read message

// Update muted until

// Update pinned at

// Delete
