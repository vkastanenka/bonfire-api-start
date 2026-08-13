package repository

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"

	"bonfire-api/internal/db"
	"bonfire-api/internal/errs"
	"bonfire-api/internal/fields"
	"bonfire-api/internal/redis"
	"bonfire-api/internal/relation"

	"github.com/google/uuid"
	"golang.org/x/sync/singleflight"
)

const MaxType = 1000

type Cache interface {
	Get(ctx context.Context, u1, u2 fields.ID) (*relation.Relation, error)
	GetBatch(ctx context.Context, u1 fields.ID, peers []fields.ID) (map[fields.ID]*relation.Relation, []fields.ID, error)
	GetUserRelations(ctx context.Context, userID fields.ID, relType relation.Type) (map[uuid.UUID]relation.Type, error)
	TransitionRelation(ctx context.Context, u1, u2 fields.ID, rel *relation.Relation) error
	RemoveRelation(ctx context.Context, u1, u2 fields.ID) error
	SetUserRelations(ctx context.Context, userID fields.ID, relations map[uuid.UUID]relation.Type) error
	InvalidateUser(ctx context.Context, userID fields.ID) error
}

type RelationRepository struct {
	store *db.Store
	cache Cache
	sf    singleflight.Group
}

func NewRelationRepository(store *db.Store, cache Cache) *RelationRepository {
	return &RelationRepository{
		store: store.WithEntity(db.EntityRelation),
		cache: cache,
	}
}

func (r *RelationRepository) DeleteByUser(ctx context.Context, user1ID, user2ID, actorID fields.ID) error {
	err := r.store.RelationDeleteByUser(ctx, db.RelationDeleteByUserParams{
		User1ID: db.ToUUID(user1ID.UUID()),
		User2ID: db.ToUUID(user2ID.UUID()),
		ActorID: db.ToUUID(actorID.UUID()),
	})
	if err != nil {
		return r.store.Err(err)
	}

	return nil
}

func (r *RelationRepository) Get(ctx context.Context, user1ID, user2ID fields.ID) (*relation.Relation, error) {
	row, err := r.store.RelationGet(ctx, db.RelationGetParams{
		User1ID: db.ToUUID(user1ID.UUID()),
		User2ID: db.ToUUID(user2ID.UUID()),
	})
	if err != nil {
		return nil, r.store.Err(err)
	}

	return relationFromRow(row)
}

// GetCached returns a canonical edge between u1 and u2 with Cache-Aside support.
func (r *RelationRepository) GetCached(ctx context.Context, u1, u2 fields.ID) (*relation.Relation, error) {
	rel, err := r.cache.Get(ctx, u1, u2)
	if err == nil && rel != nil {
		return rel, nil
	}

	if err != nil && !errors.Is(err, redis.ErrCacheMiss) {
		slog.WarnContext(ctx, "relation edge cache read failed, falling back to database",
			"u1", u1.String(),
			"u2", u2.String(),
			"error", err,
			"scope", redis.ScopeRelation,
		)
	}

	sfKey := fmt.Sprintf("get_cached_edge:%s:%s", u1.String(), u2.String())
	sfCtx := context.WithoutCancel(ctx)

	val, err, _ := r.sf.Do(sfKey, func() (any, error) {
		dbRel, repoErr := r.Get(sfCtx, u1, u2)
		if repoErr != nil {
			return nil, repoErr
		}

		if cacheErr := r.cache.TransitionRelation(sfCtx, u1, u2, dbRel); cacheErr != nil {
			slog.WarnContext(sfCtx, "failed to backfill relation edge cache",
				"u1", u1.String(),
				"u2", u2.String(),
				"error", cacheErr,
				"scope", redis.ScopeRelation,
			)
		}

		return dbRel, nil
	})

	if err != nil {
		return nil, err
	}

	return val.(*relation.Relation), nil
}

// GetCachedBatch fetches relation edges for multiple peer IDs, utilizing cache read-through for missing edges.
func (r *RelationRepository) GetCachedBatch(
	ctx context.Context,
	u1 fields.ID,
	peers []fields.ID,
) (map[fields.ID]*relation.Relation, error) {
	if len(peers) == 0 {
		return map[fields.ID]*relation.Relation{}, nil
	}

	found, missing, err := r.cache.GetBatch(ctx, u1, peers)
	if err != nil {
		slog.WarnContext(ctx, "batch relation edge cache read failed, falling back to db for all peers",
			"u1", u1.String(),
			"error", err,
			"scope", redis.ScopeRelation,
		)
		missing = peers
		found = make(map[fields.ID]*relation.Relation, len(peers))
	}

	if len(missing) == 0 {
		return found, nil
	}

	// Fetch missing edges directly using GetCached to leverage singleflight & cache backfills
	for _, peerID := range missing {
		rel, err := r.GetCached(ctx, u1, peerID)
		if err != nil {
			if errors.Is(err, db.ErrNotFound) {
				continue
			}
			return nil, err
		}
		found[peerID] = rel
	}

	return found, nil
}

func (r *RelationRepository) GetForUpdate(ctx context.Context, user1ID, user2ID fields.ID) (*relation.Relation, error) {
	row, err := r.store.RelationGetForUpdate(ctx, db.RelationGetForUpdateParams{
		User1ID: db.ToUUID(user1ID.UUID()),
		User2ID: db.ToUUID(user2ID.UUID()),
	})
	if err != nil {
		return nil, r.store.Err(err)
	}

	return relationFromRow(row)
}

func (r *RelationRepository) GetByChannel(ctx context.Context, channelID fields.ID) (*relation.Relation, error) {
	row, err := r.store.RelationGetByChannel(ctx, db.ToUUID(channelID.UUID()))
	if err != nil {
		return nil, r.store.Err(err)
	}

	return relationFromRow(row)
}

func (r *RelationRepository) ListTypeByUser(
	ctx context.Context,
	userID fields.ID,
	relType relation.Type,
	limit int32,
) ([]*relation.Relation, error) {
	rows, err := r.store.RelationListTypeByUser(ctx, db.RelationListTypeByUserParams{
		UserID:     db.ToUUID(userID.UUID()),
		TypeVal:    int16(relType),
		BatchLimit: limit,
	})
	if err != nil {
		return nil, r.store.Err(err)
	}

	relations := make([]*relation.Relation, 0, len(rows))
	for _, row := range rows {
		rel, err := relationFromListRow(userID, row)
		if err != nil {
			return nil, err
		}
		relations = append(relations, rel)
	}

	return relations, nil
}

// ListCachedTypeByUser fetches full relation models filtered by relation.Type for a user up to limit.
// Uses index-first caching: reads user:userID:relations hash (filtered by relType), backfills complete
// user relations on cache miss via singleflight, and fetches full entities using GetCachedBatch.
func (r *RelationRepository) ListCachedTypeByUser(
	ctx context.Context,
	userID fields.ID,
	relType relation.Type,
	limit int32,
) ([]*relation.Relation, error) {
	relMap, err := r.cache.GetUserRelations(ctx, userID, relType)

	if err != nil && !errors.Is(err, redis.ErrCacheMiss) {
		slog.WarnContext(ctx, "user relations hash cache read failed, falling back to database",
			"user_id", userID.String(),
			"error", err,
			"scope", redis.ScopeRelation,
		)
	}

	// Cache miss or read failure: Singleflight backfill ALL relations into Hash Index
	if err != nil || relMap == nil {
		sfKey := fmt.Sprintf("get_cached_user_relations:%s", userID.String())
		sfCtx := context.WithoutCancel(ctx)

		val, sfErr, _ := r.sf.Do(sfKey, func() (any, error) {
			// Fetch ALL relations (TypeUnknown / 0) to ensure complete hash backfill in Redis
			allRelations, repoErr := r.ListTypeByUser(sfCtx, userID, relation.TypeUnknown, MaxType)
			if repoErr != nil {
				return nil, repoErr
			}

			capLimit := len(allRelations)
			if capLimit > MaxType {
				capLimit = MaxType
			}

			fullMap := make(map[uuid.UUID]relation.Type, capLimit)
			for i, rel := range allRelations {
				if i >= MaxType {
					break
				}
				fullMap[rel.PeerID(userID).UUID()] = rel.Type()
			}

			if cacheErr := r.cache.SetUserRelations(sfCtx, userID, fullMap); cacheErr != nil {
				slog.WarnContext(sfCtx, "failed to backfill user relations cache hash",
					"user_id", userID.String(),
					"error", cacheErr,
					"scope", redis.ScopeRelation,
				)
			}

			return fullMap, nil
		})

		if sfErr != nil {
			return nil, sfErr
		}

		fullMap := val.(map[uuid.UUID]relation.Type)

		relMap = make(map[uuid.UUID]relation.Type, len(fullMap))
		for peerUUID, t := range fullMap {
			if relType == relation.TypeUnknown || t == relType {
				relMap[peerUUID] = t
			}
		}
	}

	if len(relMap) == 0 {
		return []*relation.Relation{}, nil
	}

	// 1. Collect ALL candidate peer IDs (do NOT truncate here)
	peerIDs := make([]fields.ID, 0, len(relMap))
	for peerUUID := range relMap {
		peerIDs = append(peerIDs, fields.ID(peerUUID))
	}

	// 2. Hydrate full relation entities from cache/DB batch
	relationBatchMap, err := r.GetCachedBatch(ctx, userID, peerIDs)
	if err != nil {
		return nil, err
	}

	relations := make([]*relation.Relation, 0, len(relationBatchMap))
	for _, rel := range relationBatchMap {
		relations = append(relations, rel)
	}

	// 3. Sort deterministically by CreatedAt DESC
	slices.SortFunc(relations, func(a, b *relation.Relation) int {
		if cmp := b.CreatedAt().Time().Compare(a.CreatedAt().Time()); cmp != 0 {
			return cmp
		}
		// Tie-breaker on Peer ID for absolute stability
		return strings.Compare(a.PeerID(userID).String(), b.PeerID(userID).String())
	})

	// 4. Truncate by limit AFTER sorting
	if limit > 0 && int32(len(relations)) > limit {
		relations = relations[:limit]
	}

	return relations, nil
}

func (r *RelationRepository) Save(ctx context.Context, rel *relation.Relation) (*relation.Relation, error) {
	row, err := r.store.RelationSave(ctx, db.RelationSaveParams{
		User1ID:   db.ToUUID(rel.User1ID().UUID()),
		User2ID:   db.ToUUID(rel.User2ID().UUID()),
		ActorID:   db.ToUUID(rel.ActorID().UUID()),
		ChannelID: db.ToUUIDPtr(rel.ChannelID().UUIDPtr()),
		Type:      rel.Type().Int16(),
		CreatedAt: db.ToTimestamptz(rel.CreatedAt().Time()),
		UpdatedAt: db.ToTimestamptz(rel.UpdatedAt().Time()),
	})
	if err != nil {
		return nil, r.store.Err(err)
	}

	return relationFromRow(row)
}

// -----------------------------------------------------------------------------
// Row Mappers
// -----------------------------------------------------------------------------

func relationFromRow(row db.Relation) (*relation.Relation, error) {
	u1UUID := db.FromUUID[uuid.UUID](row.User1ID)
	u2UUID := db.FromUUID[uuid.UUID](row.User2ID)

	mapErr := func(msg, key string, val any, err error) *errs.Error {
		return errs.Internal(msg).
			Wrap(err).
			Reason("CORRUPT_DATABASE_RECORD").
			Meta(key, fmt.Sprintf("%v", val)).
			Resource("Relation", fmt.Sprintf("%s:%s", u1UUID.String(), u2UUID.String()), "", "database row mapping")
	}

	user1ID, err := fields.ParseRequiredID("user1_id", u1UUID)
	if err != nil {
		return nil, mapErr("failed to parse user1_id from database", "user1_id", u1UUID.String(), err)
	}

	user2ID, err := fields.ParseRequiredID("user2_id", u2UUID)
	if err != nil {
		return nil, mapErr("failed to parse user2_id from database", "user2_id", u2UUID.String(), err)
	}

	actorUUID := db.FromUUID[uuid.UUID](row.ActorID)
	actorID, err := fields.ParseRequiredID("actor_id", actorUUID)
	if err != nil {
		return nil, mapErr("failed to parse actor_id from database", "actor_id", actorUUID.String(), err)
	}

	var channelID fields.ID
	if row.ChannelID.Valid {
		chUUID := db.FromUUID[uuid.UUID](row.ChannelID)
		channelID, err = fields.ParseRequiredID("channel_id", chUUID)
		if err != nil {
			return nil, mapErr("failed to parse channel_id from database", "channel_id", chUUID.String(), err)
		}
	}

	createdAt := fields.NewTimestampFromTime(db.FromTimestamptz(row.CreatedAt))
	updatedAt := fields.NewTimestampFromTime(db.FromTimestamptz(row.UpdatedAt))

	return relation.Reconstitute(
		user1ID,
		user2ID,
		actorID,
		channelID,
		relation.Type(row.Type),
		createdAt,
		updatedAt,
	), nil
}

func relationFromListRow(userID fields.ID, row db.RelationListTypeByUserRow) (*relation.Relation, error) {
	peerUUID := db.FromUUID[uuid.UUID](row.PeerID)

	mapErr := func(msg, key string, val any, err error) *errs.Error {
		return errs.Internal(msg).
			Wrap(err).
			Reason("CORRUPT_DATABASE_RECORD").
			Meta(key, fmt.Sprintf("%v", val)).
			Resource("Relation", fmt.Sprintf("%s:%s", userID.String(), peerUUID.String()), "", "database list row mapping")
	}

	peerID, err := fields.ParseRequiredID("peer_id", peerUUID)
	if err != nil {
		return nil, mapErr("failed to parse peer_id from database list row", "peer_id", peerUUID.String(), err)
	}

	user1ID, user2ID := relation.SortUserIDs(userID, peerID)

	actorUUID := db.FromUUID[uuid.UUID](row.ActorID)
	actorID, err := fields.ParseRequiredID("actor_id", actorUUID)
	if err != nil {
		return nil, mapErr("failed to parse actor_id from database list row", "actor_id", actorUUID.String(), err)
	}

	var channelID fields.ID
	if row.ChannelID.Valid {
		chUUID := db.FromUUID[uuid.UUID](row.ChannelID)
		channelID, err = fields.ParseRequiredID("channel_id", chUUID)
		if err != nil {
			return nil, mapErr("failed to parse channel_id from database list row", "channel_id", chUUID.String(), err)
		}
	}

	createdAt := fields.NewTimestampFromTime(db.FromTimestamptz(row.CreatedAt))
	updatedAt := fields.NewTimestampFromTime(db.FromTimestamptz(row.UpdatedAt))

	return relation.Reconstitute(
		user1ID,
		user2ID,
		actorID,
		channelID,
		relation.Type(row.Type),
		createdAt,
		updatedAt,
	), nil
}
