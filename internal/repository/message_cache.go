package repository

import (
	"context"

	"bonfire-api/internal/channel"

	"github.com/google/uuid"
)

type CachedMessageRepository struct {
	cache MessageCache
	repo  *MessageRepository
}

func NewCachedMessageRepository(cache MessageCache, repo *MessageRepository) *CachedMessageRepository {
	return &CachedMessageRepository{
		cache: cache,
		repo:  repo,
	}
}

func (r *CachedMessageRepository) Get(ctx context.Context, id uuid.UUID) (*channel.Message, error) {
	msg, err := r.cache.Get(ctx, id)
	if err == nil && msg != nil {
		return msg, nil
	}

	msg, err = r.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	_ = r.cache.Set(ctx, msg)

	return msg, nil
}

func (r *CachedMessageRepository) ListAroundByChannelID(
	ctx context.Context,
	channelID, cursorID uuid.UUID,
	beforeLimit, afterLimit int,
) ([]*channel.Message, bool, bool, error) {
	messages, hasMoreBefore, hasMoreAfter, hit, err := r.cache.GetAroundByChannelID(
		ctx,
		channelID,
		cursorID,
		beforeLimit,
		afterLimit,
	)
	if err == nil && hit {
		return messages, hasMoreBefore, hasMoreAfter, nil
	}

	messages, hasMoreBefore, hasMoreAfter, err = r.repo.ListAroundByChannelID(
		ctx,
		channelID,
		cursorID,
		beforeLimit,
		afterLimit,
	)
	if err != nil {
		return nil, false, false, err
	}

	_ = r.cache.SetBatch(ctx, channelID, messages)

	return messages, hasMoreBefore, hasMoreAfter, nil
}

func (r *CachedMessageRepository) ListBeforeByChannelID(
	ctx context.Context,
	channelID, cursorID uuid.UUID,
	limit int,
) ([]*channel.Message, bool, error) {
	messages, hasMoreBefore, hit, err := r.cache.GetBeforeByChannelID(ctx, channelID, cursorID, limit)
	if err == nil && hit {
		return messages, hasMoreBefore, nil
	}

	messages, hasMoreBefore, err = r.repo.ListBeforeByChannelID(ctx, channelID, cursorID, limit)
	if err != nil {
		return nil, false, err
	}

	_ = r.cache.SetBatch(ctx, channelID, messages)

	return messages, hasMoreBefore, nil
}

func (r *CachedMessageRepository) ListAfterByChannelID(
	ctx context.Context,
	channelID, cursorID uuid.UUID,
	limit int,
) ([]*channel.Message, bool, error) {
	messages, hasMoreAfter, hit, err := r.cache.GetAfterByChannelID(ctx, channelID, cursorID, limit)
	if err == nil && hit {
		return messages, hasMoreAfter, nil
	}

	messages, hasMoreAfter, err = r.repo.ListAfterByChannelID(ctx, channelID, cursorID, limit)
	if err != nil {
		return nil, false, err
	}

	_ = r.cache.SetBatch(ctx, channelID, messages)

	return messages, hasMoreAfter, nil
}
