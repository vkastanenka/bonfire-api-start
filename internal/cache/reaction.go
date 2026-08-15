package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"bonfire-api/internal/channel"
	"bonfire-api/internal/fields"
	"bonfire-api/internal/redis"

	"github.com/google/uuid"
)

func reactionKey(messageID, userID fields.ID, emoji channel.ReactionEmoji) string {
	return fmt.Sprintf("reaction:%s:%s:%s", messageID.String(), userID.String(), emoji.String())
}

type ReactionKeyIDs struct {
	MessageID fields.ID
	UserID    fields.ID
	Emoji     channel.ReactionEmoji
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
	store *redis.Store
	ttl   time.Duration
}

func NewReactionCache(store *redis.Store, ttl time.Duration) *ReactionCache {
	return &ReactionCache{
		store: store.WithScope(redis.ScopeMessageReaction),
		ttl:   ttl,
	}
}

func (c *ReactionCache) Get(
	ctx context.Context,
	messageID, userID fields.ID,
	emoji channel.ReactionEmoji,
) (*channel.Reaction, error) {
	k := reactionKey(messageID, userID, emoji)

	var raw string
	err := c.store.Get(ctx, k, &raw)
	if err != nil {
		return nil, c.store.Err(err)
	}
	if raw == "" {
		return nil, nil
	}

	var cached Reaction
	if err := json.Unmarshal([]byte(raw), &cached); err != nil {
		return nil, err
	}

	return cached.ToDomain()
}

func (c *ReactionCache) GetBatch(
	ctx context.Context,
	keys []ReactionKeyIDs,
) (map[ReactionKeyIDs]*channel.Reaction, []ReactionKeyIDs, error) {
	if len(keys) == 0 {
		return map[ReactionKeyIDs]*channel.Reaction{}, nil, nil
	}

	redisKeys := make([]string, len(keys))
	for i, k := range keys {
		redisKeys[i] = reactionKey(k.MessageID, k.UserID, k.Emoji)
	}

	rawValues, err := c.store.MGet(ctx, redisKeys...)
	if err != nil {
		return nil, nil, c.store.Err(err)
	}

	found := make(map[ReactionKeyIDs]*channel.Reaction, len(keys))
	var missing []ReactionKeyIDs

	for i, raw := range rawValues {
		keyIDs := keys[i]

		if raw == nil || raw == "" {
			missing = append(missing, keyIDs)
			continue
		}

		rawStr, ok := raw.(string)
		if !ok {
			missing = append(missing, keyIDs)
			continue
		}

		var cached Reaction
		if err := json.Unmarshal([]byte(rawStr), &cached); err != nil {
			missing = append(missing, keyIDs)
			continue
		}

		react, err := cached.ToDomain()
		if err != nil {
			missing = append(missing, keyIDs)
			continue
		}

		found[keyIDs] = react
	}

	return found, missing, nil
}

func (c *ReactionCache) Set(ctx context.Context, r *channel.Reaction) error {
	k := reactionKey(r.MessageID(), r.UserID(), r.Emoji())
	dto := ParseReaction(r)

	if err := c.store.Set(ctx, k, dto, c.ttl); err != nil {
		return c.store.Err(err)
	}

	return nil
}

func (c *ReactionCache) SetBatch(ctx context.Context, reactions []*channel.Reaction) error {
	if len(reactions) == 0 {
		return nil
	}

	return c.store.ExecPipeline(ctx, func(pipeCtx context.Context) error {
		for _, r := range reactions {
			k := reactionKey(r.MessageID(), r.UserID(), r.Emoji())
			dto := ParseReaction(r)

			if err := c.store.Set(pipeCtx, k, dto, c.ttl); err != nil {
				return err
			}
		}
		return nil
	})
}

func (c *ReactionCache) Invalidate(
	ctx context.Context,
	messageID, userID fields.ID,
	emoji channel.ReactionEmoji,
) error {
	return c.store.Delete(ctx, reactionKey(messageID, userID, emoji))
}

func (c *ReactionCache) InvalidateBatch(ctx context.Context, keys []ReactionKeyIDs) error {
	if len(keys) == 0 {
		return nil
	}

	return c.store.ExecPipeline(ctx, func(pipeCtx context.Context) error {
		for _, k := range keys {
			if err := c.store.Delete(pipeCtx, reactionKey(k.MessageID, k.UserID, k.Emoji)); err != nil {
				return err
			}
		}
		return nil
	})
}
