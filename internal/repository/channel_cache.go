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
