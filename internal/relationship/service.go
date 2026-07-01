package relationship

import (
	"context"
	"errors"

	"bonfire-api/internal/apperr"
	"bonfire-api/internal/presence"
	"bonfire-api/internal/repository"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// --- relationship service ---

type Service struct {
	store    repository.Store
	presence PresenceProvider
}

func NewService(store repository.Store, presence PresenceProvider) *Service {
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

func (s *Service) List(ctx context.Context, p ListParams) ([]View, error) {
	if p.UserID == uuid.Nil {
		return []View{}, nil
	}

	dbUUID := pgtype.UUID{Bytes: p.UserID, Valid: true}
	var rows []repository.RelationshipPerspective
	var err error

	// 1. Route to the optimized view target based on the filter criteria
	switch p.Status {
	case StatusFriends, StatusOnline:
		rows, err = s.store.RelationshipsListFriendsByUserID(ctx, dbUUID)
	case StatusBlocked:
		rows, err = s.store.RelationshipsListBlockedByUserID(ctx, dbUUID)
	case StatusPending:
		rows, err = s.store.RelationshipsListPendingByUserID(ctx, dbUUID)
	case StatusAll, "":
		rows, err = s.store.RelationshipsListByUserID(ctx, dbUUID)
	default:
		return nil, apperr.New(apperr.CodeBadRequest, "invalid relationship status filter")
	}

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return []View{}, nil
		}
		return nil, apperr.NewDBError(err)
	}

	if len(rows) == 0 {
		return []View{}, nil
	}

	// 2. Hydrate presence details and compile down to public views
	return s.hydratePerspectivesPipeline(ctx, rows, p.Status)
}

// Hydration and processing pipeline
func (s *Service) hydratePerspectivesPipeline(ctx context.Context, rows []repository.RelationshipPerspective, filter Status) ([]View, error) {
	peerIDs := make([]string, len(rows))
	for i, row := range rows {
		peerIDs[i] = uuid.UUID(row.PeerID.Bytes).String()
	}

	realtimeStatuses, err := s.presence.GetBulkActivity(ctx, peerIDs)
	if err != nil {
		realtimeStatuses = map[string]presence.Activity{}
	}

	views := make([]View, 0, len(rows))

	for _, row := range rows {
		peerIDStr := uuid.UUID(row.PeerID.Bytes).String()

		finalActivity := presence.Activity(row.UserStatus)
		if redisStatus, exists := realtimeStatuses[peerIDStr]; exists {
			finalActivity = redisStatus
		}

		switch filter {
		case StatusOnline:
			if finalActivity == presence.StatusOffline || finalActivity == presence.StatusInvisible {
				continue
			}
		case StatusBlocked:
			finalActivity = presence.StatusOffline
		}

		views = append(views, NewView(row, finalActivity))
	}

	return views, nil
}

// ==========================================
// UPSERT / UPDATE
// ==========================================

// --- relationship service SendFriendRequest ---

type SendFriendRequestParams struct {
	ActorID uuid.UUID
	PeerID  uuid.UUID
}

func (s *Service) SendFriendRequest(ctx context.Context, p SendFriendRequestParams) error {
	if p.ActorID == p.PeerID {
		return apperr.New(apperr.CodeBadRequest, "cannot add yourself as a friend")
	}

	u1, u2 := orderUUIDs(p.ActorID, p.PeerID)

	relRow, err := s.store.RelationshipGet(ctx, repository.RelationshipGetParams{
		User1ID: pgtype.UUID{Bytes: u1, Valid: true},
		User2ID: pgtype.UUID{Bytes: u2, Valid: true},
	})

	// 1. Handle the case where no relationship exists yet
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			_, err = s.store.RelationshipUpsert(ctx, repository.RelationshipUpsertParams{
				User1ID: pgtype.UUID{Bytes: u1, Valid: true},
				User2ID: pgtype.UUID{Bytes: u2, Valid: true},
				Type:    0, // 0 = Pending
				ActorID: pgtype.UUID{Bytes: p.ActorID, Valid: true},
			})
			if err != nil {
				return apperr.NewDBError(err)
			}
			return nil
		}
		return apperr.NewDBError(err)
	}

	// 2. State machine constraints for existing records
	switch relRow.Type {
	case 1: // 1 = Friends
		return apperr.New(apperr.CodeBadRequest, "already friends")

	case 2: // 2 = Blocked
		// Generic response preserves the privacy shield by obfuscating who blocked whom
		return apperr.New(apperr.CodeForbidden, "cannot interact with this user")

	case 0: // 0 = Pending
		if uuid.UUID(relRow.ActorID.Bytes) != p.ActorID {
			// Implicit Match: The other person already sent a request to the actor.
			// Instead of failing, auto-upgrade the connection to friends by executing an accept.
			return s.AcceptFriendRequest(ctx, AcceptFriendRequestParams{
				ActorID: p.ActorID,
				PeerID:  p.PeerID,
			})
		}
		return apperr.New(apperr.CodeBadRequest, "friend request already pending")
	}

	return nil
}

// --- relationship service AcceptFriendRequest ---

type AcceptFriendRequestParams struct {
	ActorID uuid.UUID
	PeerID  uuid.UUID
}

func (s *Service) AcceptFriendRequest(ctx context.Context, p AcceptFriendRequestParams) error {
	u1, u2 := orderUUIDs(p.ActorID, p.PeerID)

	// 1. Fetch current relationship state
	rel, err := s.store.RelationshipGetForUpdate(ctx, repository.RelationshipGetForUpdateParams{
		User1ID: pgtype.UUID{Bytes: u1, Valid: true},
		User2ID: pgtype.UUID{Bytes: u2, Valid: true},
	})
	if err != nil {
		// Clean handling: if no row exists, there's obviously no request to accept
		if errors.Is(err, pgx.ErrNoRows) {
			return apperr.New(apperr.CodeBadRequest, "no pending request to accept")
		}
		return apperr.NewDBError(err)
	}

	// 2. State check: Verify the current relationship is actually pending (0)
	if rel.Type != 0 { // 0 = Pending
		return apperr.New(apperr.CodeBadRequest, "no pending request to accept")
	}

	// 3. Direction check: Ensure the user accepting isn't the one who sent it
	if uuid.UUID(rel.ActorID.Bytes) == p.ActorID {
		return apperr.New(apperr.CodeForbidden, "cannot accept your own request")
	}

	// 4. Update the row to friends (1). The actor_id becomes the user who accepted.
	_, err = s.store.RelationshipUpsert(ctx, repository.RelationshipUpsertParams{
		User1ID: pgtype.UUID{Bytes: u1, Valid: true},
		User2ID: pgtype.UUID{Bytes: u2, Valid: true},
		Type:    1, // 1 = Friends
		ActorID: pgtype.UUID{Bytes: p.ActorID, Valid: true},
	})
	if err != nil {
		return apperr.NewDBError(err)
	}

	return nil
}

// --- relationship service Block ---

type BlockParams struct {
	ActorID uuid.UUID
	PeerID  uuid.UUID
}

func (s *Service) Block(ctx context.Context, p BlockParams) error {
	if p.ActorID == p.PeerID {
		return apperr.New(apperr.CodeBadRequest, "cannot block yourself")
	}

	u1, u2 := orderUUIDs(p.ActorID, p.PeerID)

	// 1. Fetch current relationship state
	rel, err := s.store.RelationshipGet(ctx, repository.RelationshipGetParams{
		User1ID: pgtype.UUID{Bytes: u1, Valid: true},
		User2ID: pgtype.UUID{Bytes: u2, Valid: true},
	})

	// 2. Privacy Shield: Prevent overwriting an existing block
	// If a block (2) already exists, and the current actor is NOT the person who
	// initiated it, it means the peer blocked them first. We return nil to
	// fake a success, masking the active block state from the caller.
	if err == nil && rel.Type == 2 {
		if uuid.UUID(rel.ActorID.Bytes) != p.ActorID {
			return nil
		}
	} else if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return apperr.NewDBError(err)
	}

	// 3. Upsert the relationship to a blocked state owned by the actor
	_, err = s.store.RelationshipUpsert(ctx, repository.RelationshipUpsertParams{
		User1ID: pgtype.UUID{Bytes: u1, Valid: true},
		User2ID: pgtype.UUID{Bytes: u2, Valid: true},
		Type:    2, // 2 = Blocked
		ActorID: pgtype.UUID{Bytes: p.ActorID, Valid: true},
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
	ActorID uuid.UUID
	PeerID  uuid.UUID
}

func (s *Service) Delete(ctx context.Context, p DeleteParams) error {
	u1, u2 := orderUUIDs(p.ActorID, p.PeerID)

	// Single roundtrip atomic execution. Deletes the relation safely
	// ONLY if it isn't a block, OR if it IS a block owned by the caller.
	_, err := s.store.RelationshipDeleteVerified(ctx, repository.RelationshipDeleteVerifiedParams{
		User1ID: pgtype.UUID{Bytes: u1, Valid: true},
		User2ID: pgtype.UUID{Bytes: u2, Valid: true},
		ActorID: pgtype.UUID{Bytes: p.ActorID, Valid: true},
	})

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// To determine if it failed because it never existed (OK) or because
			// it was a foreign block (Forbidden), verify if the raw record actually exists.
			_, fallbackErr := s.store.RelationshipGet(ctx, repository.RelationshipGetParams{
				User1ID: pgtype.UUID{Bytes: u1, Valid: true},
				User2ID: pgtype.UUID{Bytes: u2, Valid: true},
			})
			if fallbackErr != nil {
				if errors.Is(fallbackErr, pgx.ErrNoRows) {
					return nil // Idempotent: Row didn't exist anyway
				}
				return apperr.NewDBError(fallbackErr)
			}
			// Row exists but wasn't deleted by the query condition -> foreign block guardrail caught it
			return apperr.New(apperr.CodeForbidden, "cannot modify this relationship")
		}
		return apperr.NewDBError(err)
	}

	return nil
}
