package channel

import (
	"bonfire-api/internal/fields"
	"context"
	"time"

	"github.com/google/uuid"
)

type ReactionRepository interface {
	Create(ctx context.Context, rx *Reaction) (*Reaction, error)
	Delete(ctx context.Context, messageID fields.ID, userID fields.ID, emoji ReactionEmoji) error
	GetBatchByMessageIDs(ctx context.Context, messageIDs []fields.ID) ([]*Reaction, error)
}

type ReactionCache interface {
	Delete(ctx context.Context, key fields.ID) error
	DeleteBatch(ctx context.Context, keys []fields.ID) error
	Get(ctx context.Context, id fields.ID) (*Message, error)
	GetBatch(ctx context.Context, ids []fields.ID) (map[fields.ID]*Message, []fields.ID, error)
	Set(ctx context.Context, ch *Message) error
	SetBatch(ctx context.Context, channels []*Message) error
}

type ReactionService struct {
	repo  ReactionRepository
	cache ReactionCache
}

func NewReactionService(repo ReactionRepository, cache ReactionCache) *ReactionService {
	return &ReactionService{repo: repo, cache: cache}
}

type ReactionCreateParams struct {
	messageID uuid.UUID
	userID    uuid.UUID
	emoji     string
}

// Get fetches channel details if the actor is a member.
func (s *ReactionService) Create(ctx context.Context, p ReactionCreateParams) (*Reaction, error) {
	messageID, err := fields.ParseRequiredID("message_id", p.messageID)
	if err != nil {
		return nil, err
	}

	userID, err := fields.ParseRequiredID("user_id", p.userID)
	if err != nil {
		return nil, err
	}

	emoji, err := ParseReactionEmoji(p.emoji)
	if err != nil {
		return nil, err
	}

	now := fields.NewTimestamp(time.Now())

	reaction := ParseReaction(messageID, userID, emoji, now)

	ch, err := s.repo.Create(ctx, reaction)
	if err != nil {
		return nil, err
	}

	// s.cache.Delete(ctx, messageID, userID, emoji)

	return ch, nil
}

// Delete
