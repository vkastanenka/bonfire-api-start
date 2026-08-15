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

type ChannelRepository struct {
	store *db.Store
}

func NewChannelRepository(store *db.Store) *ChannelRepository {
	return &ChannelRepository{
		store: store.WithEntity(db.EntityChannel),
	}
}

func (r *ChannelRepository) ChannelCreate(ctx context.Context, ch *channel.Channel) (*channel.Channel, error) {
	row, err := r.store.ChannelCreate(ctx, db.ChannelCreateParams{
		ID:        db.ToUUID(ch.ID().UUID()),
		CreatedAt: db.ToTimestamptz(ch.CreatedAt().Time()),
		UpdatedAt: db.ToTimestamptz(ch.UpdatedAt().Time()),
		Type:      ch.Type().Int16(),
		Name:      db.ToTextPtr(ch.Name().StringPtr()),
		IconURL:   db.ToTextPtr(ch.IconURL().StringPtr()),
	})
	if err != nil {
		return nil, r.store.Err(err)
	}

	return channelFromRow(row)
}

func (r *ChannelRepository) ChannelGet(ctx context.Context, id fields.ID) (*channel.Channel, error) {
	row, err := r.store.ChannelGet(ctx, db.ToUUID(id.UUID()))
	if err != nil {
		return nil, r.store.Err(err)
	}

	return channelFromRow(row)
}

func (r *ChannelRepository) ChannelGetBatch(ctx context.Context, ids []fields.ID) ([]*channel.Channel, error) {
	if len(ids) == 0 {
		return []*channel.Channel{}, nil
	}

	uuidSlice := make([]uuid.UUID, len(ids))
	for i, id := range ids {
		uuidSlice[i] = id.UUID()
	}

	rows, err := r.store.ChannelGetBatch(ctx, db.ToUUIDs(uuidSlice))
	if err != nil {
		return nil, r.store.Err(err)
	}

	channels := make([]*channel.Channel, 0, len(rows))
	for _, row := range rows {
		ch, err := channelFromRow(row)
		if err != nil {
			return nil, err
		}
		channels = append(channels, ch)
	}

	return channels, nil
}

func (r *ChannelRepository) ChannelUpdateGroup(
	ctx context.Context,
	id fields.ID,
	name channel.ChannelName,
	iconURL fields.URL,
	updatedAt fields.Timestamp,
) (*channel.Channel, error) {
	row, err := r.store.ChannelUpdateGroup(ctx, db.ChannelUpdateGroupParams{
		ID:        db.ToUUID(id.UUID()),
		Name:      db.ToTextPtr(name.StringPtr()),
		IconURL:   db.ToTextPtr(iconURL.StringPtr()),
		UpdatedAt: db.ToTimestamptz(updatedAt.Time()),
	})
	if err != nil {
		return nil, r.store.Err(err)
	}

	return channelFromRow(row)
}

func (r *ChannelRepository) ChannelUpdateLastMessage(
	ctx context.Context,
	id fields.ID,
	lastMessageID fields.ID,
	lastMessageAt fields.Timestamp,
	updatedAt fields.Timestamp,
) (*channel.Channel, error) {
	row, err := r.store.ChannelUpdateLastMessage(ctx, db.ChannelUpdateLastMessageParams{
		ID:            db.ToUUID(id.UUID()),
		LastMessageID: db.ToUUID(lastMessageID.UUID()),
		LastMessageAt: db.ToTimestamptz(lastMessageAt.Time()),
		UpdatedAt:     db.ToTimestamptz(updatedAt.Time()),
	})
	if err != nil {
		return nil, r.store.Err(err)
	}

	return channelFromRow(row)
}

func (r *ChannelRepository) ChannelDelete(ctx context.Context, id fields.ID) error {
	err := r.store.ChannelDelete(ctx, db.ToUUID(id.UUID()))
	if err != nil {
		return r.store.Err(err)
	}

	return nil
}

func channelFromRow(row db.Channel) (*channel.Channel, error) {
	channelID := db.FromUUID[uuid.UUID](row.ID)
	channelIDStr := channelID.String()

	mapErr := func(msg, key string, val any, err error) *errs.Error {
		return errs.Internal(msg).
			Wrap(err).
			Reason("CORRUPT_DATABASE_RECORD").
			Meta(key, fmt.Sprintf("%v", val)).
			Resource("Channel", channelIDStr, "", "database row mapping")
	}

	id, err := fields.ParseRequiredID("id", channelID)
	if err != nil {
		return nil, mapErr("failed to parse channel id from database", "id", channelIDStr, err)
	}

	chType, err := channel.ParseChannelType(row.Type)
	if err != nil {
		return nil, mapErr("failed to parse channel type from database", "type", row.Type, err)
	}

	name, err := channel.ParseChannelName(db.FromTextPtr[string](row.Name))
	if err != nil {
		return nil, mapErr("failed to parse channel name from database", "name", row.Name, err)
	}

	iconURL, err := fields.ParseURL("icon_url", db.FromTextPtr[string](row.IconURL))
	if err != nil {
		return nil, mapErr("failed to parse channel name from database", "name", row.Name, err)
	}

	// lastMessageID, err := fields.ParseID("last_message_id", db.FromUUIDPtr[uuid.UUID](row.LastMessageID))
	// if err != nil {
	// 	return nil, mapErr("failed to parse channel name from database", "name", row.Name, err)
	// }

	var lastMessageID fields.ID

	lastMessageAt := fields.NewTimestampFromTime(db.FromTimestamptz(row.LastMessageAt))
	createdAt := fields.NewTimestampFromTime(db.FromTimestamptz(row.CreatedAt))
	updatedAt := fields.NewTimestampFromTime(db.FromTimestamptz(row.UpdatedAt))

	return channel.ParseChannel(
		id,
		chType,
		name,
		iconURL,
		lastMessageID,
		lastMessageAt,
		createdAt,
		updatedAt,
	), nil
}
