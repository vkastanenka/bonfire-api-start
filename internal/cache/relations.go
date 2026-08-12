package cache

import (
	"context"
	"encoding/json"
	"time"

	"bonfire-api/internal/fields"
	"bonfire-api/internal/redis"
	"bonfire-api/internal/relation"

	"github.com/google/uuid"
)

type Relation struct {
	PeerID    uuid.UUID `json:"peer_id"`
	ActorID   uuid.UUID `json:"actor_id"`
	ChannelID uuid.UUID `json:"channel_id"`
}

type RelationsCache struct {
	store *redis.Store
	ttl   time.Duration
}

func NewRelationsCache(store *redis.Store, ttl time.Duration) *RelationsCache {
	return &RelationsCache{
		store: store.WithScope(redis.ScopeRelation),
		ttl:   ttl,
	}
}

func (c *RelationsCache) key(userID fields.ID, relType relation.Type) string {
	return "user:" + userID.String() + ":relations:" + relType.String()
}

func (c *RelationsCache) Get(ctx context.Context, userID fields.ID, relType relation.Type) ([]Relation, error) {
	k := c.key(userID, relType)

	var rawMap map[string]string
	if err := c.store.HGetAll(ctx, k, &rawMap); err != nil {
		return nil, c.store.Err(err)
	}

	if len(rawMap) == 0 {
		return nil, nil
	}

	relations := make([]Relation, 0, len(rawMap))
	for _, raw := range rawMap {
		var rel Relation
		if err := json.Unmarshal([]byte(raw), &rel); err != nil {
			continue
		}
		relations = append(relations, rel)
	}

	return relations, nil
}

func (c *RelationsCache) GetPeer(ctx context.Context, userID fields.ID, relType relation.Type, peerID uuid.UUID) (*Relation, error) {
	k := c.key(userID, relType)

	var raw string
	err := c.store.HGet(ctx, k, peerID.String(), &raw)
	if err != nil {
		return nil, c.store.Err(err)
	}
	if raw == "" {
		return nil, nil
	}

	var rel Relation
	if err := json.Unmarshal([]byte(raw), &rel); err != nil {
		return nil, err
	}

	return &rel, nil
}

func (c *RelationsCache) Set(ctx context.Context, userID fields.ID, relType relation.Type, relations []Relation) error {
	k := c.key(userID, relType)

	if len(relations) == 0 {
		return c.store.Delete(ctx, k)
	}

	fieldsMap := make(map[string]interface{}, len(relations))
	for _, rel := range relations {
		data, err := json.Marshal(rel)
		if err != nil {
			return err
		}
		fieldsMap[rel.PeerID.String()] = data
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

func (c *RelationsCache) Add(ctx context.Context, userID fields.ID, relType relation.Type, rel Relation) error {
	k := c.key(userID, relType)
	data, err := json.Marshal(rel)
	if err != nil {
		return err
	}

	return c.store.ExecPipeline(ctx, func(pipeCtx context.Context) error {
		if err := c.store.HSet(pipeCtx, k, rel.PeerID.String(), data); err != nil {
			return err
		}
		return c.store.Expire(pipeCtx, k, c.ttl)
	})
}

func (c *RelationsCache) Remove(ctx context.Context, userID fields.ID, relType relation.Type, peerIDs ...uuid.UUID) error {
	if len(peerIDs) == 0 {
		return nil
	}

	k := c.key(userID, relType)
	fields := make([]string, len(peerIDs))
	for i, id := range peerIDs {
		fields[i] = id.String()
	}

	return c.store.HDel(ctx, k, fields...)
}

func (c *RelationsCache) Delete(ctx context.Context, userID fields.ID, types ...relation.Type) error {
	if len(types) == 0 {
		return nil
	}

	keys := make([]string, len(types))
	for i, relType := range types {
		keys[i] = c.key(userID, relType)
	}

	return c.store.Delete(ctx, keys...)
}
