package cache

import (
	"context"
	"time"

	"bonfire-api/internal/redis"

	"github.com/google/uuid"
	redisdriver "github.com/redis/go-redis/v9"
)

var (
	userPendingsTTL = 24 * time.Hour
	userFriendsTTL  = 24 * time.Hour
	userBlocksTTL   = 24 * time.Hour
)

func userIncomingPendingsKey(id uuid.UUID) string {
	return "{user:" + id.String() + "}:incoming_pendings"
}
func userOutgoingPendingsKey(id uuid.UUID) string {
	return "{user:" + id.String() + "}:outgoing_pendings"
}
func userFriendsKey(id uuid.UUID) string        { return "{user:" + id.String() + "}:friends" }
func userIncomingBlocksKey(id uuid.UUID) string { return "{user:" + id.String() + "}:incoming_blocks" }
func userOutgoingBlocksKey(id uuid.UUID) string { return "{user:" + id.String() + "}:outgoing_blocks" }

type RelationCache struct {
	client redisdriver.Cmdable
}

func NewRelationCache(client redisdriver.Cmdable) *RelationCache {
	return &RelationCache{client: client}
}

// ============================================================================
// Incoming Pendings (Requests received by userID)
// ============================================================================

func (c *RelationCache) GetIncomingPendingIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	return getSetIDs(ctx, redis.ScopeRelation, c.client, userIncomingPendingsKey(userID))
}

func (c *RelationCache) SetIncomingPendingIDs(ctx context.Context, userID uuid.UUID, pendingIDs []uuid.UUID) error {
	return setSetIDs(ctx, redis.ScopeRelation, c.client, userIncomingPendingsKey(userID), pendingIDs, userPendingsTTL)
}

func (c *RelationCache) AddIncomingPendingID(ctx context.Context, userID uuid.UUID, senderID uuid.UUID) error {
	return addToSetIDs(ctx, redis.ScopeRelation, c.client, userIncomingPendingsKey(userID), userPendingsTTL, senderID)
}

func (c *RelationCache) RemoveIncomingPendingID(ctx context.Context, userID uuid.UUID, senderID uuid.UUID) error {
	return removeFromSetIDs(ctx, redis.ScopeRelation, c.client, userIncomingPendingsKey(userID), senderID)
}

func (c *RelationCache) DeleteIncomingPendingsIndex(ctx context.Context, userID uuid.UUID) error {
	return deleteSet(ctx, redis.ScopeRelation, c.client, userIncomingPendingsKey(userID))
}

// ============================================================================
// Outgoing Pendings (Requests sent by userID)
// ============================================================================

func (c *RelationCache) GetOutgoingPendingIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	return getSetIDs(ctx, redis.ScopeRelation, c.client, userOutgoingPendingsKey(userID))
}

func (c *RelationCache) SetOutgoingPendingIDs(ctx context.Context, userID uuid.UUID, pendingIDs []uuid.UUID) error {
	return setSetIDs(ctx, redis.ScopeRelation, c.client, userOutgoingPendingsKey(userID), pendingIDs, userPendingsTTL)
}

func (c *RelationCache) AddOutgoingPendingID(ctx context.Context, userID uuid.UUID, targetID uuid.UUID) error {
	return addToSetIDs(ctx, redis.ScopeRelation, c.client, userOutgoingPendingsKey(userID), userPendingsTTL, targetID)
}

func (c *RelationCache) RemoveOutgoingPendingID(ctx context.Context, userID uuid.UUID, targetID uuid.UUID) error {
	return removeFromSetIDs(ctx, redis.ScopeRelation, c.client, userOutgoingPendingsKey(userID), targetID)
}

func (c *RelationCache) DeleteOutgoingPendingsIndex(ctx context.Context, userID uuid.UUID) error {
	return deleteSet(ctx, redis.ScopeRelation, c.client, userOutgoingPendingsKey(userID))
}

// ============================================================================
// Friends
// ============================================================================

func (c *RelationCache) GetFriendIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	return getSetIDs(ctx, redis.ScopeRelation, c.client, userFriendsKey(userID))
}

func (c *RelationCache) SetFriendIDs(ctx context.Context, userID uuid.UUID, friendIDs []uuid.UUID) error {
	return setSetIDs(ctx, redis.ScopeRelation, c.client, userFriendsKey(userID), friendIDs, userFriendsTTL)
}

func (c *RelationCache) AddFriendID(ctx context.Context, userID uuid.UUID, friendID uuid.UUID) error {
	return addToSetIDs(ctx, redis.ScopeRelation, c.client, userFriendsKey(userID), userFriendsTTL, friendID)
}

func (c *RelationCache) RemoveFriendID(ctx context.Context, userID uuid.UUID, friendID uuid.UUID) error {
	return removeFromSetIDs(ctx, redis.ScopeRelation, c.client, userFriendsKey(userID), friendID)
}

func (c *RelationCache) DeleteFriendsIndex(ctx context.Context, userID uuid.UUID) error {
	return deleteSet(ctx, redis.ScopeRelation, c.client, userFriendsKey(userID))
}

// ============================================================================
// Incoming Blocks (Users who have blocked userID)
// ============================================================================

func (c *RelationCache) GetIncomingBlockIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	return getSetIDs(ctx, redis.ScopeRelation, c.client, userIncomingBlocksKey(userID))
}

func (c *RelationCache) SetIncomingBlockIDs(ctx context.Context, userID uuid.UUID, blockIDs []uuid.UUID) error {
	return setSetIDs(ctx, redis.ScopeRelation, c.client, userIncomingBlocksKey(userID), blockIDs, userBlocksTTL)
}

func (c *RelationCache) AddIncomingBlockID(ctx context.Context, userID uuid.UUID, blockerUserID uuid.UUID) error {
	return addToSetIDs(ctx, redis.ScopeRelation, c.client, userIncomingBlocksKey(userID), userBlocksTTL, blockerUserID)
}

func (c *RelationCache) RemoveIncomingBlockID(ctx context.Context, userID uuid.UUID, blockerUserID uuid.UUID) error {
	return removeFromSetIDs(ctx, redis.ScopeRelation, c.client, userIncomingBlocksKey(userID), blockerUserID)
}

func (c *RelationCache) DeleteIncomingBlocksIndex(ctx context.Context, userID uuid.UUID) error {
	return deleteSet(ctx, redis.ScopeRelation, c.client, userIncomingBlocksKey(userID))
}

// ============================================================================
// Outgoing Blocks (Users blocked by userID)
// ============================================================================

func (c *RelationCache) GetOutgoingBlockIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	return getSetIDs(ctx, redis.ScopeRelation, c.client, userOutgoingBlocksKey(userID))
}

func (c *RelationCache) SetOutgoingBlockIDs(ctx context.Context, userID uuid.UUID, blockIDs []uuid.UUID) error {
	return setSetIDs(ctx, redis.ScopeRelation, c.client, userOutgoingBlocksKey(userID), blockIDs, userBlocksTTL)
}

func (c *RelationCache) AddOutgoingBlockID(ctx context.Context, userID uuid.UUID, blockedUserID uuid.UUID) error {
	return addToSetIDs(ctx, redis.ScopeRelation, c.client, userOutgoingBlocksKey(userID), userBlocksTTL, blockedUserID)
}

func (c *RelationCache) RemoveOutgoingBlockID(ctx context.Context, userID uuid.UUID, blockedUserID uuid.UUID) error {
	return removeFromSetIDs(ctx, redis.ScopeRelation, c.client, userOutgoingBlocksKey(userID), blockedUserID)
}

func (c *RelationCache) DeleteOutgoingBlocksIndex(ctx context.Context, userID uuid.UUID) error {
	return deleteSet(ctx, redis.ScopeRelation, c.client, userOutgoingBlocksKey(userID))
}

// ============================================================================
// Multi-Key Composite Operations
// ============================================================================

func (c *RelationCache) AddPendingPair(ctx context.Context, actorID, peerID uuid.UUID) error {
	additions := map[string][]uuid.UUID{
		userOutgoingPendingsKey(actorID): {peerID},
		userIncomingPendingsKey(peerID):  {actorID},
	}
	return addToSetIDsPipelined(ctx, redis.ScopeRelation, c.client, additions, userPendingsTTL)
}

func (c *RelationCache) RemovePendingPair(ctx context.Context, actorID, peerID uuid.UUID) error {
	removals := map[string][]uuid.UUID{
		userOutgoingPendingsKey(actorID): {peerID},
		userIncomingPendingsKey(peerID):  {actorID},
	}
	return removeFromSetIDsPipelined(ctx, redis.ScopeRelation, c.client, removals)
}

func (c *RelationCache) DeletePendingPair(ctx context.Context, actorID, peerID uuid.UUID) error {
	keys := []string{
		userOutgoingPendingsKey(actorID),
		userIncomingPendingsKey(peerID),
	}
	return deleteSetPipelined(ctx, redis.ScopeRelation, c.client, keys)
}

func (c *RelationCache) AddFriendPair(ctx context.Context, userA, userB uuid.UUID) error {
	additions := map[string][]uuid.UUID{
		userFriendsKey(userA): {userB},
		userFriendsKey(userB): {userA},
	}
	return addToSetIDsPipelined(ctx, redis.ScopeRelation, c.client, additions, userFriendsTTL)
}

func (c *RelationCache) RemoveFriendPair(ctx context.Context, userA, userB uuid.UUID) error {
	removals := map[string][]uuid.UUID{
		userFriendsKey(userA): {userB},
		userFriendsKey(userB): {userA},
	}
	return removeFromSetIDsPipelined(ctx, redis.ScopeRelation, c.client, removals)
}

func (c *RelationCache) DeleteFriendsPair(ctx context.Context, userA, userB uuid.UUID) error {
	keys := []string{
		userFriendsKey(userA),
		userFriendsKey(userB),
	}
	return deleteSetPipelined(ctx, redis.ScopeRelation, c.client, keys)
}

func (c *RelationCache) BlockUser(ctx context.Context, blockerID, targetID uuid.UUID) error {
	additions := map[string][]uuid.UUID{
		userOutgoingBlocksKey(blockerID): {targetID},
		userIncomingBlocksKey(targetID):  {blockerID},
	}
	return addToSetIDsPipelined(ctx, redis.ScopeRelation, c.client, additions, userBlocksTTL)
}

func (c *RelationCache) UnblockUser(ctx context.Context, blockerID, targetID uuid.UUID) error {
	removals := map[string][]uuid.UUID{
		userOutgoingBlocksKey(blockerID): {targetID},
		userIncomingBlocksKey(targetID):  {blockerID},
	}
	return removeFromSetIDsPipelined(ctx, redis.ScopeRelation, c.client, removals)
}

func (c *RelationCache) DeleteBlocksPair(ctx context.Context, blockerID, targetID uuid.UUID) error {
	keys := []string{
		userOutgoingBlocksKey(blockerID),
		userIncomingBlocksKey(targetID),
	}
	return deleteSetPipelined(ctx, redis.ScopeRelation, c.client, keys)
}
