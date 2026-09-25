package repository

import (
	"context"
	"log/slog"
	"time"

	"bonfire-api/internal/relation"

	"github.com/google/uuid"
)

const maxRelationFetchLimit = 1000

type CachedRelationRepository struct {
	cache RelationCache
	repo  *RelationRepository
}

func NewCachedRelationRepository(cache RelationCache, repo *RelationRepository) *CachedRelationRepository {
	return &CachedRelationRepository{cache: cache, repo: repo}
}

func (r *CachedRelationRepository) GetPendingIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	ids, err := r.cache.GetUserPendingIDs(ctx, userID)
	if err == nil && ids != nil {
		return ids, nil
	}

	if err != nil {
		slog.WarnContext(ctx, "redis user pending IDs cache read failure, falling back to database",
			slog.String("user_id", userID.String()),
			slog.Any("error", err),
		)
	}

	rels, err := r.repo.ListIncomingPendingByUserID(ctx, userID, maxRelationFetchLimit)
	if err != nil {
		return nil, err
	}

	pendingIDs := extractPeerIDs(rels, userID)

	cacheCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
	defer cancel()

	if setErr := r.cache.SetPendingIDs(cacheCtx, userID, pendingIDs); setErr != nil {
		slog.WarnContext(cacheCtx, "failed to backfill pending IDs cache after database read",
			slog.String("user_id", userID.String()),
			slog.Int("count", len(pendingIDs)),
			slog.Any("error", setErr),
		)
	}

	return pendingIDs, nil
}

func (r *CachedRelationRepository) GetFriendIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	ids, err := r.cache.GetUserFriendIDs(ctx, userID)
	if err == nil && ids != nil {
		return ids, nil
	}

	if err != nil {
		slog.WarnContext(ctx, "redis user friend IDs cache read failure, falling back to database",
			slog.String("user_id", userID.String()),
			slog.Any("error", err),
		)
	}

	rels, err := r.repo.ListFriendsByUserID(ctx, userID, maxRelationFetchLimit)
	if err != nil {
		return nil, err
	}

	friendIDs := extractPeerIDs(rels, userID)

	cacheCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
	defer cancel()

	if setErr := r.cache.SetFriendIDs(cacheCtx, userID, friendIDs); setErr != nil {
		slog.WarnContext(cacheCtx, "failed to backfill friend IDs cache after database read",
			slog.String("user_id", userID.String()),
			slog.Int("count", len(friendIDs)),
			slog.Any("error", setErr),
		)
	}

	return friendIDs, nil
}

func (r *CachedRelationRepository) GetBlockIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	ids, err := r.cache.GetUserBlockIDs(ctx, userID)
	if err == nil && ids != nil {
		return ids, nil
	}

	if err != nil {
		slog.WarnContext(ctx, "redis user block IDs cache read failure, falling back to database",
			slog.String("user_id", userID.String()),
			slog.Any("error", err),
		)
	}

	rels, err := r.repo.ListOutgoingBlocksByUserID(ctx, userID, maxRelationFetchLimit)
	if err != nil {
		return nil, err
	}

	blockIDs := extractPeerIDs(rels, userID)

	cacheCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
	defer cancel()

	if setErr := r.cache.SetBlockIDs(cacheCtx, userID, blockIDs); setErr != nil {
		slog.WarnContext(cacheCtx, "failed to backfill block IDs cache after database read",
			slog.String("user_id", userID.String()),
			slog.Int("count", len(blockIDs)),
			slog.Any("error", setErr),
		)
	}

	return blockIDs, nil
}

func (r *CachedRelationRepository) GetBlockedByIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	ids, err := r.cache.GetUserBlockedByIDs(ctx, userID)
	if err == nil && ids != nil {
		return ids, nil
	}

	if err != nil {
		slog.WarnContext(ctx, "redis user blocked_by IDs cache read failure, falling back to database",
			slog.String("user_id", userID.String()),
			slog.Any("error", err),
		)
	}

	rels, err := r.repo.ListIncomingBlocksByUserID(ctx, userID, maxRelationFetchLimit)
	if err != nil {
		return nil, err
	}

	blockedByIDs := extractPeerIDs(rels, userID)

	cacheCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
	defer cancel()

	if setErr := r.cache.SetBlockedByIDs(cacheCtx, userID, blockedByIDs); setErr != nil {
		slog.WarnContext(cacheCtx, "failed to backfill blocked_by IDs cache after database read",
			slog.String("user_id", userID.String()),
			slog.Int("count", len(blockedByIDs)),
			slog.Any("error", setErr),
		)
	}

	return blockedByIDs, nil
}

func extractPeerIDs(rels []*relation.Relation, subjectID uuid.UUID) []uuid.UUID {
	peerIDs := make([]uuid.UUID, 0, len(rels))
	for _, rel := range rels {
		if rel.User1ID == subjectID {
			peerIDs = append(peerIDs, rel.User2ID)
		} else {
			peerIDs = append(peerIDs, rel.User1ID)
		}
	}
	return peerIDs
}
