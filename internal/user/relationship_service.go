package user

import (
	"context"

	"bonfire-api/internal/apperr"
	"bonfire-api/internal/repository"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// orderUUIDs ensures we always query/insert with the lesser UUID first.
func orderUUIDs(id1, id2 uuid.UUID) (uuid.UUID, uuid.UUID) {
	if id1.String() < id2.String() {
		return id1, id2
	}
	return id2, id1
}

// ==========================================
// RELATIONSHIPS
// ==========================================

func (s *Service) SendFriendRequest(ctx context.Context, actorID, targetID uuid.UUID) error {
	if actorID == targetID {
		return apperr.New(apperr.CodeBadRequest, "cannot add yourself as a friend")
	}

	u1, u2 := orderUUIDs(actorID, targetID)

	// Check existing relationship
	rel, err := s.store.RelationshipGet(ctx, repository.RelationshipGetParams{
		User1ID: pgtype.UUID{Bytes: u1, Valid: true},
		User2ID: pgtype.UUID{Bytes: u2, Valid: true},
	})

	if err == nil { // Relationship exists
		if rel.Status == repository.RelationshipStatusBlocked {
			return apperr.New(apperr.CodeForbidden, "cannot interact with this user")
		}
		if rel.Status == repository.RelationshipStatusFriends {
			return apperr.New(apperr.CodeBadRequest, "already friends")
		}
		if rel.Status == repository.RelationshipStatusPending {
			return apperr.New(apperr.CodeBadRequest, "request already pending")
		}
	}

	_, err = s.store.RelationshipUpsert(ctx, repository.RelationshipUpsertParams{
		User1ID:      pgtype.UUID{Bytes: u1, Valid: true},
		User2ID:      pgtype.UUID{Bytes: u2, Valid: true},
		Status:       repository.RelationshipStatusPending,
		ActionUserID: pgtype.UUID{Bytes: actorID, Valid: true},
	})

	if err != nil {
		return apperr.NewDBError(err, Domain)
	}
	return nil
}

func (s *Service) AcceptFriendRequest(ctx context.Context, actorID, targetID uuid.UUID) error {
	u1, u2 := orderUUIDs(actorID, targetID)

	rel, err := s.store.RelationshipGet(ctx, repository.RelationshipGetParams{
		User1ID: pgtype.UUID{Bytes: u1, Valid: true},
		User2ID: pgtype.UUID{Bytes: u2, Valid: true},
	})
	if err != nil {
		return apperr.New(apperr.CodeNotFound, "friend request not found")
	}

	if rel.Status != repository.RelationshipStatusPending {
		return apperr.New(apperr.CodeBadRequest, "no pending request to accept")
	}

	// The person who sent the request cannot accept it.
	if rel.ActionUserID.Bytes == actorID {
		return apperr.New(apperr.CodeForbidden, "cannot accept your own request")
	}

	_, err = s.store.RelationshipUpsert(ctx, repository.RelationshipUpsertParams{
		User1ID:      pgtype.UUID{Bytes: u1, Valid: true},
		User2ID:      pgtype.UUID{Bytes: u2, Valid: true},
		Status:       repository.RelationshipStatusFriends,
		ActionUserID: pgtype.UUID{Bytes: actorID, Valid: true}, // Log who accepted it
	})

	return err
}

func (s *Service) BlockUser(ctx context.Context, actorID, targetID uuid.UUID) error {
	u1, u2 := orderUUIDs(actorID, targetID)

	_, err := s.store.RelationshipUpsert(ctx, repository.RelationshipUpsertParams{
		User1ID:      pgtype.UUID{Bytes: u1, Valid: true},
		User2ID:      pgtype.UUID{Bytes: u2, Valid: true},
		Status:       repository.RelationshipStatusBlocked,
		ActionUserID: pgtype.UUID{Bytes: actorID, Valid: true},
	})

	if err != nil {
		return apperr.NewDBError(err, Domain)
	}
	return nil
}

func (s *Service) RemoveRelationship(ctx context.Context, actorID, targetID uuid.UUID) error {
	u1, u2 := orderUUIDs(actorID, targetID)
	// Deletes friends, withdraws pending requests, or unblocks a user.
	err := s.store.RelationshipDelete(ctx, repository.RelationshipDeleteParams{
		User1ID: pgtype.UUID{Bytes: u1, Valid: true},
		User2ID: pgtype.UUID{Bytes: u2, Valid: true},
	})

	if err != nil {
		return apperr.NewDBError(err, Domain)
	}
	return nil
}

// ListRelationships can be used to list 'friends', 'pending', or 'blocked'
func (s *Service) ListRelationships(ctx context.Context, userID uuid.UUID, status repository.RelationshipStatus) ([]repository.RelationshipsListByUserRow, error) {
	rows, err := s.store.RelationshipsListByUser(ctx, repository.RelationshipsListByUserParams{
		ID: pgtype.UUID{Bytes: userID, Valid: true},
		Status:  status,
	})
	if err != nil {
		return nil, apperr.NewDBError(err, Domain)
	}
	return rows, nil
}
