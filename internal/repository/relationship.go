package repository

import (
	"context"

	"bonfire-api/internal/db"
	"bonfire-api/internal/relationship"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type Relationship struct {
	q db.Store
}

var _ relationship.Repository = (*Relationship)(nil)

func NewRelationship(q db.Store) *Relationship {
	return &Relationship{q: q}
}

// Get fetches a single relationship aggregate by participants.
func (r *Relationship) Get(ctx context.Context, user1ID, user2ID uuid.UUID) (*relationship.Relationship, error) {
	row, err := r.q.RelationshipGet(ctx, db.RelationshipGetParams{
		User1ID: pgtype.UUID{Bytes: user1ID, Valid: true},
		User2ID: pgtype.UUID{Bytes: user2ID, Valid: true},
	})
	if err != nil {
		return nil, db.NewError(err, db.EntityRelationship)
	}

	return relationshipFromRow(row), nil
}

// GetForUpdate fetches a single relationship aggregate with row-level locking.
func (r *Relationship) GetForUpdate(ctx context.Context, user1ID, user2ID uuid.UUID) (*relationship.Relationship, error) {
	row, err := r.q.RelationshipGetForUpdate(ctx, db.RelationshipGetForUpdateParams{
		User1ID: pgtype.UUID{Bytes: user1ID, Valid: true},
		User2ID: pgtype.UUID{Bytes: user2ID, Valid: true},
	})
	if err != nil {
		return nil, db.NewError(err, db.EntityRelationship)
	}

	return relationshipFromGetForUpdateRow(row), nil
}

// Upsert creates or updates a relationship aggregate state.
func (r *Relationship) Upsert(ctx context.Context, rel *relationship.Relationship) error {
	row, err := r.q.RelationshipUpsert(ctx, db.RelationshipUpsertParams{
		User1ID: pgtype.UUID{Bytes: rel.User1ID(), Valid: true},
		User2ID: pgtype.UUID{Bytes: rel.User2ID(), Valid: true},
		ActorID: pgtype.UUID{Bytes: rel.ActorID(), Valid: true},
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
	err := r.q.RelationshipDelete(ctx, db.RelationshipDeleteParams{
		User1ID: pgtype.UUID{Bytes: user1ID, Valid: true},
		User2ID: pgtype.UUID{Bytes: user2ID, Valid: true},
	})
	if err != nil {
		return db.NewError(err, db.EntityRelationship)
	}

	return nil
}

// DeleteVerified removes a relationship while verifying actor permissions (e.g., blocking safeguards).
func (r *Relationship) DeleteVerified(ctx context.Context, user1ID, user2ID, actorID uuid.UUID) error {
	err := r.q.RelationshipDeleteVerified(ctx, db.RelationshipDeleteVerifiedParams{
		User1ID: pgtype.UUID{Bytes: user1ID, Valid: true},
		User2ID: pgtype.UUID{Bytes: user2ID, Valid: true},
		ActorID: pgtype.UUID{Bytes: actorID, Valid: true},
	})
	if err != nil {
		return db.NewError(err, db.EntityRelationship)
	}

	return nil
}

// GetPerspective retrieves the UI projection for a specific user and peer.
func (r *Relationship) GetPerspective(ctx context.Context, userID, peerID uuid.UUID) (*relationship.Perspective, error) {
	row, err := r.q.RelationshipPerspectiveGet(ctx, db.RelationshipPerspectiveGetParams{
		UserID: pgtype.UUID{Bytes: userID, Valid: true},
		PeerID: pgtype.UUID{Bytes: peerID, Valid: true},
	})
	if err != nil {
		return nil, db.NewError(err, db.EntityRelationship)
	}

	return perspectiveFromRow(row), nil
}

// ListPerspectives retrieves all relationship projections for a user, optionally filtered by relationship type.
func (r *Relationship) ListPerspectives(ctx context.Context, userID uuid.UUID, filterVariant *relationship.Variant) ([]relationship.Perspective, error) {
	var variantParam pgtype.Int2
	if filterVariant != nil {
		variantParam = pgtype.Int2{Int16: int16(*filterVariant), Valid: true}
	}

	rows, err := r.q.RelationshipPerspectivesList(ctx, db.RelationshipPerspectivesListParams{
		UserID:        pgtype.UUID{Bytes: userID, Valid: true},
		FilterVariant: variantParam,
	})
	if err != nil {
		return nil, db.NewError(err, db.EntityRelationship)
	}

	perspectives := make([]relationship.Perspective, len(rows))
	for i, row := range rows {
		perspectives[i] = *perspectiveFromRow(row) // Reused helper here
	}

	return perspectives, nil
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
		row.CreatedAt.Time,
		row.UpdatedAt.Time,
	)
}

func relationshipFromGetForUpdateRow(row db.RelationshipGetForUpdateRow) *relationship.Relationship {
	return relationship.Reconstitute(
		uuid.UUID(row.User1ID.Bytes),
		uuid.UUID(row.User2ID.Bytes),
		uuid.UUID(row.ActorID.Bytes),
		relationship.Variant(row.Variant),
		row.CreatedAt.Time,
		row.UpdatedAt.Time,
	)
}

func relationshipFromUpsertRow(row db.RelationshipUpsertRow) *relationship.Relationship {
	return relationship.Reconstitute(
		uuid.UUID(row.User1ID.Bytes),
		uuid.UUID(row.User2ID.Bytes),
		uuid.UUID(row.ActorID.Bytes),
		relationship.Variant(row.Variant),
		row.CreatedAt.Time,
		row.UpdatedAt.Time,
	)
}

func perspectiveFromRow(row db.RelationshipPerspective) *relationship.Perspective {
	var channelID *uuid.UUID
	if row.ChannelID.Valid {
		id := uuid.UUID(row.ChannelID.Bytes)
		channelID = &id
	}

	var displayName *string
	if row.DisplayName.Valid {
		displayName = &row.DisplayName.String
	}

	var avatarURL *string
	if row.AvatarUrl.Valid {
		avatarURL = &row.AvatarUrl.String
	}

	return relationship.ReconstitutePerspective(
		uuid.UUID(row.UserID.Bytes),
		uuid.UUID(row.PeerID.Bytes),
		relationship.Variant(row.Variant),
		uuid.UUID(row.ActorID.Bytes),
		row.IsInitiator,
		row.CreatedAt.Time,
		row.UpdatedAt.Time,
		row.Username,
		displayName,
		avatarURL,
		row.UserPreferredPresence.Int16,
		channelID,
	)
}
