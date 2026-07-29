package repository

import (
	"context"

	"bonfire-api/internal/db"
	"bonfire-api/internal/errs"
	"bonfire-api/internal/presence"
	"bonfire-api/internal/relationship"
	"bonfire-api/internal/user"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type RelationshipStore interface {
	RelationshipGet(ctx context.Context, arg db.RelationshipGetParams) (db.RelationshipGetRow, error)
	RelationshipGetForUpdate(ctx context.Context, arg db.RelationshipGetForUpdateParams) (db.RelationshipGetForUpdateRow, error)
	RelationshipHasBlockBetweenUserAndPeers(ctx context.Context, arg db.RelationshipHasBlockBetweenUserAndPeersParams) (bool, error)
	RelationshipUpsert(ctx context.Context, arg db.RelationshipUpsertParams) (db.RelationshipUpsertRow, error)
	RelationshipDelete(ctx context.Context, arg db.RelationshipDeleteParams) error
	RelationshipDeleteVerified(ctx context.Context, arg db.RelationshipDeleteVerifiedParams) error
	RelationshipPerspectiveGet(ctx context.Context, arg db.RelationshipPerspectiveGetParams) (db.RelationshipPerspective, error)
	RelationshipPerspectivesList(ctx context.Context, arg db.RelationshipPerspectivesListParams) ([]db.RelationshipPerspective, error)
}

type Relationship struct {
	store RelationshipStore
}

func NewRelationship(store RelationshipStore) *Relationship {
	return &Relationship{store: store}
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

	return relationshipFromRow(row), nil
}

// GetForUpdate fetches a single relationship aggregate with row-level locking.
func (r *Relationship) GetForUpdate(ctx context.Context, user1ID, user2ID uuid.UUID) (*relationship.Relationship, error) {
	row, err := r.store.RelationshipGetForUpdate(ctx, db.RelationshipGetForUpdateParams{
		User1ID: db.UUID(user1ID),
		User2ID: db.UUID(user2ID),
	})
	if err != nil {
		return nil, db.NewError(err, db.EntityRelationship)
	}

	return relationshipFromGetForUpdateRow(row), nil
}

// Upsert creates or updates a relationship aggregate state.
func (r *Relationship) Upsert(ctx context.Context, rel *relationship.Relationship) error {
	row, err := r.store.RelationshipUpsert(ctx, db.RelationshipUpsertParams{
		User1ID: db.UUID(rel.User1ID()),
		User2ID: db.UUID(rel.User2ID()),
		ActorID: db.UUID(rel.ActorID()),
		Variant: int16(rel.Variant()),
	})
	if err != nil {
		return db.NewError(err, db.EntityRelationship)
	}

	*rel = *relationshipFromUpsertRow(row)
	return nil
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

// GetPerspective retrieves the UI projection for a specific user and peer.
func (r *Relationship) GetPerspective(ctx context.Context, userID, peerID uuid.UUID) (*relationship.Perspective, error) {
	row, err := r.store.RelationshipPerspectiveGet(ctx, db.RelationshipPerspectiveGetParams{
		UserID: db.UUID(userID),
		PeerID: db.UUID(peerID),
	})
	if err != nil {
		return nil, db.NewError(err, db.EntityRelationship)
	}

	return perspectiveFromRow(row)
}

// ListPerspectives retrieves all relationship projections for a user, optionally filtered by relationship type.
func (r *Relationship) ListPerspectives(ctx context.Context, userID uuid.UUID, filterVariant *relationship.Variant) ([]relationship.Perspective, error) {
	rows, err := r.store.RelationshipPerspectivesList(ctx, db.RelationshipPerspectivesListParams{
		UserID:        db.UUID(userID),
		FilterVariant: db.Int2Ptr(filterVariant),
	})
	if err != nil {
		return nil, db.NewError(err, db.EntityRelationship)
	}

	perspectives := make([]relationship.Perspective, len(rows))
	for i, row := range rows {
		p, err := perspectiveFromRow(row)
		if err != nil {
			return nil, err
		}
		perspectives[i] = *p
	}

	return perspectives, nil
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

// ============================================================================
// Internal Domain Reconstitution Helpers
// ============================================================================

func relationshipFromRow(row db.RelationshipGetRow) *relationship.Relationship {
	return relationship.Reconstitute(
		uuid.UUID(row.User1ID.Bytes),
		uuid.UUID(row.User2ID.Bytes),
		uuid.UUID(row.ActorID.Bytes),
		relationship.Variant(row.Variant),
		row.CreatedAt.Time.UTC(),
		row.UpdatedAt.Time.UTC(),
	)
}

func relationshipFromGetForUpdateRow(row db.RelationshipGetForUpdateRow) *relationship.Relationship {
	return relationship.Reconstitute(
		uuid.UUID(row.User1ID.Bytes),
		uuid.UUID(row.User2ID.Bytes),
		uuid.UUID(row.ActorID.Bytes),
		relationship.Variant(row.Variant),
		row.CreatedAt.Time.UTC(),
		row.UpdatedAt.Time.UTC(),
	)
}

func relationshipFromUpsertRow(row db.RelationshipUpsertRow) *relationship.Relationship {
	return relationship.Reconstitute(
		uuid.UUID(row.User1ID.Bytes),
		uuid.UUID(row.User2ID.Bytes),
		uuid.UUID(row.ActorID.Bytes),
		relationship.Variant(row.Variant),
		row.CreatedAt.Time.UTC(),
		row.UpdatedAt.Time.UTC(),
	)
}

func perspectiveFromRow(row db.RelationshipPerspective) (*relationship.Perspective, error) {
	userID := uuid.UUID(row.UserID.Bytes).String()
	peerID := uuid.UUID(row.PeerID.Bytes).String()

	username, err := user.NewUsername(row.Username)
	if err != nil {
		return nil, errs.Internal("failed to parse peer username from database perspective").
			Wrap(err).
			Reason("CORRUPT_DATABASE_RECORD").
			Meta("username", row.Username).
			Resource("RelationshipPerspective", userID, peerID, "database perspective mapping")
	}

	var displayName *user.ProfileDisplayName
	if row.DisplayName.Valid && row.DisplayName.String != "" {
		dn, err := user.NewProfileDisplayName(row.DisplayName.String)
		if err != nil {
			return nil, errs.Internal("failed to parse peer display name from database perspective").
				Wrap(err).
				Reason("CORRUPT_DATABASE_RECORD").
				Meta("display_name", row.DisplayName.String).
				Resource("RelationshipPerspective", userID, peerID, "database perspective mapping")
		}
		displayName = &dn
	}

	var avatarURL *string
	if row.AvatarUrl.Valid && row.AvatarUrl.String != "" {
		avatarURL = &row.AvatarUrl.String
	}

	prefPresence := presence.Presence(row.UserPreferredPresence.Int16)

	return relationship.ReconstitutePerspective(
		uuid.UUID(row.UserID.Bytes),
		uuid.UUID(row.PeerID.Bytes),
		relationship.Variant(row.Variant),
		uuid.UUID(row.ActorID.Bytes),
		row.IsInitiator,
		row.CreatedAt.Time.UTC(),
		row.UpdatedAt.Time.UTC(),
		username,
		displayName,
		avatarURL,
		prefPresence,
		db.UUIDPtrFromDB(row.ChannelID),
	), nil
}
