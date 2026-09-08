package repository

import (
	"context"

	"bonfire-api/internal/channel"
	"bonfire-api/internal/fields"
)

type CachedChannelRepository struct {
	cache ChannelCache
	repo  *ChannelRepository
}

func NewCachedChannelRepository(cache ChannelCache, repo *ChannelRepository) *CachedChannelRepository {
	return &CachedChannelRepository{
		cache: cache,
		repo:  repo,
	}
}

func (r *CachedChannelRepository) Get(ctx context.Context, id fields.ID) (*channel.Channel, error) {
	ch, err := r.cache.Get(ctx, id)
	if err == nil && ch != nil {
		return ch, nil
	}

	ch, err = r.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	_ = r.cache.Set(ctx, ch)

	return ch, nil
}

func (r *CachedChannelRepository) GetBatch(
	ctx context.Context,
	ids []fields.ID,
) (map[fields.ID]*channel.Channel, error) {
	if len(ids) == 0 {
		return make(map[fields.ID]*channel.Channel), nil
	}

	cached, missing, err := r.cache.GetBatch(ctx, ids)
	if err != nil {
		missing = ids
		cached = make(map[fields.ID]*channel.Channel, len(ids))
	}

	if len(missing) == 0 {
		return cached, nil
	}

	dbChannels, err := r.repo.GetBatch(ctx, missing)
	if err != nil {
		return nil, err
	}

	if len(dbChannels) > 0 {
		_ = r.cache.SetBatch(ctx, dbChannels)
	}

	for id, ch := range dbChannels {
		cached[id] = ch
	}

	return cached, nil
}
