package repository

import (
	"context"

	"bonfire-api/internal/channel"
	"bonfire-api/internal/db"

	"github.com/google/uuid"
)

type ReactionRepository struct {
	store *db.Store
}

func NewReactionRepository(store *db.Store) *ReactionRepository {
	return &ReactionRepository{
		store: store.WithEntity(db.EntityMessageReaction),
	}
}

func (r *ReactionRepository) Create(ctx context.Context, rx *channel.Reaction) (*channel.Reaction, error) {
	row, err := r.store.ReactionCreate(ctx, db.ReactionCreateParams{
		MessageID: db.ToUUID(rx.MessageID),
		UserID:    db.ToUUID(rx.UserID),
		Emoji:     rx.Emoji,
		CreatedAt: db.ToTimestamptz(rx.CreatedAt),
	})
	if err != nil {
		return nil, r.store.Err(err)
	}

	return reactionFromRow(row)
}

func (r *ReactionRepository) Get(
	ctx context.Context,
	messageID, userID uuid.UUID,
	emoji string,
) (*channel.Reaction, error) {
	row, err := r.store.ReactionGet(ctx, db.ReactionGetParams{
		MessageID: db.ToUUID(messageID),
		UserID:    db.ToUUID(userID),
		Emoji:     emoji,
	})
	if err != nil {
		return nil, r.store.Err(err)
	}

	return reactionFromRow(row)
}

func (r *ReactionRepository) GetBatchSummaryByMessageIDs(
	ctx context.Context,
	userID uuid.UUID,
	messageIDs []uuid.UUID,
) (map[uuid.UUID]*channel.ReactionSummary, error) {
	summaries := make(map[uuid.UUID]*channel.ReactionSummary, len(messageIDs))
	if len(messageIDs) == 0 {
		return summaries, nil
	}

	uuidMsgs := make([]uuid.UUID, len(messageIDs))
	for i, id := range messageIDs {
		uuidMsgs[i] = id
		summaries[id] = &channel.ReactionSummary{
			MessageID: id,
			Counts:    []channel.EmojiCount{},
		}
	}

	dbRows, err := r.store.ReactionGetBatchSummaryByMessageIDs(ctx, db.ToUUIDs(uuidMsgs))
	if err != nil {
		return nil, r.store.Err(err)
	}

	countsMap := make(map[uuid.UUID]map[string]int, len(messageIDs))
	for _, row := range dbRows {
		msgID := db.FromUUID[uuid.UUID](row.MessageID)
		if countsMap[msgID] == nil {
			countsMap[msgID] = make(map[string]int)
		}
		countsMap[msgID][row.Emoji] = int(row.Count)
	}

	userReactions := make(map[uuid.UUID]map[string]bool)
	if userID != uuid.Nil {
		userRows, err := r.store.ReactionGetBatchByUserIDAndMessageIDs(ctx, db.ReactionGetBatchByUserIDAndMessageIDsParams{
			MessageIds: db.ToUUIDs(uuidMsgs),
			UserID:     db.ToUUID(userID),
		})
		if err != nil {
			return nil, r.store.Err(err)
		}

		for _, row := range userRows {
			msgID := db.FromUUID[uuid.UUID](row.MessageID)
			if userReactions[msgID] == nil {
				userReactions[msgID] = make(map[string]bool)
			}
			userReactions[msgID][row.Emoji] = true
		}
	}

	for msgID, emojiCounts := range countsMap {
		userEmojiMap := userReactions[msgID]
		countsList := make([]channel.EmojiCount, 0, len(emojiCounts))

		for emoji, count := range emojiCounts {
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
	messageID uuid.UUID,
	emoji string,
) (int, error) {
	count, err := r.store.ReactionCountByEmoji(ctx, db.ReactionCountByEmojiParams{
		MessageID: db.ToUUID(messageID),
		Emoji:     emoji,
	})
	if err != nil {
		return 0, r.store.Err(err)
	}

	return int(count), nil
}

func (r *ReactionRepository) Delete(
	ctx context.Context,
	messageID, userID uuid.UUID,
	emoji string,
) error {
	err := r.store.ReactionDelete(ctx, db.ReactionDeleteParams{
		MessageID: db.ToUUID(messageID),
		UserID:    db.ToUUID(userID),
		Emoji:     emoji,
	})
	if err != nil {
		return r.store.Err(err)
	}

	return nil
}

func reactionFromRow(row db.MessageReaction) (*channel.Reaction, error) {
	return channel.ReconstituteReaction(
		db.FromUUID[uuid.UUID](row.MessageID),
		db.FromUUID[uuid.UUID](row.UserID),
		row.Emoji,
		db.FromTimestamptz(row.CreatedAt),
	), nil
}
