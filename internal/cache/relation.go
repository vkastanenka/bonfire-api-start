package cache

import (
	"context"
	"time"

	"bonfire-api/internal/redis"

	"github.com/google/uuid"
	redisdriver "github.com/redis/go-redis/v9"
)

var (
	userPendingsTTL   = 24 * time.Hour
	userFriendsTTL    = 24 * time.Hour
	userBlocksTTL     = 24 * time.Hour
	userBlockedBysTTL = 24 * time.Hour
)

func userPendingsKey(id uuid.UUID) string  { return "{user:" + id.String() + "}:pendings" }
func userFriendsKey(id uuid.UUID) string   { return "{user:" + id.String() + "}:friends" }
func userBlocksKey(id uuid.UUID) string    { return "{user:" + id.String() + "}:blocks" }
func userBlockedByKey(id uuid.UUID) string { return "{user:" + id.String() + "}:blocked_by" }

type RelationCache struct {
	client redisdriver.Cmdable
}

func NewRelationCache(client redisdriver.Cmdable) *RelationCache {
	return &RelationCache{client: client}
}

// ============================================================================
// Pendings
// ============================================================================

func (c *RelationCache) GetUserPendingIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	return getSetIDs(ctx, redis.ScopeRelation, c.client, userPendingsKey(userID))
}

func (c *RelationCache) SetPendingIDs(ctx context.Context, userID uuid.UUID, pendingIDs []uuid.UUID) error {
	return setSetIDs(ctx, redis.ScopeRelation, c.client, userPendingsKey(userID), pendingIDs, userPendingsTTL)
}

func (c *RelationCache) AddPendingID(ctx context.Context, userID uuid.UUID, pendingID uuid.UUID) error {
	return addToSetIDs(ctx, redis.ScopeRelation, c.client, userPendingsKey(userID), userPendingsTTL, pendingID)
}

func (c *RelationCache) RemovePendingID(ctx context.Context, userID uuid.UUID, pendingID uuid.UUID) error {
	return removeFromSetIDs(ctx, redis.ScopeRelation, c.client, userPendingsKey(userID), pendingID)
}

func (c *RelationCache) RemovePendingPair(ctx context.Context, userA, userB uuid.UUID) error {
	removals := map[string][]uuid.UUID{
		userPendingsKey(userA): {userB},
		userPendingsKey(userB): {userA},
	}
	return removeFromSetIDsPipelined(ctx, redis.ScopeRelation, c.client, removals)
}

func (c *RelationCache) DeletePendingsIndex(ctx context.Context, userID uuid.UUID) error {
	return deleteSet(ctx, redis.ScopeRelation, c.client, userPendingsKey(userID))
}

// ============================================================================
// Friends
// ============================================================================

func (c *RelationCache) GetUserFriendIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	return getSetIDs(ctx, redis.ScopeRelation, c.client, userFriendsKey(userID))
}

func (c *RelationCache) SetFriendIDs(ctx context.Context, userID uuid.UUID, friendIDs []uuid.UUID) error {
	return setSetIDs(ctx, redis.ScopeRelation, c.client, userFriendsKey(userID), friendIDs, userFriendsTTL)
}

func (c *RelationCache) AddFriendID(ctx context.Context, userID uuid.UUID, friendID uuid.UUID) error {
	return addToSetIDs(ctx, redis.ScopeRelation, c.client, userFriendsKey(userID), userFriendsTTL, friendID)
}

func (c *RelationCache) AddFriendPair(ctx context.Context, userA, userB uuid.UUID) error {
	additions := map[string][]uuid.UUID{
		userFriendsKey(userA): {userB},
		userFriendsKey(userB): {userA},
	}
	return addToSetIDsPipelined(ctx, redis.ScopeRelation, c.client, additions, userFriendsTTL)
}

func (c *RelationCache) RemoveFriendID(ctx context.Context, userID uuid.UUID, friendID uuid.UUID) error {
	return removeFromSetIDs(ctx, redis.ScopeRelation, c.client, userFriendsKey(userID), friendID)
}

func (c *RelationCache) RemoveFriendPair(ctx context.Context, userA, userB uuid.UUID) error {
	removals := map[string][]uuid.UUID{
		userFriendsKey(userA): {userB},
		userFriendsKey(userB): {userA},
	}
	return removeFromSetIDsPipelined(ctx, redis.ScopeRelation, c.client, removals)
}

func (c *RelationCache) DeleteFriendsIndex(ctx context.Context, userID uuid.UUID) error {
	return deleteSet(ctx, redis.ScopeRelation, c.client, userFriendsKey(userID))
}

// ============================================================================
// Blocks (Users blocked by userID)
// ============================================================================

func (c *RelationCache) GetUserBlockIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	return getSetIDs(ctx, redis.ScopeRelation, c.client, userBlocksKey(userID))
}

func (c *RelationCache) SetBlockIDs(ctx context.Context, userID uuid.UUID, blockIDs []uuid.UUID) error {
	return setSetIDs(ctx, redis.ScopeRelation, c.client, userBlocksKey(userID), blockIDs, userBlocksTTL)
}

func (c *RelationCache) AddBlockID(ctx context.Context, userID uuid.UUID, blockedUserID uuid.UUID) error {
	return addToSetIDs(ctx, redis.ScopeRelation, c.client, userBlocksKey(userID), userBlocksTTL, blockedUserID)
}

func (c *RelationCache) RemoveBlockID(ctx context.Context, userID uuid.UUID, blockedUserID uuid.UUID) error {
	return removeFromSetIDs(ctx, redis.ScopeRelation, c.client, userBlocksKey(userID), blockedUserID)
}

func (c *RelationCache) DeleteBlocksIndex(ctx context.Context, userID uuid.UUID) error {
	return deleteSet(ctx, redis.ScopeRelation, c.client, userBlocksKey(userID))
}

// ============================================================================
// BlockedBy (Users who have blocked userID)
// ============================================================================

func (c *RelationCache) GetUserBlockedByIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	return getSetIDs(ctx, redis.ScopeRelation, c.client, userBlockedByKey(userID))
}

func (c *RelationCache) SetBlockedByIDs(ctx context.Context, userID uuid.UUID, blockedByIDs []uuid.UUID) error {
	return setSetIDs(ctx, redis.ScopeRelation, c.client, userBlockedByKey(userID), blockedByIDs, userBlockedBysTTL)
}

func (c *RelationCache) AddBlockedByID(ctx context.Context, userID uuid.UUID, blockerUserID uuid.UUID) error {
	return addToSetIDs(ctx, redis.ScopeRelation, c.client, userBlockedByKey(userID), userBlockedBysTTL, blockerUserID)
}

func (c *RelationCache) RemoveBlockedByID(ctx context.Context, userID uuid.UUID, blockerUserID uuid.UUID) error {
	return removeFromSetIDs(ctx, redis.ScopeRelation, c.client, userBlockedByKey(userID), blockerUserID)
}

func (c *RelationCache) DeleteBlockedByIndex(ctx context.Context, userID uuid.UUID) error {
	return deleteSet(ctx, redis.ScopeRelation, c.client, userBlockedByKey(userID))
}

// ============================================================================
// Multi-Key Composite Operations
// ============================================================================

func (c *RelationCache) BlockUser(ctx context.Context, blockerID, targetID uuid.UUID) error {
	additions := map[string][]uuid.UUID{
		userBlocksKey(blockerID):   {targetID},
		userBlockedByKey(targetID): {blockerID},
	}
	return addToSetIDsPipelined(ctx, redis.ScopeRelation, c.client, additions, userBlocksTTL)
}

func (c *RelationCache) UnblockUser(ctx context.Context, blockerID, targetID uuid.UUID) error {
	removals := map[string][]uuid.UUID{
		userBlocksKey(blockerID):   {targetID},
		userBlockedByKey(targetID): {blockerID},
	}
	return removeFromSetIDsPipelined(ctx, redis.ScopeRelation, c.client, removals)
}
