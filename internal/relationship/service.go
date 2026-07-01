package relationship

import (
	"context"

	"bonfire-api/internal/apperr"
	"bonfire-api/internal/presence"
	"bonfire-api/internal/repository"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// --- relationship service ---

type Service struct {
	store    repository.Store
	presence *presence.Service
}

func NewService(
	store repository.Store,
	presence *presence.Service,
) *Service {
	return &Service{
		store:    store,
		presence: presence,
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

func (s *Service) List(ctx context.Context, p ListParams) ([]UserView, error) {
	if p.UserID == uuid.Nil {
		return []UserView{}, nil
	}

	dbUUID := pgtype.UUID{Bytes: p.UserID, Valid: true}
	var views []UserView

	switch p.Status {
	case StatusFriends, StatusOnline:
		rows, err := s.store.RelationshipsListFriendsByUser(ctx, dbUUID)
		if err != nil {
			return nil, apperr.NewDBError(err)
		}
		if len(rows) == 0 {
			return []UserView{}, nil
		}

		// 1. Bulk-fetch real-time activities from Redis
		userIDs := make([]string, len(rows))
		for i, row := range rows {
			userIDs[i] = uuid.UUID(row.RelatedUserID.Bytes).String()
		}

		realtimeStatuses, err := s.presence.GetBulkActivity(ctx, userIDs)
		if err != nil {
			realtimeStatuses = map[string]presence.Activity{}
		}

		// 2. Resolve final presence state and map results
		for _, row := range rows {
			idStr := uuid.UUID(row.RelatedUserID.Bytes).String()

			// Resolution pipeline: Check Redis activity first, fallback to DB default status
			finalActivity := presence.Activity(row.UserStatus)
			if redisStatus, exists := realtimeStatuses[idStr]; exists {
				finalActivity = redisStatus
			}

			// If specific filter requested, prune offline friends early
			if p.Status == StatusOnline && finalActivity == presence.StatusOffline {
				continue
			}

			views = append(views, UserView{
				UserID:       row.RelatedUserID.Bytes,
				Username:     row.Username,
				ActionUserID: row.ActionUserID.Bytes,
				Status:       Status(row.Status),
				Activity:     finalActivity,
				CreatedAt:    row.CreatedAt.Time,
			})
		}

	case StatusBlocked:
		rows, err := s.store.RelationshipsListBlockedByUser(ctx, dbUUID)
		if err != nil {
			return nil, apperr.NewDBError(err)
		}
		for _, row := range rows {
			// Privacy Guardrail: Blocked users should always resolve as completely offline
			// to avoid leaking online status data across mutual servers or cached profiles.
			views = append(views, UserView{
				UserID:       row.RelatedUserID.Bytes,
				Username:     row.Username,
				ActionUserID: row.ActionUserID.Bytes,
				Status:       Status(row.Status),
				Activity:     presence.StatusOffline,
				CreatedAt:    row.CreatedAt.Time,
			})
		}

	case StatusPending:
		rows, err := s.store.RelationshipsListPendingByUser(ctx, dbUUID)
		if err != nil {
			return nil, apperr.NewDBError(err)
		}
		for _, row := range rows {
			views = append(views, UserView{
				UserID:       row.RelatedUserID.Bytes,
				Username:     row.Username,
				ActionUserID: row.ActionUserID.Bytes,
				Status:       Status(row.Status),
				Activity:     presence.Activity(row.UserStatus),
				CreatedAt:    row.CreatedAt.Time,
			})
		}

	case StatusAll, "":
		rows, err := s.store.RelationshipsListByUser(ctx, dbUUID)
		if err != nil {
			return nil, apperr.NewDBError(err)
		}
		if len(rows) == 0 {
			return []UserView{}, nil
		}

		// 1. Bulk-fetch real-time activities from Redis
		userIDs := make([]string, len(rows))
		for i, row := range rows {
			userIDs[i] = uuid.UUID(row.RelatedUserID.Bytes).String()
		}
		realtimeStatuses, _ := s.presence.GetBulkActivity(ctx, userIDs)

		// 2. Map rows safely using the dedicated NewUserView constructor
		for _, row := range rows {
			idStr := uuid.UUID(row.RelatedUserID.Bytes).String()

			finalActivity := presence.Activity(row.UserStatus)
			if redisStatus, exists := realtimeStatuses[idStr]; exists {
				finalActivity = redisStatus
			}

			views = append(views, NewUserView(row, finalActivity))
		}

	default:
		return nil, apperr.New(apperr.CodeBadRequest, "invalid relationship status filter")
	}

	if views == nil {
		views = []UserView{}
	}

	return views, nil
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
