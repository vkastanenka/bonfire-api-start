package repository

import (
	"context"
	"fmt"

	"bonfire-api/internal/db"
	"bonfire-api/internal/errs"
	"bonfire-api/internal/fields"
	"bonfire-api/internal/relation"

	"github.com/google/uuid"
)

type RelationRepository struct {
	store *db.Store
}

func NewRelationRepository(store *db.Store) *RelationRepository {
	return &RelationRepository{
		store: store.WithEntity(db.EntityRelation),
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
