package repository

import (
	"bytes"
	"context"
	"encoding/json"
	"slices"
	"time"

	"bonfire-api/internal/channel"
	"bonfire-api/internal/db"
	"bonfire-api/internal/pkg/errs"

	"github.com/google/uuid"
)

type MessageRepository struct {
	store *db.Store
}

func NewMessageRepository(store *db.Store) *MessageRepository {
	return &MessageRepository{
		store: store,
	}
}

func (r *MessageRepository) Create(ctx context.Context, msg *channel.Message) (*channel.Message, error) {
	row, err := r.store.MessageCreate(ctx, db.MessageCreateParams{
		ID:               db.ToUUID(msg.ID),
		ChannelID:        db.ToUUID(msg.ChannelID),
		AuthorID:         db.ToUUIDPtr(msg.AuthorID),
		ReplyToMessageID: db.ToUUIDPtr(msg.ReplyToMessageID),
		ForwardMessageID: db.ToUUIDPtr(msg.ForwardMessageID),
		ForwardChannelID: db.ToUUIDPtr(msg.ForwardChannelID),
		CreatedAt:        db.ToTimestamptz(msg.CreatedAt),
		UpdatedAt:        db.ToTimestamptz(msg.UpdatedAt),
		EditedAt:         db.ToTimestamptzPtr(msg.EditedAt),
		PinnedAt:         db.ToTimestamptzPtr(msg.PinnedAt),
		Type:             int16(msg.Type),
		Content:          db.ToTextPtr(msg.Content),
		Metadata:         msg.Metadata,
	})
	if err != nil {
		return nil, db.NewError(err, db.EntityMessage)
	}

	return messageFromRow(row), nil
}

func (r *MessageRepository) CreateBatch(
	ctx context.Context,
	messages []*channel.Message,
) ([]*channel.Message, error) {
	if len(messages) == 0 {
		return []*channel.Message{}, nil
	}

	type messagePayload struct {
		ID               uuid.UUID       `json:"id"`
		ChannelID        uuid.UUID       `json:"channel_id"`
		AuthorID         *uuid.UUID      `json:"author_id,omitempty"`
		ReplyToMessageID *uuid.UUID      `json:"reply_to_message_id,omitempty"`
		ForwardMessageID *uuid.UUID      `json:"forward_message_id,omitempty"`
		ForwardChannelID *uuid.UUID      `json:"forward_channel_id,omitempty"`
		CreatedAt        time.Time       `json:"created_at"`
		UpdatedAt        time.Time       `json:"updated_at"`
		EditedAt         *time.Time      `json:"edited_at,omitempty"`
		PinnedAt         *time.Time      `json:"pinned_at,omitempty"`
		Type             int16           `json:"type"`
		Content          *string         `json:"content,omitempty"`
		Metadata         json.RawMessage `json:"metadata,omitempty"`
	}

	payloads := make([]messagePayload, len(messages))
	for i, msg := range messages {
		payloads[i] = messagePayload{
			ID:               msg.ID,
			ChannelID:        msg.ChannelID,
			AuthorID:         msg.AuthorID,
			ReplyToMessageID: msg.ReplyToMessageID,
			ForwardMessageID: msg.ForwardMessageID,
			ForwardChannelID: msg.ForwardChannelID,
			CreatedAt:        msg.CreatedAt,
			UpdatedAt:        msg.UpdatedAt,
			EditedAt:         msg.EditedAt,
			PinnedAt:         msg.PinnedAt,
			Type:             int16(msg.Type),
			Content:          msg.Content,
			Metadata:         msg.Metadata,
		}
	}

	jsonBytes, err := json.Marshal(payloads)
	if err != nil {
		return nil, errs.Internal("failed to marshal create message batch payload").
			Meta("entity", "message").
			Wrap(err)
	}

	rows, err := r.store.MessageCreateBatch(ctx, jsonBytes)
	if err != nil {
		return nil, db.NewError(err, db.EntityMessage)
	}

	result := make([]*channel.Message, len(rows))
	for i, row := range rows {
		msg := messageFromRow(row)
		result[i] = msg
	}

	return result, nil
}

func (r *MessageRepository) Get(ctx context.Context, id uuid.UUID) (*channel.Message, error) {
	row, err := r.store.MessageGet(ctx, db.ToUUID(id))
	if err != nil {
		return nil, db.NewError(err, db.EntityMessage)
	}

	return messageFromRow(row), nil
}

func (r *MessageRepository) ListAroundByChannelID(
	ctx context.Context,
	channelID, cursorID uuid.UUID,
	beforeLimit, afterLimit int,
) ([]*channel.Message, bool, bool, error) {
	rows, err := r.store.MessageListAroundByChannelID(ctx, db.MessageListAroundByChannelIDParams{
		ChannelID:         db.ToUUID(channelID),
		LastReadMessageID: db.ToUUID(cursorID),
		BeforeLimit:       int32(beforeLimit),
		AfterLimit:        int32(afterLimit),
	})
	if err != nil {
		return nil, false, false, db.NewError(err, db.EntityMessage)
	}

	messages, err := messagesFromRows(rows)
	if err != nil {
		return nil, false, false, err
	}

	var (
		beforeCount int
		afterCount  int
		targetID    = cursorID
	)

	for _, msg := range messages {
		msgUUID := msg.ID
		if bytes.Compare(msgUUID[:], targetID[:]) <= 0 {
			beforeCount++
		} else {
			afterCount++
		}
	}

	hasMoreBefore := beforeCount >= beforeLimit
	hasMoreAfter := afterCount >= afterLimit

	return messages, hasMoreBefore, hasMoreAfter, nil
}

func (r *MessageRepository) ListBeforeByChannelID(
	ctx context.Context,
	channelID, cursorID uuid.UUID,
	limit int,
) ([]*channel.Message, bool, error) {
	rows, err := r.store.MessageListBeforeByChannelID(ctx, db.MessageListBeforeByChannelIDParams{
		ChannelID: db.ToUUID(channelID),
		CursorID:  db.ToUUID(cursorID),
		LimitVal:  int32(limit + 1),
	})
	if err != nil {
		return nil, false, db.NewError(err, db.EntityMessage)
	}

	hasMoreBefore := len(rows) > limit
	if hasMoreBefore {
		rows = rows[:limit]
	}

	messages, err := messagesFromRows(rows)
	if err != nil {
		return nil, false, err
	}

	slices.Reverse(messages)
	return messages, hasMoreBefore, nil
}

func (r *MessageRepository) ListAfterByChannelID(
	ctx context.Context,
	channelID, cursorID uuid.UUID,
	limit int,
) ([]*channel.Message, bool, error) {
	rows, err := r.store.MessageListAfterByChannelID(ctx, db.MessageListAfterByChannelIDParams{
		ChannelID: db.ToUUID(channelID),
		CursorID:  db.ToUUID(cursorID),
		LimitVal:  int32(limit + 1),
	})
	if err != nil {
		return nil, false, db.NewError(err, db.EntityMessage)
	}

	hasMoreAfter := len(rows) > limit
	if hasMoreAfter {
		rows = rows[:limit]
	}

	messages, err := messagesFromRows(rows)
	if err != nil {
		return nil, false, err
	}

	return messages, hasMoreAfter, nil
}

func (r *MessageRepository) ListPinnedByChannelID(
	ctx context.Context,
	channelID uuid.UUID,
	cursorID *uuid.UUID,
	cursorPinnedAt *time.Time,
	limit int,
) ([]*channel.Message, bool, error) {
	rows, err := r.store.MessageListPinnedByChannelID(ctx, db.MessageListPinnedByChannelIDParams{
		ChannelID:      db.ToUUID(channelID),
		CursorID:       db.ToUUIDPtr(cursorID),
		CursorPinnedAt: db.ToTimestamptzPtr(cursorPinnedAt),
		LimitVal:       int32(limit),
	})
	if err != nil {
		return nil, false, db.NewError(err, db.EntityMessage)
	}

	messages, err := messagesFromRows(rows)
	if err != nil {
		return nil, false, err
	}

	hasMoreBefore := len(messages) >= limit

	return messages, hasMoreBefore, nil
}

func (r *MessageRepository) CountByChannelID(ctx context.Context, channelID uuid.UUID) (int, error) {
	count, err := r.store.MessageCountByChannelID(ctx, db.ToUUID(channelID))
	if err != nil {
		return 0, db.NewError(err, db.EntityMessage)
	}

	return int(count), nil
}

func (r *MessageRepository) UpdateContent(
	ctx context.Context,
	id uuid.UUID,
	content string,
	editedAt, updatedAt time.Time,
) (*channel.Message, error) {
	row, err := r.store.MessageUpdateContent(ctx, db.MessageUpdateContentParams{
		ID:        db.ToUUID(id),
		Content:   content,
		EditedAt:  db.ToTimestamptz(editedAt),
		UpdatedAt: db.ToTimestamptz(updatedAt),
	})
	if err != nil {
		return nil, db.NewError(err, db.EntityMessage)
	}

	return messageFromRow(row), nil
}

func (r *MessageRepository) UpdatePinnedAt(
	ctx context.Context,
	id uuid.UUID,
	pinnedAt *time.Time,
	updatedAt time.Time,
) (*channel.Message, error) {
	row, err := r.store.MessageUpdatePinnedAt(ctx, db.MessageUpdatePinnedAtParams{
		ID:        db.ToUUID(id),
		PinnedAt:  db.ToTimestamptzPtr(pinnedAt),
		UpdatedAt: db.ToTimestamptz(updatedAt),
	})
	if err != nil {
		return nil, db.NewError(err, db.EntityMessage)
	}

	return messageFromRow(row), nil
}

func (r *MessageRepository) Delete(ctx context.Context, id uuid.UUID) error {
	err := r.store.MessageDelete(ctx, db.ToUUID(id))
	if err != nil {
		return db.NewError(err, db.EntityMessage)
	}

	return nil
}

func messageFromRow(row db.Message) *channel.Message {
	return channel.ReconstituteMessage(
		db.FromUUID(row.ID),
		db.FromUUID(row.ChannelID),
		db.FromUUIDPtr(row.AuthorID),
		channel.MessageType(int(row.Type)),
		db.FromTextPtr(row.Content),
		row.Metadata,
		db.FromUUIDPtr(row.ReplyToMessageID),
		db.FromUUIDPtr(row.ForwardMessageID),
		db.FromUUIDPtr(row.ForwardChannelID),
		db.FromTimestamptzPtr(row.PinnedAt),
		db.FromTimestamptzPtr(row.EditedAt),
		db.FromTimestamptz(row.CreatedAt),
		db.FromTimestamptz(row.UpdatedAt),
	)
}

func messagesFromRows(rows []db.Message) ([]*channel.Message, error) {
	messages := make([]*channel.Message, 0, len(rows))
	for _, row := range rows {
		messages = append(messages, messageFromRow(row))
	}
	return messages, nil
}
