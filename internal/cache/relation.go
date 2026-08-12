package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"bonfire-api/internal/fields"
	"bonfire-api/internal/redis"
	"bonfire-api/internal/relation"

	"github.com/google/uuid"
)

type Relation struct {
	Type      relation.Type `json:"type"`
	ActorID   uuid.UUID     `json:"actor_id"`
	ChannelID uuid.UUID     `json:"channel_id"`
}

type RelationCache struct {
	store *redis.Store
	ttl   time.Duration
}

func NewRelationCache(store *redis.Store, ttl time.Duration) *RelationCache {
	return &RelationCache{
		store: store.WithScope(redis.ScopeRelation),
		ttl:   ttl,
	}
}

func (c *RelationCache) relationKey(u1, u2 fields.ID) string {
	return fmt.Sprintf("relation:%s:%s", u1.String(), u2.String())
}

func (c *RelationCache) userRelationsKey(userID fields.ID) string {
	return fmt.Sprintf("user:%s:relations", userID.String())
}

// Get performs an O(1) direct lookup for the canonical relationship edge between two users.
func (c *RelationCache) Get(ctx context.Context, u1, u2 fields.ID) (*Relation, error) {
	k := c.relationKey(u1, u2)

	var raw string
	err := c.store.Get(ctx, k, &raw)
	if err != nil {
		return nil, c.store.Err(err)
	}
	if raw == "" {
		return nil, nil
	}

	var edge Relation
	if err := json.Unmarshal([]byte(raw), &edge); err != nil {
		return nil, err
	}

	return &edge, nil
}

// GetUserRelations fetches all relationship mappings (PeerID -> RelationType) for a user.
func (c *RelationCache) GetUserRelations(ctx context.Context, userID fields.ID) (map[uuid.UUID]relation.Type, error) {
	k := c.userRelationsKey(userID)

	var rawMap map[string]string
	if err := c.store.HGetAll(ctx, k, &rawMap); err != nil {
		return nil, c.store.Err(err)
	}

	if len(rawMap) == 0 {
		return nil, nil
	}

	res := make(map[uuid.UUID]relation.Type, len(rawMap))
	for peerStr, rawType := range rawMap {
		peerID, err := uuid.Parse(peerStr)
		if err != nil {
			continue
		}

		val, err := strconv.Atoi(rawType)
		if err != nil {
			continue
		}

		res[peerID] = relation.Type(val)
	}

	return res, nil
}

// TransitionRelation performs an atomic state update across the canonical edge and both user hashes.
func (c *RelationCache) TransitionRelation(
	ctx context.Context,
	u1, u2 fields.ID,
	rel Relation,
) error {
	edgeKey := c.relationKey(u1, u2)
	u1Key := c.userRelationsKey(u1)
	u2Key := c.userRelationsKey(u2)

	rawEdge, err := json.Marshal(rel)
	if err != nil {
		return err
	}

	return c.store.ExecPipeline(ctx, func(pipeCtx context.Context) error {
		// 1. Set source-of-truth canonical edge (contains ActorID, ChannelID, and Type)
		if err := c.store.Set(pipeCtx, edgeKey, rawEdge, c.ttl); err != nil {
			return err
		}

		// 2. Update relationship state in u1's index hash
		if err := c.store.HSet(pipeCtx, u1Key, u2.String(), rel.Type.Int16()); err != nil {
			return err
		}
		if err := c.store.Expire(pipeCtx, u1Key, c.ttl); err != nil {
			return err
		}

		// 3. Update relationship state in u2's index hash
		if err := c.store.HSet(pipeCtx, u2Key, u1.String(), rel.Type.Int16()); err != nil {
			return err
		}
		return c.store.Expire(pipeCtx, u2Key, c.ttl)
	})
}

// RemoveRelation deletes the canonical edge and removes peer entries from both user hashes.
func (c *RelationCache) RemoveRelation(ctx context.Context, u1, u2 fields.ID) error {
	edgeKey := c.relationKey(u1, u2)
	u1Key := c.userRelationsKey(u1)
	u2Key := c.userRelationsKey(u2)

	return c.store.ExecPipeline(ctx, func(pipeCtx context.Context) error {
		if err := c.store.Delete(pipeCtx, edgeKey); err != nil {
			return err
		}
		if err := c.store.HDel(pipeCtx, u1Key, u2.String()); err != nil {
			return err
		}
		return c.store.HDel(pipeCtx, u2Key, u1.String())
	})
}

// SetUserRelations backfills a user's entire relation hash from a DB fetch on cache miss.
func (c *RelationCache) SetUserRelations(ctx context.Context, userID fields.ID, relations map[uuid.UUID]relation.Type) error {
	k := c.userRelationsKey(userID)

	if len(relations) == 0 {
		return c.store.Delete(ctx, k)
	}

	fieldsMap := make(map[string]interface{}, len(relations))
	for peerID, relType := range relations {
		fieldsMap[peerID.String()] = int16(relType)
	}

	return c.store.ExecPipeline(ctx, func(pipeCtx context.Context) error {
		if err := c.store.Delete(pipeCtx, k); err != nil {
			return err
		}
		if err := c.store.HMSet(pipeCtx, k, fieldsMap); err != nil {
			return err
		}
		return c.store.Expire(pipeCtx, k, c.ttl)
	})
}

// InvalidateUser evicts a user's entire relations hash.
func (c *RelationCache) InvalidateUser(ctx context.Context, userID fields.ID) error {
	return c.store.Delete(ctx, c.userRelationsKey(userID))
}