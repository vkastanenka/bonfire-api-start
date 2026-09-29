package repository

import (
	"context"
	"log/slog"
	"time"

	"bonfire-api/internal/channel"
	"bonfire-api/internal/pkg/errs"
	"bonfire-api/internal/pkg/helpers"

	"github.com/google/uuid"
)

type CachedMemberRepository struct {
	cache        MemberCache
	channelCache ChannelCache
	repo         *MemberRepository
}

func NewCachedMemberRepository(
	cache MemberCache,
	channelCache ChannelCache,
	repo *MemberRepository,
) *CachedMemberRepository {
	return &CachedMemberRepository{
		cache:        cache,
		channelCache: channelCache,
		repo:         repo,
	}
}

func (r *CachedMemberRepository) Get(ctx context.Context, channelID, userID uuid.UUID) (*channel.Member, error) {
	mem, err := r.cache.Get(ctx, channelID, userID)
	if err == nil && mem != nil {
		return mem, nil
	}

	if err != nil {
		slog.WarnContext(ctx, "redis channel member cache read failure, falling back to database",
			slog.String("channel_id", channelID.String()),
			slog.String("user_id", userID.String()),
			slog.Any("error", err),
		)
	}

	mem, err = r.repo.Get(ctx, channelID, userID)
	if err != nil {
		return nil, err
	}

	cacheCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
	defer cancel()

	if setErr := r.cache.Add(cacheCtx, channelID, []*channel.Member{mem}); setErr != nil {
		slog.WarnContext(cacheCtx, "failed to populate channel member cache after database read",
			slog.String("channel_id", channelID.String()),
			slog.String("user_id", userID.String()),
			slog.Any("error", setErr),
		)
	}

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
	if !ok {
		return nil, errs.NotFound("Channel members not found.").
			Reason("CHANNEL_MEMBERS_NOT_FOUND").
			Meta("channelID", channelID.String())
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

	found, missing, err := r.cache.GetBatchByChannelIDs(ctx, channelIDs)
	if err != nil {
		slog.WarnContext(ctx, "redis channel members batch cache read failure, falling back to database",
			slog.Int("requested_count", len(channelIDs)),
			slog.Any("error", err),
		)
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
		cacheCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
		defer cancel()

		if setErr := r.cache.SetBatchByChannelIDs(cacheCtx, dbMembersMap); setErr != nil {
			slog.WarnContext(cacheCtx, "failed to populate channel members batch cache after database read",
				slog.Int("missing_count", len(missing)),
				slog.Any("error", setErr),
			)
		}
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
	if userID == uuid.Nil {
		return []*channel.Member{}, nil
	}

	channelIDs, hit, err := r.channelCache.GetUserChannelIDs(ctx, userID)
	if err != nil {
		slog.WarnContext(ctx, "redis user channel index read failed, falling back to db",
			slog.String("user_id", userID.String()),
			slog.Any("error", err),
		)
	} else if hit {
		if len(channelIDs) == 0 {
			return []*channel.Member{}, nil
		}

		targetIDs := channelIDs
		if limit > 0 && len(targetIDs) > limit {
			targetIDs = targetIDs[:limit]
		}

		foundMap, missingIDs, err := r.cache.GetBatchByChannelIDs(ctx, targetIDs)
		if err == nil && len(missingIDs) == 0 {
			result := make([]*channel.Member, 0, len(targetIDs))
			for _, chID := range targetIDs {
				members := foundMap[chID]
				for _, m := range members {
					if m != nil && m.UserID == userID {
						result = append(result, m)
						break
					}
				}
			}
			return result, nil
		}

		if err != nil {
			slog.WarnContext(ctx, "redis member batch read failed, falling back to db",
				slog.String("user_id", userID.String()),
				slog.Any("error", err),
			)
		}
	}

	dbMembers, err := r.repo.ListVisibleByUserID(ctx, userID, limit)
	if err != nil {
		return nil, err
	}

	if len(dbMembers) == 0 {
		return dbMembers, nil
	}

	cacheCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
	defer cancel()

	if backfillErr := r.cache.SetUserMembers(cacheCtx, userID, dbMembers); backfillErr != nil {
		slog.WarnContext(cacheCtx, "failed to backfill user members cache",
			slog.String("user_id", userID.String()),
			slog.Any("error", backfillErr),
		)
	}

	return dbMembers, nil
}
