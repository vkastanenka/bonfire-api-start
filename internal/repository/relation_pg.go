package repository

import (
	"context"

	"bonfire-api/internal/db"
	"bonfire-api/internal/errs"
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

func (r *RelationRepository) Save(ctx context.Context, rel *relation.Relation) (*relation.Relation, error) {
	row, err := r.store.RelationSave(ctx, db.RelationSaveParams{
		User1ID:   db.ToUUID(rel.User1ID),
		User2ID:   db.ToUUID(rel.User2ID),
		ActorID:   db.ToUUID(rel.ActorID),
		ChannelID: db.ToUUIDPtr(rel.ChannelID),
		Type:      int16(rel.Type),
		CreatedAt: db.ToTimestamptz(rel.CreatedAt),
		UpdatedAt: db.ToTimestamptz(rel.UpdatedAt),
	})
	if err != nil {
		return nil, r.store.Err(err)
	}

	return relationFromRow(row), nil
}

func (r *RelationRepository) Get(ctx context.Context, user1ID, user2ID uuid.UUID) (*relation.Relation, error) {
	row, err := r.store.RelationGet(ctx, db.RelationGetParams{
		User1ID: db.ToUUID(user1ID),
		User2ID: db.ToUUID(user2ID),
	})
	if err != nil {
		return nil, r.store.Err(err)
	}

	return relationFromRow(row), nil
}

func (r *RelationRepository) GetForUpdate(ctx context.Context, user1ID, user2ID uuid.UUID) (*relation.Relation, error) {
	row, err := r.store.RelationGetForUpdate(ctx, db.RelationGetForUpdateParams{
		User1ID: db.ToUUID(user1ID),
		User2ID: db.ToUUID(user2ID),
	})
	if err != nil {
		return nil, r.store.Err(err)
	}

	return relationFromRow(row), nil
}

func (r *RelationRepository) ListTypeByUserID(
	ctx context.Context,
	userID uuid.UUID,
	relType relation.Type,
	limit int,
) ([]*relation.Relation, error) {
	rows, err := r.store.RelationListTypeByUserID(ctx, db.RelationListTypeByUserIDParams{
		UserID:   db.ToUUID(userID),
		Type:     int16(relType),
		LimitVal: int32(limit),
	})
	if err != nil {
		return nil, r.store.Err(err)
	}

	return relationsFromRows(rows), nil
}

func (r *RelationRepository) ListFriendsByUserID(
	ctx context.Context,
	userID uuid.UUID,
	limit int,
) ([]*relation.Relation, error) {
	rows, err := r.store.RelationListFriendsByUserID(ctx, db.RelationListFriendsByUserIDParams{
		UserID:   db.ToUUID(userID),
		LimitVal: int32(limit),
	})
	if err != nil {
		return nil, r.store.Err(err)
	}

	return relationsFromRows(rows), nil
}

func (r *RelationRepository) ListIncomingPendingByUserID(
	ctx context.Context,
	userID uuid.UUID,
	limit int,
) ([]*relation.Relation, error) {
	rows, err := r.store.RelationListIncomingPendingByUserID(ctx, db.RelationListIncomingPendingByUserIDParams{
		UserID:   db.ToUUID(userID),
		LimitVal: int32(limit),
	})
	if err != nil {
		return nil, r.store.Err(err)
	}

	return relationsFromRows(rows), nil
}

func (r *RelationRepository) ListOutgoingBlocksByUserID(
	ctx context.Context,
	userID uuid.UUID,
	limit int,
) ([]*relation.Relation, error) {
	rows, err := r.store.RelationListOutgoingBlocksByUserID(ctx, db.RelationListOutgoingBlocksByUserIDParams{
		UserID:   db.ToUUID(userID),
		LimitVal: int32(limit),
	})
	if err != nil {
		return nil, r.store.Err(err)
	}

	return relationsFromRows(rows), nil
}

func (r *RelationRepository) ListIncomingBlocksByUserID(
	ctx context.Context,
	userID uuid.UUID,
	limit int,
) ([]*relation.Relation, error) {
	rows, err := r.store.RelationListIncomingBlocksByUserID(ctx, db.RelationListIncomingBlocksByUserIDParams{
		UserID:   db.ToUUID(userID),
		LimitVal: int32(limit),
	})
	if err != nil {
		return nil, r.store.Err(err)
	}

	return relationsFromRows(rows), nil
}

func (r *RelationRepository) HasIncomingBlock(ctx context.Context, actorID uuid.UUID, peerIDs []uuid.UUID) error {
	if len(peerIDs) == 0 {
		return nil
	}

	uuids := make([]uuid.UUID, len(peerIDs))
	for i, id := range peerIDs {
		uuids[i] = id
	}

	hasBlock, err := r.store.RelationHasIncomingBlock(ctx, db.RelationHasIncomingBlockParams{
		ActorID: db.ToUUID(actorID),
		PeerIds: db.ToUUIDs(uuids),
	})
	if err != nil {
		return r.store.Err(err)
	}

	if hasBlock {
		return errs.InvalidArgument("Cannot interact with users who have blocked you.").
			Reason("INCOMING_BLOCK_DETECTED")
	}

	return nil
}

func (r *RelationRepository) DeleteByUserID(ctx context.Context, user1ID, user2ID, actorID uuid.UUID) error {
	err := r.store.RelationDeleteByUserID(ctx, db.RelationDeleteByUserIDParams{
		User1ID: db.ToUUID(user1ID),
		User2ID: db.ToUUID(user2ID),
		ActorID: db.ToUUID(actorID),
	})
	if err != nil {
		return r.store.Err(err)
	}

	return nil
}

func relationFromRow(row db.Relation) *relation.Relation {
	return relation.Reconstitute(
		db.FromUUID[uuid.UUID](row.User1ID),
		db.FromUUID[uuid.UUID](row.User2ID),
		db.FromUUID[uuid.UUID](row.ActorID),
		db.FromUUIDPtr[uuid.UUID](row.ChannelID),
		relation.Type(int(row.Type)),
		db.FromTimestamptz(row.CreatedAt),
		db.FromTimestamptz(row.UpdatedAt),
	)
}

func relationsFromRows(rows []db.Relation) []*relation.Relation {
	relations := make([]*relation.Relation, 0, len(rows))
	for _, row := range rows {
		relations = append(relations, relationFromRow(row))
	}
	return relations
}
