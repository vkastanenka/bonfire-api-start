package repository

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"bonfire-api/internal/channel"
	"bonfire-api/internal/db"
	"bonfire-api/internal/errs"
	"bonfire-api/internal/fields"

	"github.com/google/uuid"
	"golang.org/x/sync/singleflight"
)

type ReactionCache interface {
	DecrementEmoji(ctx context.Context, messageID fields.ID, _ string) error
	Delete(ctx context.Context, messageID fields.ID) error
	Get(ctx context.Context, messageID fields.ID) (map[string]int, bool, error)
	GetBatch(ctx context.Context, messageIDs []fields.ID) (hits map[fields.ID]map[string]int, misses []fields.ID, err error)
	IncrementEmoji(ctx context.Context, messageID fields.ID, _ string) error
	Set(ctx context.Context, messageID fields.ID, counts map[string]int) error
	SetBatch(ctx context.Context, countsMap map[fields.ID]map[string]int, missedIDs []fields.ID) error
}

type ReactionRepository struct {
	store *db.Store
	cache ReactionCache
	sf    singleflight.Group
}

func NewReactionRepository(store *db.Store, cache ReactionCache) *ReactionRepository {
	return &ReactionRepository{
		store: store.WithEntity(db.EntityMessageReaction),
		cache: cache,
	}
}

func (r *ReactionRepository) Create(ctx context.Context, rx *channel.Reaction) (*channel.Reaction, error) {
	row, err := r.store.ReactionCreate(ctx, db.ReactionCreateParams{
		MessageID: db.ToUUID(rx.MessageID().UUID()),
		UserID:    db.ToUUID(rx.UserID().UUID()),
		Emoji:     rx.Emoji().String(),
		CreatedAt: db.ToTimestamptz(rx.CreatedAt().Time()),
	})
	if err != nil {
		return nil, r.store.Err(err)
	}

	return reactionFromRow(row)
}

func (r *ReactionRepository) Get(
	ctx context.Context,
	messageID, userID fields.ID,
	emoji channel.ReactionEmoji,
) (*channel.Reaction, error) {
	emojiStr := emoji.String()

	// 1. Fast path: Check Redis aggregate counts
	counts, hit, err := r.cache.Get(ctx, messageID)
	if err == nil && hit {
		// If total count is 0 or missing from Redis, this user cannot have reacted
		if cnt, exists := counts[emojiStr]; !exists || cnt == 0 {
			return nil, errs.NotFound("reaction not found")
		}
	}

	// 2. Cache Miss: Backfill message aggregate counts if absent from Redis
	if !hit {
		dbRows, dbErr := r.store.ReactionGetBatchSummaryByMessageIDs(ctx, db.ToUUIDs([]uuid.UUID{messageID.UUID()}))
		if dbErr != nil {
			slog.WarnContext(ctx, "failed to fetch DB summary for backfill", "error", dbErr)
		} else {
			summary := make(map[string]int, len(dbRows))
			for _, row := range dbRows {
				summary[row.Emoji] = int(row.Count)
			}
			if setErr := r.cache.Set(ctx, messageID, summary); setErr != nil {
				slog.WarnContext(ctx, "failed to set reaction cache", "error", setErr)
			}
		}
	}

	// 3. Direct DB Query for this specific user's reaction
	row, dbErr := r.store.ReactionGet(ctx, db.ReactionGetParams{
		MessageID: db.ToUUID(messageID.UUID()),
		UserID:    db.ToUUID(userID.UUID()),
		Emoji:     emojiStr,
	})
	if dbErr != nil {
		return nil, r.store.Err(dbErr)
	}

	return reactionFromRow(row)
}

func (r *ReactionRepository) ReactionGetBatchSummaryByMessageIDs(
	ctx context.Context,
	userID fields.ID,
	messageIDs []fields.ID,
) (map[fields.ID]*channel.ReactionSummary, error) {
	// 1. Read aggregate counts from Redis Cache
	cachedCounts, missedIDs, err := r.cache.GetBatch(ctx, messageIDs)
	if err != nil {
		slog.WarnContext(ctx, "reaction cache read failed, falling back to DB", "error", err)
		missedIDs = messageIDs
		cachedCounts = make(map[fields.ID]map[string]int, len(messageIDs))
	}

	// 2. Fetch aggregate counts from DB for cache misses
	if len(missedIDs) > 0 {
		uuidMisses := make([]uuid.UUID, len(missedIDs))
		for i, id := range missedIDs {
			uuidMisses[i] = id.UUID()
		}

		dbRows, err := r.store.ReactionGetBatchSummaryByMessageIDs(ctx, db.ToUUIDs(uuidMisses))
		if err != nil {
			return nil, r.store.Err(err)
		}

		dbCounts := make(map[fields.ID]map[string]int, len(missedIDs))
		for _, id := range missedIDs {
			dbCounts[id] = make(map[string]int)
		}

		for _, row := range dbRows {
			rawMsgID := db.FromUUID[uuid.UUID](row.MessageID)
			msgID, parseErr := fields.ParseRequiredID("id", rawMsgID)
			if parseErr != nil {
				return nil, parseErr
			}
			dbCounts[msgID][row.Emoji] = int(row.Count)
		}

		// Asynchronously backfill cache to prevent blocking response
		go func(ids []fields.ID, counts map[fields.ID]map[string]int) {
			asyncCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			if cacheErr := r.cache.SetBatch(asyncCtx, counts, ids); cacheErr != nil {
				slog.WarnContext(asyncCtx, "failed to backfill reaction cache", "error", cacheErr)
			}
		}(missedIDs, dbCounts)

		for msgID, counts := range dbCounts {
			cachedCounts[msgID] = counts
		}
	}

	// 3. Fetch user's personal reactions
	userReactions := make(map[fields.ID]map[string]bool)
	if !userID.IsZero() {
		uuidMsgs := make([]uuid.UUID, len(messageIDs))
		for i, id := range messageIDs {
			uuidMsgs[i] = id.UUID()
		}

		userRows, err := r.store.ReactionGetBatchByUserIDAndMessageIDs(ctx, db.ReactionGetBatchByUserIDAndMessageIDsParams{
			MessageIds: db.ToUUIDs(uuidMsgs),
			UserID:     db.ToUUID(userID.UUID()),
		})
		if err != nil {
			return nil, r.store.Err(err) // Fail fast on DB errors
		}

		for _, row := range userRows {
			rawMsgID := db.FromUUID[uuid.UUID](row.MessageID)
			msgID, parseErr := fields.ParseRequiredID("id", rawMsgID)
			if parseErr != nil {
				return nil, parseErr
			}
			if userReactions[msgID] == nil {
				userReactions[msgID] = make(map[string]bool)
			}
			userReactions[msgID][row.Emoji] = true
		}
	}

	// 4. Build final ReactionSummary DTOs
	summaries := make(map[fields.ID]*channel.ReactionSummary, len(messageIDs))
	for _, msgID := range messageIDs {
		countsMap := cachedCounts[msgID]
		userEmojiMap := userReactions[msgID]

		countsList := make([]channel.EmojiCount, 0, len(countsMap))
		for emoji, count := range countsMap {
			countsList = append(countsList, channel.EmojiCount{
				Emoji:   emoji,
				Count:   count,
				Reacted: userEmojiMap[emoji],
			})
		}

		summaries[msgID] = &channel.ReactionSummary{
			MessageID: msgID,
			Counts:    countsList,
		}
	}

	return summaries, nil
}

func (r *ReactionRepository) CountByEmoji(
	ctx context.Context,
	messageID fields.ID,
	emoji channel.ReactionEmoji,
) (int, error) {
	count, err := r.store.ReactionCountByEmoji(ctx, db.ReactionCountByEmojiParams{
		MessageID: db.ToUUID(messageID.UUID()),
		Emoji:     emoji.String(),
	})
	if err != nil {
		return 0, r.store.Err(err)
	}

	return int(count), nil
}

func (r *ReactionRepository) Delete(ctx context.Context, messageID, userID fields.ID, emoji channel.ReactionEmoji) error {
	err := r.store.ReactionDelete(ctx, db.ReactionDeleteParams{
		MessageID: db.ToUUID(messageID.UUID()),
		UserID:    db.ToUUID(userID.UUID()),
		Emoji:     emoji.String(),
	})
	if err != nil {
		return r.store.Err(err)
	}

	return nil
}

func reactionFromRow(row db.MessageReaction) (*channel.Reaction, error) {
	msgID := db.FromUUID[uuid.UUID](row.MessageID)
	userID := db.FromUUID[uuid.UUID](row.UserID)
	compositeKey := fmt.Sprintf("%s:%s:%s", msgID.String(), userID.String(), row.Emoji)

	mapErr := func(msgText, key string, val any, err error) *errs.Error {
		return errs.Internal(msgText).
			Wrap(err).
			Reason("CORRUPT_DATABASE_RECORD").
			Meta(key, fmt.Sprintf("%v", val)).
			Resource("MessageReaction", compositeKey, "", "database row mapping")
	}

	parsedMessageID, err := fields.ParseRequiredID("message_id", msgID)
	if err != nil {
		return nil, mapErr("failed to parse message id from database", "message_id", msgID.String(), err)
	}

	parsedUserID, err := fields.ParseRequiredID("user_id", userID)
	if err != nil {
		return nil, mapErr("failed to parse user id from database", "user_id", userID.String(), err)
	}

	emoji, err := channel.ParseReactionEmoji(row.Emoji)
	if err != nil {
		return nil, mapErr("failed to parse emoji from database", "emoji", row.Emoji, err)
	}

	createdAt := fields.NewTimestamp(db.FromTimestamptz(row.CreatedAt))

	return channel.ParseReaction(
		parsedMessageID,
		parsedUserID,
		emoji,
		createdAt,
	), nil
}
