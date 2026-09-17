package repository

import (
	"context"
	"time"

	"bonfire-api/internal/channel"
	"bonfire-api/internal/db"

	"github.com/google/uuid"
)

type ChannelRepository struct {
	store *db.Store
}

func NewChannelRepository(store *db.Store) *ChannelRepository {
	return &ChannelRepository{
		store: store,
	}
}

func (r *ChannelRepository) Create(ctx context.Context, ch *channel.Channel) (*channel.Channel, error) {
	row, err := r.store.ChannelCreate(ctx, db.ChannelCreateParams{
		ID:            db.ToUUID(ch.ID),
		LastMessageID: db.ToUUIDPtr(ch.LastMessageID),
		CreatedAt:     db.ToTimestamptz(ch.CreatedAt),
		UpdatedAt:     db.ToTimestamptz(ch.UpdatedAt),
		LastMessageAt: db.ToTimestamptzPtr(ch.LastMessageAt),
		Type:          int16(ch.Type),
		Name:          db.ToTextPtr(ch.Name),
		IconURL:       db.ToTextPtr(ch.IconURL),
	})
	if err != nil {
		return nil, db.NewError(err, db.EntityChannel)
	}

	return channelFromRow(row), nil
}

func (r *ChannelRepository) Get(ctx context.Context, id uuid.UUID) (*channel.Channel, error) {
	row, err := r.store.ChannelGet(ctx, db.ToUUID(id))
	if err != nil {
		return nil, db.NewError(err, db.EntityChannel)
	}

	return channelFromRow(row), nil
}

func (r *ChannelRepository) GetForUpdate(ctx context.Context, id uuid.UUID) (*channel.Channel, error) {
	row, err := r.store.ChannelGetForUpdate(ctx, db.ToUUID(id))
	if err != nil {
		return nil, db.NewError(err, db.EntityChannel)
	}

	return channelFromRow(row), nil
}

func (r *ChannelRepository) GetBatch(
	ctx context.Context,
	ids []uuid.UUID,
) (map[uuid.UUID]*channel.Channel, error) {
	if len(ids) == 0 {
		return make(map[uuid.UUID]*channel.Channel), nil
	}

	rows, err := r.store.ChannelGetBatch(ctx, db.ToUUIDs(ids))
	if err != nil {
		return nil, db.NewError(err, db.EntityChannel)
	}

	resultMap := make(map[uuid.UUID]*channel.Channel, len(rows))
	for _, row := range rows {
		ch := channelFromRow(row)
		resultMap[ch.ID] = ch
	}

	return resultMap, nil
}

func (r *ChannelRepository) UpdateGroup(
	ctx context.Context,
	id uuid.UUID,
	name *string,
	iconURL *string,
	updatedAt time.Time,
) (*channel.Channel, error) {
	row, err := r.store.ChannelUpdateGroup(ctx, db.ChannelUpdateGroupParams{
		ID:        db.ToUUID(id),
		Name:      db.ToTextPtr(name),
		IconURL:   db.ToTextPtr(iconURL),
		UpdatedAt: db.ToTimestamptz(updatedAt),
	})
	if err != nil {
		return nil, db.NewError(err, db.EntityChannel)
	}

	return channelFromRow(row), nil
}

func (r *ChannelRepository) UpdateLastMessage(
	ctx context.Context,
	id uuid.UUID,
	lastMessageID *uuid.UUID,
	lastMessageAt *time.Time,
	updatedAt time.Time,
) (*channel.Channel, error) {
	row, err := r.store.ChannelUpdateLastMessage(ctx, db.ChannelUpdateLastMessageParams{
		ID:            db.ToUUID(id),
		LastMessageID: db.ToUUIDPtr(lastMessageID),
		LastMessageAt: db.ToTimestamptzPtr(lastMessageAt),
		UpdatedAt:     db.ToTimestamptz(updatedAt),
	})
	if err != nil {
		return nil, db.NewError(err, db.EntityChannel)
	}

	return channelFromRow(row), nil
}

func (r *ChannelRepository) Delete(ctx context.Context, id uuid.UUID) error {
	err := r.store.ChannelDelete(ctx, db.ToUUID(id))
	if err != nil {
		return db.NewError(err, db.EntityChannel)
	}

	return nil
}

func channelFromRow(row db.Channel) *channel.Channel {
	return channel.ReconstituteChannel(
		db.FromUUID(row.ID),
		channel.ChannelType(row.Type),
		db.FromTextPtr(row.Name),
		db.FromTextPtr(row.IconURL),
		db.FromUUIDPtr(row.LastMessageID),
		db.FromTimestamptzPtr(row.LastMessageAt),
		db.FromTimestamptz(row.CreatedAt),
		db.FromTimestamptz(row.UpdatedAt),
	)
}
