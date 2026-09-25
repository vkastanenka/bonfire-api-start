package repository

import (
	"context"
	"log/slog"
	"time"

	"bonfire-api/internal/relation"

	"github.com/google/uuid"
)

type CachedRelationRepository struct {
	cache RelationCache
	repo  *RelationRepository
}

func NewCachedRelationRepository(cache RelationCache, repo *RelationRepository) *CachedRelationRepository {
	return &CachedRelationRepository{cache: cache, repo: repo}
}

func (r *CachedRelationRepository) GetIncomingPendingIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	ids, err := r.cache.GetIncomingPendingIDs(ctx, userID)
	if err == nil && ids != nil {
		return ids, nil
	}

	if err != nil {
		slog.WarnContext(ctx, "redis incoming pending IDs cache read failure, falling back to database",
			slog.String("user_id", userID.String()),
			slog.Any("error", err),
		)
	}

	rels, err := r.repo.ListIncomingPendingsByUserID(ctx, userID, relation.MaxPeerTypeLimit)
	if err != nil {
		return nil, err
	}

	pendingIDs := extractPeerIDs(rels, userID)

	r.backfillCache(ctx, "incoming pending IDs", userID, len(pendingIDs), func(cacheCtx context.Context) error {
		return r.cache.SetIncomingPendingIDs(cacheCtx, userID, pendingIDs)
	})

	return pendingIDs, nil
}

func (r *CachedRelationRepository) GetOutgoingPendingIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	ids, err := r.cache.GetOutgoingPendingIDs(ctx, userID)
	if err == nil && ids != nil {
		return ids, nil
	}

	if err != nil {
		slog.WarnContext(ctx, "redis outgoing pending IDs cache read failure, falling back to database",
			slog.String("user_id", userID.String()),
			slog.Any("error", err),
		)
	}

	rels, err := r.repo.ListOutgoingPendingsByUserID(ctx, userID, relation.MaxPeerTypeLimit)
	if err != nil {
		return nil, err
	}

	pendingIDs := extractPeerIDs(rels, userID)

	r.backfillCache(ctx, "outgoing pending IDs", userID, len(pendingIDs), func(cacheCtx context.Context) error {
		return r.cache.SetOutgoingPendingIDs(cacheCtx, userID, pendingIDs)
	})

	return pendingIDs, nil
}

func (r *CachedRelationRepository) GetFriendIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	ids, err := r.cache.GetFriendIDs(ctx, userID)
	if err == nil && ids != nil {
		return ids, nil
	}

	if err != nil {
		slog.WarnContext(ctx, "redis friend IDs cache read failure, falling back to database",
			slog.String("user_id", userID.String()),
			slog.Any("error", err),
		)
	}

	rels, err := r.repo.ListFriendsByUserID(ctx, userID, relation.MaxPeerTypeLimit)
	if err != nil {
		return nil, err
	}

	friendIDs := extractPeerIDs(rels, userID)

	r.backfillCache(ctx, "friend IDs", userID, len(friendIDs), func(cacheCtx context.Context) error {
		return r.cache.SetFriendIDs(cacheCtx, userID, friendIDs)
	})

	return friendIDs, nil
}

func (r *CachedRelationRepository) GetIncomingBlockIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	ids, err := r.cache.GetIncomingBlockIDs(ctx, userID)
	if err == nil && ids != nil {
		return ids, nil
	}

	if err != nil {
		slog.WarnContext(ctx, "redis incoming block IDs cache read failure, falling back to database",
			slog.String("user_id", userID.String()),
			slog.Any("error", err),
		)
	}

	rels, err := r.repo.ListIncomingBlocksByUserID(ctx, userID, relation.MaxPeerTypeLimit)
	if err != nil {
		return nil, err
	}

	blockedByIDs := extractPeerIDs(rels, userID)

	r.backfillCache(ctx, "incoming block IDs", userID, len(blockedByIDs), func(cacheCtx context.Context) error {
		return r.cache.SetIncomingBlockIDs(cacheCtx, userID, blockedByIDs)
	})

	return blockedByIDs, nil
}

func (r *CachedRelationRepository) GetOutgoingBlockIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	ids, err := r.cache.GetOutgoingBlockIDs(ctx, userID)
	if err == nil && ids != nil {
		return ids, nil
	}

	if err != nil {
		slog.WarnContext(ctx, "redis outgoing block IDs cache read failure, falling back to database",
			slog.String("user_id", userID.String()),
			slog.Any("error", err),
		)
	}

	rels, err := r.repo.ListOutgoingBlocksByUserID(ctx, userID, relation.MaxPeerTypeLimit)
	if err != nil {
		return nil, err
	}

	blockIDs := extractPeerIDs(rels, userID)

	r.backfillCache(ctx, "outgoing block IDs", userID, len(blockIDs), func(cacheCtx context.Context) error {
		return r.cache.SetOutgoingBlockIDs(cacheCtx, userID, blockIDs)
	})

	return blockIDs, nil
}

func (r *CachedRelationRepository) backfillCache(
	ctx context.Context,
	entityName string,
	userID uuid.UUID,
	count int,
	setFn func(cacheCtx context.Context) error,
) {
	cacheCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
	defer cancel()

	if err := setFn(cacheCtx); err != nil {
		slog.WarnContext(cacheCtx, "failed to backfill cache after database read",
			slog.String("entity", entityName),
			slog.String("user_id", userID.String()),
			slog.Int("count", count),
			slog.Any("error", err),
		)
	}
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
