package channel

import (
	"bonfire-api/internal/fields"
	"context"
)

type MessageRepository interface {
	Create(ctx context.Context, msg *Message) (*Message, error)
	Delete(ctx context.Context, id fields.ID) error
	Get(ctx context.Context, id fields.ID) (*Message, error)
	ListAfterByChannelID(ctx context.Context, channelID fields.ID, afterID fields.ID, limit int32) ([]*Message, error)
	ListAroundByChannelID(ctx context.Context, channelID fields.ID, lastReadMessageID fields.ID, beforeLimit int32, afterLimit int32) ([]*Message, error)
	ListBeforeByChannelID(ctx context.Context, channelID fields.ID, beforeID fields.ID, limit int32) ([]*Message, error)
	ListPinnedByChannelID(ctx context.Context, channelID fields.ID, limit int32) ([]*Message, error)
	UpdateContent(ctx context.Context, id fields.ID, content MessageContent, editedAt fields.Timestamp, updatedAt fields.Timestamp) (*Message, error)
	UpdatePinnedAt(ctx context.Context, id fields.ID, pinnedAt fields.Timestamp, updatedAt fields.Timestamp) (*Message, error)
}

type MessageCache interface {
	Delete(ctx context.Context, key fields.ID) error
	DeleteBatch(ctx context.Context, keys []fields.ID) error
	Get(ctx context.Context, id fields.ID) (*Message, error)
	GetBatch(ctx context.Context, ids []fields.ID) (map[fields.ID]*Message, []fields.ID, error)
	Set(ctx context.Context, ch *Message) error
	SetBatch(ctx context.Context, channels []*Message) error
}

type MessageService struct {
	repo         MessageRepository
	cache        MessageCache
	channelRepo  ChannelRepository
	channelCache ChannelCache
	memberRepo   MemberRepository
	memberCache  MemberCache
}

func NewMessageService(repo MessageRepository, cache MessageCache) *MessageService {
	return &MessageService{repo: repo, cache: cache}
}

// Create message
// -> Update channel last message
// -> Increment batch mention count on members

// Update content

// Update pinned at

// Delete