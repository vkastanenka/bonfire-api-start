package repository

import (
	"context"

	"bonfire-api/internal/db"
	"bonfire-api/internal/relationship"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type Relationship struct {
	store db.Querier
}

func NewRelationship(store db.Querier) *Relationship {
	return &Relationship{store: store}
}

// Delete removes a relationship aggregate given its participant IDs.
func (r *Relationship) Delete(ctx context.Context, user1ID, user2ID uuid.UUID) error {
	err := r.store.RelationshipDelete(ctx, db.RelationshipDeleteParams{
		User1ID: db.UUID(user1ID),
		User2ID: db.UUID(user2ID),
	})
	if err != nil {
		return db.NewError(err, db.EntityRelationship)
	}

	return nil
}

// DeleteVerified removes a relationship while verifying actor permissions (e.g., blocking safeguards).
func (r *Relationship) DeleteVerified(ctx context.Context, user1ID, user2ID, actorID uuid.UUID) error {
	err := r.store.RelationshipDeleteVerified(ctx, db.RelationshipDeleteVerifiedParams{
		User1ID: db.UUID(user1ID),
		User2ID: db.UUID(user2ID),
		ActorID: db.UUID(actorID),
	})
	if err != nil {
		return db.NewError(err, db.EntityRelationship)
	}

	return nil
}

// Get fetches a single relationship aggregate by participants.
func (r *Relationship) Get(ctx context.Context, user1ID, user2ID uuid.UUID) (*relationship.Relationship, error) {
	row, err := r.store.RelationshipGet(ctx, db.RelationshipGetParams{
		User1ID: db.UUID(user1ID),
		User2ID: db.UUID(user2ID),
	})
	if err != nil {
		return nil, db.NewError(err, db.EntityRelationship)
	}

	return relationshipFromRow(row)
}

// GetByChannelID fetches a single relationship aggregate by its direct message channel ID.
func (r *Relationship) GetByChannelID(ctx context.Context, channelID uuid.UUID) (*relationship.Relationship, error) {
	row, err := r.store.RelationshipGetByChannelID(ctx, db.UUID(channelID))
	if err != nil {
		return nil, db.NewError(err, db.EntityRelationship)
	}

	return relationshipFromGetByChannelIDRow(row)
}

// GetForUpdate fetches a single relationship aggregate by participants and locks the row FOR UPDATE.
func (r *Relationship) GetForUpdate(ctx context.Context, user1ID, user2ID uuid.UUID) (*relationship.Relationship, error) {
	row, err := r.store.RelationshipGetForUpdate(ctx, db.RelationshipGetForUpdateParams{
		User1ID: db.UUID(user1ID),
		User2ID: db.UUID(user2ID),
	})
	if err != nil {
		return nil, db.NewError(err, db.EntityRelationship)
	}

	return relationshipFromRow(row)
}

// GetPerspectiveByChannelID retrieves the UI projection for a user by DM channel ID.
func (r *Relationship) GetPerspective(ctx context.Context, user1ID, user2ID uuid.UUID) (*relationship.Perspective, error) {
	row, err := r.store.RelationshipPerspectiveGet(ctx, db.RelationshipPerspectiveGetParams{
		UserID: db.UUID(user1ID),
		PeerID: db.UUID(user2ID),
	})
	if err != nil {
		return nil, db.NewError(err, db.EntityRelationship)
	}

	return perspectiveFromRow(row)
}

// GetPerspectiveByChannelID retrieves the UI projection for a user by DM channel ID.
func (r *Relationship) GetPerspectiveByChannelID(ctx context.Context, userID, channelID uuid.UUID) (*relationship.Perspective, error) {
	row, err := r.store.RelationshipPerspectiveGetByChannelID(ctx, db.RelationshipPerspectiveGetByChannelIDParams{
		UserID:    db.UUID(userID),
		ChannelID: db.UUID(channelID),
	})
	if err != nil {
		return nil, db.NewError(err, db.EntityRelationship)
	}

	return perspectiveFromRow(row)
}

// HasBlockBetweenUserAndPeers checks if a block relationship exists between a user and any of the provided peer IDs.
func (r *Relationship) HasBlockBetweenUserAndPeers(ctx context.Context, userID uuid.UUID, peerIDs []uuid.UUID) (bool, error) {
	if len(peerIDs) == 0 {
		return false, nil
	}

	pgPeerIDs := make([]pgtype.UUID, len(peerIDs))
	for i, id := range peerIDs {
		pgPeerIDs[i] = db.UUID(id)
	}

	blocked, err := r.store.RelationshipHasBlockBetweenUserAndPeers(ctx, db.RelationshipHasBlockBetweenUserAndPeersParams{
		UserID:  db.UUID(userID),
		PeerIds: pgPeerIDs,
	})
	if err != nil {
		return false, db.NewError(err, db.EntityRelationship)
	}

	return blocked, nil
}

// // ListPerspectives retrieves all relationship projections for a user, optionally filtered by relationship type.
// func (r *Relationship) ListPerspectives(ctx context.Context, userID uuid.UUID, filterVariant *relationship.Variant) ([]relationship.Perspective, error) {
// 	rows, err := r.store.RelationshipPerspectivesList(ctx, db.RelationshipPerspectivesListParams{
// 		UserID:        db.UUID(userID),
// 		FilterVariant: db.Int2Ptr(filterVariant),
// 	})
// 	if err != nil {
// 		return nil, db.NewError(err, db.EntityRelationship)
// 	}

// 	perspectives := make([]relationship.Perspective, len(rows))
// 	for i, row := range rows {
// 		p, err := perspectiveFromRow(row)
// 		if err != nil {
// 			return nil, err
// 		}
// 		perspectives[i] = *p
// 	}

// 	return perspectives, nil
// }

// Upsert creates or updates a relationship aggregate state.
func (r *Relationship) Upsert(ctx context.Context, rel *relationship.Relationship) (*relationship.Relationship, error) {
	var channelID *uuid.UUID
	if rel.ChannelID() != nil {
		id := rel.ChannelID().UUID()
		channelID = &id
	}

	row, err := r.store.RelationshipUpsert(ctx, db.RelationshipUpsertParams{
		User1ID:   db.UUID(rel.User1ID().UUID()),
		User2ID:   db.UUID(rel.User2ID().UUID()),
		ActorID:   db.UUID(rel.ActorID().UUID()),
		ChannelID: db.UUIDPtr(channelID),
		CreatedAt: db.Timestamptz(rel.CreatedAt()),
		UpdatedAt: db.Timestamptz(rel.UpdatedAt()),
		Variant:   int16(rel.Variant()),
	})
	if err != nil {
		return nil, db.NewError(err, db.EntityRelationship)
	}

	reconstituted, err := relationshipFromRow(row)
	if err != nil {
		return nil, db.NewError(err, db.EntityRelationship)
	}

	return reconstituted, nil
}

// ============================================================================
// Internal Domain Reconstitution Helpers
// ============================================================================

func relationshipFromRow(row db.Relationship) (*relationship.Relationship, error) {
	return relationship.Reconstitute(
		uuid.UUID(row.User1ID.Bytes),
		uuid.UUID(row.User2ID.Bytes),
		uuid.UUID(row.ActorID.Bytes),
		db.UUIDPtrFromDB(row.ChannelID),
		uint8(row.Variant),
		row.CreatedAt.Time.UTC(),
		row.UpdatedAt.Time.UTC(),
	)
}

func relationshipFromGetByChannelIDRow(row db.Relationship) (*relationship.Relationship, error) {
	return relationship.Reconstitute(
		uuid.UUID(row.User1ID.Bytes),
		uuid.UUID(row.User2ID.Bytes),
		uuid.UUID(row.ActorID.Bytes),
		db.UUIDPtrFromDB(row.ChannelID),
		uint8(row.Variant),
		row.CreatedAt.Time.UTC(),
		row.UpdatedAt.Time.UTC(),
	)
}

func perspectiveFromRow(row db.RelationshipPerspective) (*relationship.Perspective, error) {
	var avatarURL *string
	if row.AvatarUrl.Valid && row.AvatarUrl.String != "" {
		avatarURL = &row.AvatarUrl.String
	}

	return relationship.ReconstitutePerspective(
		uuid.UUID(row.UserID.Bytes),
		uuid.UUID(row.PeerID.Bytes),
		uuid.UUID(row.ActorID.Bytes),
		db.UUIDPtrFromDB(row.ChannelID),
		uint8(row.Variant),
		row.IsInitiator,
		row.CreatedAt.Time.UTC(),
		row.UpdatedAt.Time.UTC(),
		row.Username,
		row.DisplayName.String,
		avatarURL,
	)
}
