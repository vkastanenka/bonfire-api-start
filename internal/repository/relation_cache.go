package repository

import (
	"context"
	"log/slog"

	"bonfire-api/internal/relation"

	"github.com/google/uuid"
)

const maxRelationFetchLimit = 1000

type CachedRelationRepository struct {
	cache UserCache
	repo  *RelationRepository
}

func NewCachedRelationRepository(cache UserCache, repo *RelationRepository) *CachedRelationRepository {
	return &CachedRelationRepository{
		cache: cache,
		repo:  repo,
	}
}

func (r *CachedRelationRepository) GetPendingIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	cachedIDs, err := r.cache.GetPendingIDs(ctx, userID)
	if err == nil && cachedIDs != nil {
		return cachedIDs, nil
	}

	rels, err := r.repo.ListIncomingPendingByUserID(ctx, userID, maxRelationFetchLimit)
	if err != nil {
		return nil, err
	}

	pendingIDs := extractPeerIDs(rels, userID)

	if err := r.cache.SetPendingIDs(ctx, userID, pendingIDs); err != nil {
		slog.WarnContext(ctx, "failed to backfill pending IDs cache", "user_id", userID.String(), "err", err)
	}

	return pendingIDs, nil
}

func (r *CachedRelationRepository) GetFriends(ctx context.Context, userID uuid.UUID) (map[uuid.UUID]uuid.UUID, error) {
	friendsMap, err := r.cache.GetFriends(ctx, userID)
	if err == nil && friendsMap != nil {
		return friendsMap, nil
	}

	rels, err := r.repo.ListFriendsByUserID(ctx, userID, maxRelationFetchLimit)
	if err != nil {
		return nil, err
	}

	friendsMap = make(map[uuid.UUID]uuid.UUID, len(rels))
	for _, rel := range rels {
		var peerID uuid.UUID
		if rel.User1ID == userID {
			peerID = rel.User2ID
		} else {
			peerID = rel.User1ID
		}

		if rel.ChannelID != nil {
			friendsMap[peerID] = *rel.ChannelID
		}
	}

	if err := r.cache.SetFriends(ctx, userID, friendsMap); err != nil {
		slog.WarnContext(ctx, "failed to backfill friends cache", "user_id", userID.String(), "err", err)
	}

	return friendsMap, nil
}

func (r *CachedRelationRepository) GetBlocklistIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	cachedIDs, err := r.cache.GetBlocklistIDs(ctx, userID)
	if err == nil && cachedIDs != nil {
		return cachedIDs, nil
	}

	rels, err := r.repo.ListOutgoingBlocksByUserID(ctx, userID, maxRelationFetchLimit)
	if err != nil {
		return nil, err
	}

	blockedIDs := extractPeerIDs(rels, userID)

	if err := r.cache.SetBlocklistIDs(ctx, userID, blockedIDs); err != nil {
		slog.WarnContext(ctx, "failed to backfill blocklist IDs cache", "user_id", userID.String(), "err", err)
	}

	return blockedIDs, nil
}

func (r *CachedRelationRepository) GetBlockedByIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	cachedIDs, err := r.cache.GetBlockedByIDs(ctx, userID)
	if err == nil && cachedIDs != nil {
		return cachedIDs, nil
	}

	rels, err := r.repo.ListIncomingBlocksByUserID(ctx, userID, maxRelationFetchLimit)
	if err != nil {
		return nil, err
	}

	blockerIDs := extractPeerIDs(rels, userID)

	if err := r.cache.SetBlockedByIDs(ctx, userID, blockerIDs); err != nil {
		slog.WarnContext(ctx, "failed to backfill blocked_by IDs cache", "user_id", userID.String(), "err", err)
	}

	return blockerIDs, nil
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
