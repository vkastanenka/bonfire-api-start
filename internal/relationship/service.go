package relationship

import (
	"bonfire-api/internal/apperr"
	"bonfire-api/internal/repository"
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type Store interface {
	repository.Querier
}

type Service struct {
	store Store
}

func NewService(store Store) *Service {
	return &Service{
		store: store,
	}
}

func (s *Service) List(ctx context.Context, uid uuid.UUID, t Type) ([]Relationship, error) {
	if uid == uuid.Nil {
		return []Relationship{}, nil
	}

	dbUUID := pgtype.UUID{Bytes: uid, Valid: true}
	var rows []repository.Relationship
	var err error

	switch t {
	case TypeFriends:
		rows, err = s.store.RelationshipsListFriendsByUserID(ctx, dbUUID)
	case TypeBlocked:
		rows, err = s.store.RelationshipsListBlockedByUserID(ctx, dbUUID)
	case TypePending:
		rows, err = s.store.RelationshipsListPendingByUserID(ctx, dbUUID)
	case TypeUnknown:
		rows, err = s.store.RelationshipsListByUserID(ctx, dbUUID)
	default:
		return nil, apperr.NewBadRequest(nil, "invalid relationship status filter")
	}

	if err != nil {
		if repository.IsNotFoundError(err) {
			return []Relationship{}, nil
		}
		return nil, repository.NewError(err, repository.ScopeRelationship)
	}

	if len(rows) == 0 {
		return []Relationship{}, nil
	}

	relationships := make([]Relationship, len(rows))
	for i, row := range rows {
		relationships[i] = FromRepository(row)
	}

	return relationships, nil
}

func (s *Service) SendFriendRequest(ctx context.Context, aid uuid.UUID, pid uuid.UUID) error {
	if aid == pid {
		return apperr.NewBadRequest(nil, "cannot add yourself as a friend")
	}

	relRow, err := s.store.RelationshipGet(ctx, repository.RelationshipGetParams{
		User1ID: pgtype.UUID{Bytes: aid, Valid: true},
		User2ID: pgtype.UUID{Bytes: pid, Valid: true},
	})

	if err != nil {
		if repository.IsNotFoundError(err) {
			_, err = s.store.RelationshipUpsert(ctx, repository.RelationshipUpsertParams{
				User1ID: pgtype.UUID{Bytes: aid, Valid: true},
				User2ID: pgtype.UUID{Bytes: pid, Valid: true},
				Type:    int16(TypePending), // Aligned: 1 = Pending
				ActorID: pgtype.UUID{Bytes: aid, Valid: true},
			})
			if err != nil {
				return repository.NewError(err, repository.ScopeRelationship)
			}
			return nil
		}
		return repository.NewError(err, repository.ScopeRelationship)
	}

	switch Type(relRow.Type) {
	case TypeFriends: // Aligned: 2 = Friends
		return apperr.NewBadRequest(nil, "already friends")

	case TypeBlocked: // Aligned: 3 = Blocked
		return apperr.NewForbidden(nil, "cannot interact with this user")

	case TypePending: // Aligned: 1 = Pending
		if uuid.UUID(relRow.ActorID.Bytes) != aid {
			return s.AcceptFriendRequest(ctx, aid, pid)
		}
		return apperr.NewBadRequest(nil, "friend request already pending")
	}

	return nil
}

func (s *Service) AcceptFriendRequest(ctx context.Context, aid uuid.UUID, pid uuid.UUID) error {
	rel, err := s.store.RelationshipGetForUpdate(ctx, repository.RelationshipGetForUpdateParams{
		User1ID: pgtype.UUID{Bytes: aid, Valid: true},
		User2ID: pgtype.UUID{Bytes: pid, Valid: true},
	})
	if err != nil {
		if repository.IsNotFoundError(err) {
			return apperr.NewBadRequest(err, "no pending request to accept")
		}
		return repository.NewError(err, repository.ScopeRelationship)
	}

	if Type(rel.Type) != TypePending { // Aligned: 1 = Pending
		return apperr.NewBadRequest(nil, "no pending request to accept")
	}

	if uuid.UUID(rel.ActorID.Bytes) == aid {
		return apperr.NewBadRequest(nil, "cannot accept your own request")
	}

	_, err = s.store.RelationshipUpsert(ctx, repository.RelationshipUpsertParams{
		User1ID: pgtype.UUID{Bytes: aid, Valid: true},
		User2ID: pgtype.UUID{Bytes: pid, Valid: true},
		Type:    int16(TypeFriends), // Aligned: 2 = Friends
		ActorID: pgtype.UUID{Bytes: aid, Valid: true},
	})
	if err != nil {
		return repository.NewError(err, repository.ScopeRelationship)
	}

	return nil
}

func (s *Service) Block(ctx context.Context, aid uuid.UUID, pid uuid.UUID) error {
	if aid == pid {
		return apperr.NewBadRequest(nil, "cannot block yourself")
	}

	rel, err := s.store.RelationshipGet(ctx, repository.RelationshipGetParams{
		User1ID: pgtype.UUID{Bytes: aid, Valid: true},
		User2ID: pgtype.UUID{Bytes: pid, Valid: true},
	})

	if err == nil && Type(rel.Type) == TypeBlocked { // Aligned: 3 = Blocked
		if uuid.UUID(rel.ActorID.Bytes) != aid {
			return nil
		}
	} else if err != nil && !repository.IsNotFoundError(err) {
		return repository.NewError(err, repository.ScopeRelationship)
	}

	_, err = s.store.RelationshipUpsert(ctx, repository.RelationshipUpsertParams{
		User1ID: pgtype.UUID{Bytes: aid, Valid: true},
		User2ID: pgtype.UUID{Bytes: pid, Valid: true},
		Type:    int16(TypeBlocked), // Aligned: 3 = Blocked
		ActorID: pgtype.UUID{Bytes: aid, Valid: true},
	})
	if err != nil {
		return repository.NewError(err, repository.ScopeRelationship)
	}

	return nil
}

func (s *Service) DeleteVerified(ctx context.Context, aid uuid.UUID, pid uuid.UUID) error {
	err := s.store.RelationshipDeleteVerified(ctx, repository.RelationshipDeleteVerifiedParams{
		User1ID: pgtype.UUID{Bytes: aid, Valid: true},
		User2ID: pgtype.UUID{Bytes: pid, Valid: true},
		ActorID: pgtype.UUID{Bytes: aid, Valid: true},
	})
	if err != nil {
		return repository.NewError(err, repository.ScopeRelationship)
	}

	return nil
}

func (s *Service) Delete(ctx context.Context, user1ID uuid.UUID, user2ID uuid.UUID) error {
	err := s.store.RelationshipDelete(ctx, repository.RelationshipDeleteParams{
		User1ID: pgtype.UUID{Bytes: user1ID, Valid: true},
		User2ID: pgtype.UUID{Bytes: user2ID, Valid: true},
	})
	if err != nil {
		return repository.NewError(err, repository.ScopeRelationship)
	}

	return nil
}
