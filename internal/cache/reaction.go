package cache

import (
	"context"
	"fmt"
	"time"

	"bonfire-api/internal/channel"
	"bonfire-api/internal/fields"
	"bonfire-api/internal/redis"

	"github.com/google/uuid"
	redisdriver "github.com/redis/go-redis/v9"
)

type ReactionKeyIDs struct {
	MessageID fields.ID
	UserID    fields.ID
	Emoji     channel.ReactionEmoji
}

// reactionKey converts a composite ReactionKeyIDs struct into a unique Redis key.
func reactionKey(k ReactionKeyIDs) string {
	return fmt.Sprintf("reaction:%s:%s:%s", k.MessageID.String(), k.UserID.String(), k.Emoji.String())
}

type Reaction struct {
	MessageID uuid.UUID `json:"message_id"`
	UserID    uuid.UUID `json:"user_id"`
	Emoji     string    `json:"emoji"`
	CreatedAt time.Time `json:"created_at"`
}

func (r Reaction) ToDomain() (*channel.Reaction, error) {
	messageID, err := fields.ParseRequiredID("message_id", r.MessageID)
	if err != nil {
		return nil, err
	}

	userID, err := fields.ParseRequiredID("user_id", r.UserID)
	if err != nil {
		return nil, err
	}

	reactionEmoji, err := channel.ParseReactionEmoji(r.Emoji)
	if err != nil {
		return nil, err
	}

	return channel.ParseReaction(
		messageID,
		userID,
		reactionEmoji,
		fields.NewTimestamp(r.CreatedAt),
	), nil
}

func ParseReaction(r *channel.Reaction) Reaction {
	return Reaction{
		MessageID: r.MessageID().UUID(),
		UserID:    r.UserID().UUID(),
		Emoji:     r.Emoji().String(),
		CreatedAt: r.CreatedAt().Time(),
	}
}

type ReactionCache struct {
	store *JSONCache[ReactionKeyIDs, Reaction]
	ttl   time.Duration
}

func NewReactionCache(client redisdriver.Cmdable, ttl time.Duration) *ReactionCache {
	return &ReactionCache{
		store: NewJSONCache[ReactionKeyIDs, Reaction](client, redis.ScopeReaction, reactionKey),
		ttl:   ttl,
	}
}

func (c *ReactionCache) Get(
	ctx context.Context,
	messageID, userID fields.ID,
	emoji channel.ReactionEmoji,
) (*channel.Reaction, error) {
	key := ReactionKeyIDs{
		MessageID: messageID,
		UserID:    userID,
		Emoji:     emoji,
	}

	dto, err := c.store.Get(ctx, key)
	if err != nil || dto == nil {
		return nil, err
	}

	return dto.ToDomain()
}

func (c *ReactionCache) GetBatch(
	ctx context.Context,
	keys []ReactionKeyIDs,
) (map[ReactionKeyIDs]*channel.Reaction, []ReactionKeyIDs, error) {
	dtos, missing, err := c.store.GetBatch(ctx, keys)
	if err != nil {
		return nil, nil, err
	}

	found := make(map[ReactionKeyIDs]*channel.Reaction, len(dtos))
	for key, dto := range dtos {
		if dto == nil {
			missing = append(missing, key)
			continue
		}

		react, err := dto.ToDomain()
		if err != nil {
			missing = append(missing, key)
			continue
		}
		found[key] = react
	}

	return found, missing, nil
}

func (c *ReactionCache) Set(ctx context.Context, r *channel.Reaction) error {
	if r == nil {
		return nil
	}

	key := ReactionKeyIDs{
		MessageID: r.MessageID(),
		UserID:    r.UserID(),
		Emoji:     r.Emoji(),
	}

	return c.store.Set(ctx, key, ParseReaction(r), c.ttl)
}

func (c *ReactionCache) SetBatch(ctx context.Context, reactions []*channel.Reaction) error {
	dtos := make(map[ReactionKeyIDs]Reaction, len(reactions))
	for _, r := range reactions {
		if r == nil {
			continue
		}

		key := ReactionKeyIDs{
			MessageID: r.MessageID(),
			UserID:    r.UserID(),
			Emoji:     r.Emoji(),
		}
		dtos[key] = ParseReaction(r)
	}

	return c.store.SetBatch(ctx, dtos, c.ttl)
}

func (c *ReactionCache) Invalidate(
	ctx context.Context,
	messageID, userID fields.ID,
	emoji channel.ReactionEmoji,
) error {
	key := ReactionKeyIDs{
		MessageID: messageID,
		UserID:    userID,
		Emoji:     emoji,
	}

	return c.store.Invalidate(ctx, key)
}

func (c *ReactionCache) InvalidateBatch(ctx context.Context, keys []ReactionKeyIDs) error {
	return c.store.InvalidateBatch(ctx, keys)
}
