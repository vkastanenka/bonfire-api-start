package relationship

import (
	"context"

	"bonfire-api/internal/apperr"
	"bonfire-api/internal/repository"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// --- relationship service ---

type Service struct {
	store repository.Store
}

func NewService(
	store repository.Store,
) *Service {
	return &Service{
		store: store,
	}
}

// ==========================================
// META
// ==========================================

// --- relationship service Count ---

func (s *Service) Count(ctx context.Context) (int64, error) {
	count, err := s.store.RelationshipsCount(ctx)
	if err != nil {
		return 0, apperr.NewDBError(err)
	}
	return count, nil
}

// ==========================================
// LIST
// ==========================================

// --- relationship service List ---

type ListParams struct {
	UserID uuid.UUID
	Status Status
}

func (s *Service) List(ctx context.Context, userID uuid.UUID, status repository.RelationshipStatus) ([]repository.RelationshipsListByUserRow, error) {
	rows, err := s.store.RelationshipsListByUser(ctx, repository.RelationshipsListByUserParams{
		UserID: pgtype.UUID{Bytes: userID, Valid: true},
		Status: status,
	})
	if err != nil {
		return nil, apperr.NewDBError(err)
	}
	return rows, nil
}

// ==========================================
// UPSERT / UPDATE
// ==========================================

// --- relationship service SendFriendRequest ---

type SendFriendRequestParams struct {
	ActorID  uuid.UUID
	TargetID uuid.UUID
}

func (s *Service) SendFriendRequest(ctx context.Context, p SendFriendRequestParams) error {
	if p.ActorID == p.TargetID {
		return apperr.New(apperr.CodeBadRequest, "cannot add yourself as a friend")
	}

	u1, u2 := orderUUIDs(p.ActorID, p.TargetID)

	relRow, err := s.store.RelationshipGet(ctx, repository.RelationshipGetParams{
		User1ID: pgtype.UUID{Bytes: u1, Valid: true},
		User2ID: pgtype.UUID{Bytes: u2, Valid: true},
	})
	if err != nil {
		return apperr.NewDBError(err)
	}

	switch relRow.Status {
	case repository.RelationshipStatusFriends:
		return apperr.New(apperr.CodeBadRequest, "already friends")
	case repository.RelationshipStatusBlocked:
		return apperr.New(apperr.CodeForbidden, "cannot interact with this user")
	case repository.RelationshipStatusPending:
		if relRow.ActionUserID.Bytes != p.ActorID {
			_, err = s.store.RelationshipUpsert(ctx, repository.RelationshipUpsertParams{
				User1ID:      pgtype.UUID{Bytes: u1, Valid: true},
				User2ID:      pgtype.UUID{Bytes: u2, Valid: true},
				ActionUserID: pgtype.UUID{Bytes: p.ActorID, Valid: true},
				Status:       repository.RelationshipStatusFriends,
			})
			if err != nil {
				return apperr.NewDBError(err)
			}
			return nil
		}
		return apperr.New(apperr.CodeBadRequest, "request already pending")
	}

	_, err = s.store.RelationshipUpsert(ctx, repository.RelationshipUpsertParams{
		User1ID:      pgtype.UUID{Bytes: u1, Valid: true},
		User2ID:      pgtype.UUID{Bytes: u2, Valid: true},
		ActionUserID: pgtype.UUID{Bytes: p.ActorID, Valid: true},
		Status:       repository.RelationshipStatusPending,
	})
	if err != nil {
		return apperr.NewDBError(err)
	}

	return nil
}

// --- relationship service AcceptFriendRequest ---

type AcceptFriendRequestParams struct {
	ActorID  uuid.UUID
	TargetID uuid.UUID
}

func (s *Service) AcceptFriendRequest(ctx context.Context, p AcceptFriendRequestParams) error {
	u1, u2 := orderUUIDs(p.ActorID, p.TargetID)

	rel, err := s.store.RelationshipGet(ctx, repository.RelationshipGetParams{
		User1ID: pgtype.UUID{Bytes: u1, Valid: true},
		User2ID: pgtype.UUID{Bytes: u2, Valid: true},
	})
	if err != nil {
		return apperr.NewDBError(err)
	}

	if rel.Status != repository.RelationshipStatusPending {
		return apperr.New(apperr.CodeBadRequest, "no pending request to accept")
	}

	if rel.ActionUserID.Bytes == p.ActorID {
		return apperr.New(apperr.CodeForbidden, "cannot accept your own request")
	}

	_, err = s.store.RelationshipUpsert(ctx, repository.RelationshipUpsertParams{
		User1ID:      pgtype.UUID{Bytes: u1, Valid: true},
		User2ID:      pgtype.UUID{Bytes: u2, Valid: true},
		ActionUserID: pgtype.UUID{Bytes: p.ActorID, Valid: true},
		Status:       repository.RelationshipStatusFriends,
	})
	if err != nil {
		return apperr.NewDBError(err)
	}

	return nil
}

// --- relationship service Block ---

type BlockParams struct {
	ActorID  uuid.UUID
	TargetID uuid.UUID
}

func (s *Service) Block(ctx context.Context, p BlockParams) error {
	u1, u2 := orderUUIDs(p.ActorID, p.TargetID)

	_, err := s.store.RelationshipUpsert(ctx, repository.RelationshipUpsertParams{
		User1ID:      pgtype.UUID{Bytes: u1, Valid: true},
		User2ID:      pgtype.UUID{Bytes: u2, Valid: true},
		ActionUserID: pgtype.UUID{Bytes: p.ActorID, Valid: true},
		Status:       repository.RelationshipStatusBlocked,
	})
	if err != nil {
		return apperr.NewDBError(err)
	}

	return nil
}

// ==========================================
// DELETE
// ==========================================

// --- relationship service Delete ---

type DeleteParams struct {
	ActorID  uuid.UUID
	TargetID uuid.UUID
}

func (s *Service) Delete(ctx context.Context, p DeleteParams) error {
	u1, u2 := orderUUIDs(p.ActorID, p.TargetID)

	err := s.store.RelationshipDelete(ctx, repository.RelationshipDeleteParams{
		User1ID: pgtype.UUID{Bytes: u1, Valid: true},
		User2ID: pgtype.UUID{Bytes: u2, Valid: true},
	})
	if err != nil {
		return apperr.NewDBError(err)
	}

	return nil
}
