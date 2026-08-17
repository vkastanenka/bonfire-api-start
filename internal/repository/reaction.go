package repository

import (
	"context"
	"fmt"

	"bonfire-api/internal/channel"
	"bonfire-api/internal/db"
	"bonfire-api/internal/errs"
	"bonfire-api/internal/fields"

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
	row, err := r.store.ReactionGet(ctx, db.ReactionGetParams{
		MessageID: db.ToUUID(messageID.UUID()),
		UserID:    db.ToUUID(userID.UUID()),
		Emoji:     emoji.String(),
	})
	if err != nil {
		return nil, r.store.Err(err)
	}

	return reactionFromRow(row)
}

func (r *ReactionRepository) GetBatchByMessageIDs(
	ctx context.Context,
	messageIDs []fields.ID,
) (map[fields.ID][]*channel.Reaction, error) {
	if len(messageIDs) == 0 {
		return make(map[fields.ID][]*channel.Reaction), nil
	}

	uuidSlice := make([]uuid.UUID, len(messageIDs))
	for i, id := range messageIDs {
		uuidSlice[i] = id.UUID()
	}

	rows, err := r.store.ReactionGetBatchByMessageIDs(ctx, db.ToUUIDs(uuidSlice))
	if err != nil {
		return nil, r.store.Err(err)
	}

	reactionMap := make(map[fields.ID][]*channel.Reaction, len(messageIDs))
	for _, row := range rows {
		rx, err := reactionFromRow(row)
		if err != nil {
			return nil, err
		}

		msgID := rx.MessageID()
		reactionMap[msgID] = append(reactionMap[msgID], rx)
	}

	return reactionMap, nil
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
