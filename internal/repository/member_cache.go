package repository

import (
	"context"

	"bonfire-api/internal/channel"
	"bonfire-api/internal/errs"
	"bonfire-api/internal/helpers"

	"github.com/google/uuid"
)

type CachedMemberRepository struct {
	cache     ChannelCache
	userCache UserCache
	repo      *MemberRepository
}

func NewCachedMemberRepository(
	cache ChannelCache,
	userCache UserCache,
	repo *MemberRepository,
) *CachedMemberRepository {
	return &CachedMemberRepository{
		cache:     cache,
		userCache: userCache,
		repo:      repo,
	}
}

func (r *CachedMemberRepository) Get(ctx context.Context, channelID, userID uuid.UUID) (*channel.Member, error) {
	mem, err := r.cache.GetMember(ctx, channelID, userID)
	if err == nil && mem != nil {
		return mem, nil
	}

	mem, err = r.repo.Get(ctx, channelID, userID)
	if err != nil {
		return nil, err
	}

	_ = r.cache.AddMembers(ctx, channelID, []*channel.Member{mem})

	return mem, nil
}

func (r *CachedMemberRepository) GetBatchByChannelID(
	ctx context.Context,
	channelID uuid.UUID,
) ([]*channel.Member, error) {
	memberMap, err := r.GetBatchByChannelIDs(ctx, []uuid.UUID{channelID})
	if err != nil {
		return nil, err
	}

	members, ok := memberMap[channelID]
	if !ok || len(members) == 0 {
		return nil, errs.NotFound("entity not found")
	}

	return members, nil
}

func (r *CachedMemberRepository) GetBatchByChannelIDs(
	ctx context.Context,
	channelIDs []uuid.UUID,
) (map[uuid.UUID][]*channel.Member, error) {
	if len(channelIDs) == 0 {
		return make(map[uuid.UUID][]*channel.Member), nil
	}

	found, missing, err := r.cache.GetBatchMembersByChannelIDs(ctx, channelIDs)
	if err != nil {
		missing = channelIDs
		found = make(map[uuid.UUID][]*channel.Member)
	}

	if len(missing) == 0 {
		return found, nil
	}

	missing = helpers.DedupeIDs(missing)

	dbMembersMap, err := r.repo.GetBatchByChannelIDs(ctx, missing)
	if err != nil {
		return nil, err
	}

	if len(dbMembersMap) > 0 {
		_ = r.cache.SetBatchMembers(ctx, dbMembersMap)
	}

	for channelID, members := range dbMembersMap {
		found[channelID] = members
	}

	return found, nil
}

func (r *CachedMemberRepository) ListVisibleByUserID(
	ctx context.Context,
	userID uuid.UUID,
	limit int,
) ([]*channel.Member, error) {
	members, hit, err := r.userCache.GetVisibleMembersByUserID(ctx, userID, limit)
	if err == nil && hit {
		return members, nil
	}

	dbMembers, err := r.repo.ListVisibleByUserID(ctx, userID, limit)
	if err != nil {
		return nil, err
	}

	if len(dbMembers) == 0 {
		return dbMembers, nil
	}

	channelIDs := make([]uuid.UUID, len(dbMembers))
	for i, m := range dbMembers {
		channelIDs[i] = m.ChannelID
	}

	_ = r.userCache.SetChannelIDs(ctx, userID, channelIDs)

	return dbMembers, nil
}
