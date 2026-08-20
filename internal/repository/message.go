package repository

import (
	"context"
	"encoding/json"
	"fmt"

	"bonfire-api/internal/channel"
	"bonfire-api/internal/db"
	"bonfire-api/internal/errs"
	"bonfire-api/internal/fields"

	"github.com/google/uuid"
)

type MessageRepository struct {
	store *db.Store
}

func NewMessageRepository(store *db.Store) *MessageRepository {
	return &MessageRepository{
		store: store.WithEntity(db.EntityMessage),
	}
}

func (r *MessageRepository) Create(ctx context.Context, msg *channel.Message) (*channel.Message, error) {
	row, err := r.store.MessageCreate(ctx, db.MessageCreateParams{
		ID:                 db.ToUUID(msg.ID().UUID()),
		ChannelID:          db.ToUUID(msg.ChannelID().UUID()),
		AuthorID:           db.ToUUIDPtr(msg.AuthorID().UUIDPtr()),
		ReplyToMessageID:   db.ToUUIDPtr(msg.ReplyToMessageID().UUIDPtr()),
		ForwardedMessageID: db.ToUUIDPtr(msg.ForwardedMessageID().UUIDPtr()),
		ForwardedChannelID: db.ToUUIDPtr(msg.ForwardedChannelID().UUIDPtr()),
		CreatedAt:          db.ToTimestamptz(msg.CreatedAt().Time()),
		UpdatedAt:          db.ToTimestamptz(msg.UpdatedAt().Time()),
		Type:               msg.Type().Int16(),
		Content:            db.ToTextPtr(msg.Content().StringPtr()),
		SystemMetadata:     msg.SystemMetadata().Bytes(),
	})
	if err != nil {
		return nil, r.store.Err(err)
	}

	return messageFromRow(row)
}

func (r *MessageRepository) CreateBatch(
	ctx context.Context,
	messages []*channel.Message,
) ([]*channel.Message, error) {
	type messagePayload struct {
		ID                 uuid.UUID       `json:"id"`
		ChannelID          uuid.UUID       `json:"channel_id"`
		AuthorID           *uuid.UUID      `json:"author_id,omitempty"`
		ReplyToMessageID   *uuid.UUID      `json:"reply_to_message_id,omitempty"`
		ForwardedMessageID *uuid.UUID      `json:"forwarded_message_id,omitempty"`
		ForwardedChannelID *uuid.UUID      `json:"forwarded_channel_id,omitempty"`
		CreatedAt          string          `json:"created_at"`
		UpdatedAt          string          `json:"updated_at"`
		Type               int16           `json:"type"`
		Content            *string         `json:"content,omitempty"`
		SystemMetadata     json.RawMessage `json:"system_metadata,omitempty"`
	}

	payloads := make([]messagePayload, len(messages))
	for i, msg := range messages {
		payloads[i] = messagePayload{
			ID:                 msg.ID().UUID(),
			ChannelID:          msg.ChannelID().UUID(),
			AuthorID:           msg.AuthorID().UUIDPtr(),
			ReplyToMessageID:   msg.ReplyToMessageID().UUIDPtr(),
			ForwardedMessageID: msg.ForwardedMessageID().UUIDPtr(),
			ForwardedChannelID: msg.ForwardedChannelID().UUIDPtr(),
			CreatedAt:          msg.CreatedAt().String(),
			UpdatedAt:          msg.UpdatedAt().String(),
			Type:               msg.Type().Int16(),
			Content:            msg.Content().StringPtr(),
			SystemMetadata:     json.RawMessage(msg.SystemMetadata().Bytes()),
		}
	}

	jsonBytes, err := json.Marshal(payloads)
	if err != nil {
		return nil, errs.Internal("failed to marshal create message batch payload").
			Meta("entity", db.EntityMessage.String()).
			Wrap(err)
	}

	rows, err := r.store.MessageCreateBatch(ctx, jsonBytes)
	if err != nil {
		return nil, r.store.Err(err)
	}

	result := make([]*channel.Message, len(rows))
	for i, row := range rows {
		msg, err := messageFromRow(row)
		if err != nil {
			return nil, err
		}
		result[i] = msg
	}

	return result, nil
}

func (r *MessageRepository) Get(ctx context.Context, id fields.ID) (*channel.Message, error) {
	row, err := r.store.MessageGet(ctx, db.ToUUID(id.UUID()))
	if err != nil {
		return nil, r.store.Err(err)
	}

	return messageFromRow(row)
}

func (r *MessageRepository) ListAroundByChannelID(
	ctx context.Context,
	channelID, lastReadMessageID fields.ID,
	beforeLimit, afterLimit int32,
) ([]*channel.Message, error) {
	rows, err := r.store.MessageListAroundByChannelID(ctx, db.MessageListAroundByChannelIDParams{
		ChannelID:         db.ToUUID(channelID.UUID()),
		LastReadMessageID: db.ToUUID(lastReadMessageID.UUID()),
		BeforeLimit:       beforeLimit,
		AfterLimit:        afterLimit,
	})
	if err != nil {
		return nil, r.store.Err(err)
	}

	return messagesFromRows(rows)
}

func (r *MessageRepository) ListBeforeByChannelID(
	ctx context.Context,
	channelID, cursorID fields.ID,
	limit int32,
) ([]*channel.Message, error) {
	rows, err := r.store.MessageListBeforeByChannelID(ctx, db.MessageListBeforeByChannelIDParams{
		ChannelID: db.ToUUID(channelID.UUID()),
		CursorID:  db.ToUUID(cursorID.UUID()),
		LimitVal:  limit,
	})
	if err != nil {
		return nil, r.store.Err(err)
	}

	return messagesFromRows(rows)
}

func (r *MessageRepository) ListAfterByChannelID(
	ctx context.Context,
	channelID, cursorID fields.ID,
	limit int32,
) ([]*channel.Message, error) {
	rows, err := r.store.MessageListAfterByChannelID(ctx, db.MessageListAfterByChannelIDParams{
		ChannelID: db.ToUUID(channelID.UUID()),
		CursorID:  db.ToUUID(cursorID.UUID()),
		LimitVal:  limit,
	})
	if err != nil {
		return nil, r.store.Err(err)
	}

	return messagesFromRows(rows)
}

func (r *MessageRepository) ListPinnedByChannelID(
	ctx context.Context,
	channelID fields.ID,
	cursorID fields.ID,
	cursorPinnedAt fields.Timestamp,
	limit int32,
) ([]*channel.Message, error) {
	rows, err := r.store.MessageListPinnedByChannelID(ctx, db.MessageListPinnedByChannelIDParams{
		ChannelID:      db.ToUUID(channelID.UUID()),
		CursorID:       db.ToUUIDPtr(cursorID.UUIDPtr()),
		CursorPinnedAt: db.ToTimestamptzPtr(cursorPinnedAt.TimePtr()),
		LimitVal:       limit,
	})
	if err != nil {
		return nil, r.store.Err(err)
	}

	return messagesFromRows(rows)
}

func (r *MessageRepository) UpdateContent(
	ctx context.Context,
	id fields.ID,
	content channel.MessageContent,
	editedAt, updatedAt fields.Timestamp,
) (*channel.Message, error) {
	row, err := r.store.MessageUpdateContent(ctx, db.MessageUpdateContentParams{
		ID:        db.ToUUID(id.UUID()),
		Content:   content.String(),
		EditedAt:  db.ToTimestamptz(editedAt.Time()),
		UpdatedAt: db.ToTimestamptz(updatedAt.Time()),
	})
	if err != nil {
		return nil, r.store.Err(err)
	}

	return messageFromRow(row)
}

func (r *MessageRepository) UpdatePinnedAt(
	ctx context.Context,
	id fields.ID,
	pinnedAt, updatedAt fields.Timestamp,
) (*channel.Message, error) {
	row, err := r.store.MessageUpdatePinnedAt(ctx, db.MessageUpdatePinnedAtParams{
		ID:        db.ToUUID(id.UUID()),
		PinnedAt:  db.ToTimestamptzPtr(pinnedAt.TimePtr()),
		UpdatedAt: db.ToTimestamptz(updatedAt.Time()),
	})
	if err != nil {
		return nil, r.store.Err(err)
	}

	return messageFromRow(row)
}

func (r *MessageRepository) Delete(ctx context.Context, id fields.ID) error {
	err := r.store.MessageDelete(ctx, db.ToUUID(id.UUID()))
	if err != nil {
		return r.store.Err(err)
	}

	return nil
}

func messagesFromRows(rows []db.Message) ([]*channel.Message, error) {
	messages := make([]*channel.Message, 0, len(rows))
	for _, row := range rows {
		msg, err := messageFromRow(row)
		if err != nil {
			return nil, err
		}
		messages = append(messages, msg)
	}
	return messages, nil
}

func messageFromRow(row db.Message) (*channel.Message, error) {
	msgID := db.FromUUID[uuid.UUID](row.ID)
	msgIDStr := msgID.String()

	mapErr := func(msgText, key string, val any, err error) *errs.Error {
		return errs.Internal(msgText).
			Wrap(err).
			Reason("CORRUPT_DATABASE_RECORD").
			Meta(key, fmt.Sprintf("%v", val)).
			Resource("Message", msgIDStr, "", "database row mapping")
	}

	id, err := fields.ParseRequiredID("id", msgID)
	if err != nil {
		return nil, mapErr("failed to parse message id from database", "id", msgIDStr, err)
	}

	channelID, err := fields.ParseRequiredID("channel_id", db.FromUUID[uuid.UUID](row.ChannelID))
	if err != nil {
		return nil, mapErr("failed to parse channel id from database", "channel_id", row.ChannelID, err)
	}

	authorID, err := fields.ParseID(db.FromUUID[uuid.UUID](row.AuthorID))
	if err != nil {
		return nil, mapErr("failed to parse author id from database", "author_id", row.AuthorID, err)
	}

	replyToMessageID, err := fields.ParseID(db.FromUUID[uuid.UUID](row.ReplyToMessageID))
	if err != nil {
		return nil, mapErr("failed to parse reply to message id from database", "reply_to_message_id", row.ReplyToMessageID, err)
	}

	forwardedMessageID, err := fields.ParseID(db.FromUUID[uuid.UUID](row.ForwardedMessageID))
	if err != nil {
		return nil, mapErr("failed to parse forwarded message id from database", "forwarded_message_id", row.ForwardedMessageID, err)
	}

	forwardedChannelID, err := fields.ParseID(db.FromUUID[uuid.UUID](row.ForwardedChannelID))
	if err != nil {
		return nil, mapErr("failed to parse forwarded channel id from database", "forwarded_channel_id", row.ForwardedChannelID, err)
	}

	msgType, err := channel.ParseMessageType(row.Type)
	if err != nil {
		return nil, mapErr("failed to parse message type from database", "type", row.Type, err)
	}

	content, err := channel.ParseMessageContent(db.FromText[string](row.Content))
	if err != nil {
		return nil, mapErr("failed to parse message content from database", "content", row.Content, err)
	}

	systemMetadata, err := fields.ParseJSON("system_metadata", row.SystemMetadata)
	if err != nil {
		return nil, mapErr("failed to parse system metadata from database", "system_metadata", string(row.SystemMetadata), err)
	}

	editedAt := fields.NewTimestamp(db.FromTimestamptz(row.EditedAt))
	pinnedAt := fields.NewTimestamp(db.FromTimestamptz(row.PinnedAt))
	createdAt := fields.NewTimestamp(db.FromTimestamptz(row.CreatedAt))
	updatedAt := fields.NewTimestamp(db.FromTimestamptz(row.UpdatedAt))

	return channel.ParseMessage(
		id,
		channelID,
		authorID,
		msgType,
		content,
		systemMetadata,
		replyToMessageID,
		forwardedMessageID,
		forwardedChannelID,
		editedAt,
		pinnedAt,
		createdAt,
		updatedAt,
	), nil
}
