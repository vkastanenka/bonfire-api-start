package repository

import (
	"context"
	"fmt"
	"log/slog"

	"bonfire-api/internal/channel"
	"bonfire-api/internal/db"
	"bonfire-api/internal/errs"
	"bonfire-api/internal/fields"
	"bonfire-api/internal/redis"

	"github.com/google/uuid"
	"golang.org/x/sync/singleflight"
)

type ChannelCache interface {
	Get(ctx context.Context, id fields.ID) (*channel.Channel, error)
	GetBatch(ctx context.Context, ids []fields.ID) (map[fields.ID]*channel.Channel, []fields.ID, error)
	Set(ctx context.Context, ch *channel.Channel) error
	SetBatch(ctx context.Context, channels []*channel.Channel) error
}

type ChannelRepository struct {
	store *db.Store
	cache ChannelCache
	sf    singleflight.Group
}

func NewChannelRepository(store *db.Store, cache ChannelCache) *ChannelRepository {
	return &ChannelRepository{
		store: store.WithEntity(db.EntityChannel),
		cache: cache,
	}
}

func (r *ChannelRepository) Create(ctx context.Context, ch *channel.Channel) (*channel.Channel, error) {
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

func (r *ChannelRepository) Get(ctx context.Context, id fields.ID) (*channel.Channel, error) {
	ch, err := r.cache.Get(ctx, id)
	if err == nil && ch != nil {
		return ch, nil
	}

	if err != nil {
		slog.WarnContext(ctx, "cache read failed, falling back to database",
			"id", id.String(),
			"error", err,
			"scope", redis.ScopeRelation,
		)
	}

	sfKey := "channel:get:" + id.String()
	sfCtx := context.WithoutCancel(ctx)

	val, err, _ := r.sf.Do(sfKey, func() (any, error) {
		row, err := r.store.ChannelGet(sfCtx, db.ToUUID(id.UUID()))
		if err != nil {
			return nil, r.store.Err(err)
		}

		dbCh, err := channelFromRow(row)
		if err != nil {
			return nil, err
		}

		if cacheErr := r.cache.Set(sfCtx, dbCh); cacheErr != nil {
			slog.WarnContext(sfCtx, "failed to backfill cache",
				"id", id.String(),
				"error", cacheErr,
				"scope", redis.ScopeRelation,
			)
		}

		return dbCh, nil
	})
	if err != nil {
		return nil, err
	}

	return val.(*channel.Channel), nil
}

func (r *ChannelRepository) GetForUpdate(ctx context.Context, id fields.ID) (*channel.Channel, error) {
	row, err := r.store.ChannelGetForUpdate(ctx, db.ToUUID(id.UUID()))
	if err != nil {
		return nil, r.store.Err(err)
	}

	return channelFromRow(row)
}

func (r *ChannelRepository) GetBatch(
	ctx context.Context,
	ids []fields.ID,
) (map[fields.ID]*channel.Channel, error) {
	cachedMap, missingIDs, err := r.cache.GetBatch(ctx, ids)
	if err != nil {
		slog.WarnContext(ctx, "cache read failed, treating all ids as missing",
			"count", len(ids),
			"error", err,
			"scope", redis.ScopeChannel,
		)
		cachedMap = make(map[fields.ID]*channel.Channel, len(ids))
		missingIDs = ids
	}

	if len(missingIDs) == 0 {
		return cachedMap, nil
	}

	uuidSlice := make([]uuid.UUID, len(missingIDs))
	for i, id := range missingIDs {
		uuidSlice[i] = id.UUID()
	}

	rows, err := r.store.ChannelGetBatch(ctx, db.ToUUIDs(uuidSlice))
	if err != nil {
		return nil, r.store.Err(err)
	}

	dbMap := make(map[fields.ID]*channel.Channel, len(rows))
	dbChannels := make([]*channel.Channel, 0, len(rows))

	for _, row := range rows {
		ch, err := channelFromRow(row)
		if err != nil {
			return nil, err
		}
		dbMap[ch.ID()] = ch
		dbChannels = append(dbChannels, ch)
	}

	if len(dbChannels) > 0 {
		if cacheErr := r.cache.SetBatch(ctx, dbChannels); cacheErr != nil {
			slog.WarnContext(ctx, "failed to backfill cache",
				"count", len(dbChannels),
				"error", cacheErr,
				"scope", redis.ScopeChannel,
			)
		}
	}

	for id, ch := range dbMap {
		cachedMap[id] = ch
	}

	return cachedMap, nil
}

func (r *ChannelRepository) UpdateGroup(
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

func (r *ChannelRepository) UpdateLastMessage(
	ctx context.Context,
	id fields.ID,
	lastMessageID fields.ID,
	lastMessageAt fields.Timestamp,
	updatedAt fields.Timestamp,
) (*channel.Channel, error) {
	row, err := r.store.ChannelUpdateLastMessage(ctx, db.ChannelUpdateLastMessageParams{
		ID:            db.ToUUID(id.UUID()),
		LastMessageID: db.ToUUIDPtr(lastMessageID.UUIDPtr()),
		LastMessageAt: db.ToTimestamptzPtr(lastMessageAt.TimePtr()),
		UpdatedAt:     db.ToTimestamptz(updatedAt.Time()),
	})
	if err != nil {
		return nil, r.store.Err(err)
	}

	return channelFromRow(row)
}

func (r *ChannelRepository) Delete(ctx context.Context, id fields.ID) error {
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

	name, err := channel.ParseChannelName(db.FromText[string](row.Name))
	if err != nil {
		return nil, mapErr("failed to parse channel name from database", "name", row.Name, err)
	}

	iconURL, err := fields.ParseURL("icon_url", db.FromText[string](row.IconURL))
	if err != nil {
		return nil, mapErr("failed to parse channel name from database", "name", row.Name, err)
	}

	lastMessageID, err := fields.ParseID("last_message_id", db.FromUUID[uuid.UUID](row.LastMessageID))
	if err != nil {
		return nil, mapErr("failed to parse channel name from database", "name", row.Name, err)
	}

	lastMessageAt := fields.NewTimestamp(db.FromTimestamptz(row.LastMessageAt))
	createdAt := fields.NewTimestamp(db.FromTimestamptz(row.CreatedAt))
	updatedAt := fields.NewTimestamp(db.FromTimestamptz(row.UpdatedAt))

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
