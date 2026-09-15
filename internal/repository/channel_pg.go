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
		store: store.WithEntity(db.EntityChannel),
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
		return nil, r.store.Err(err)
	}

	return channelFromRow(row), nil
}

func (r *ChannelRepository) Get(ctx context.Context, id uuid.UUID) (*channel.Channel, error) {
	row, err := r.store.ChannelGet(ctx, db.ToUUID(id))
	if err != nil {
		return nil, r.store.Err(err)
	}

	return channelFromRow(row), nil
}

func (r *ChannelRepository) GetForUpdate(ctx context.Context, id uuid.UUID) (*channel.Channel, error) {
	row, err := r.store.ChannelGetForUpdate(ctx, db.ToUUID(id))
	if err != nil {
		return nil, r.store.Err(err)
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

	uuidSlice := make([]uuid.UUID, len(ids))
	for i, id := range ids {
		uuidSlice[i] = id
	}

	rows, err := r.store.ChannelGetBatch(ctx, db.ToUUIDs(uuidSlice))
	if err != nil {
		return nil, r.store.Err(err)
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
		return nil, r.store.Err(err)
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
		return nil, r.store.Err(err)
	}

	return channelFromRow(row), nil
}

func (r *ChannelRepository) Delete(ctx context.Context, id uuid.UUID) error {
	err := r.store.ChannelDelete(ctx, db.ToUUID(id))
	if err != nil {
		return r.store.Err(err)
	}

	return nil
}

func channelFromRow(row db.Channel) *channel.Channel {
	return channel.ReconstituteChannel(
		db.FromUUID[uuid.UUID](row.ID),
		channel.ChannelType(row.Type),
		db.FromTextPtr[string](row.Name),
		db.FromTextPtr[string](row.IconURL),
		db.FromUUIDPtr[uuid.UUID](row.LastMessageID),
		db.FromTimestamptzPtr(row.LastMessageAt),
		db.FromTimestamptz(row.CreatedAt),
		db.FromTimestamptz(row.UpdatedAt),
	)
}
