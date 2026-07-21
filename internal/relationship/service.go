package relationship

import (
	"context"

	"bonfire-api/internal/apperr"

	"github.com/google/uuid"
)

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{
		repo: repo,
	}
}

func (s *Service) List(ctx context.Context, uid uuid.UUID, t Type) ([]Relationship, error) {
	if !t.Valid() && t != TypeUnknown {
		return nil, apperr.NewInvalidArgument(nil, apperr.WithMsg("invalid relationship status filter"))
	}

	res, err := s.repo.ListByUserID(ctx, uid, t)
	if err != nil {
		if apperr.IsNotFound(err) {
			return []Relationship{}, nil
		}
		return nil, err
	}

	return res, nil
}

func (s *Service) SendFriendRequest(ctx context.Context, aid uuid.UUID, pid uuid.UUID) error {
	if aid == pid {
		return apperr.NewInvalidArgument(nil, apperr.WithMsg("cannot add yourself as a friend"))
	}

	rel, err := s.repo.Get(ctx, GetParams{
		User1ID: aid,
		User2ID: pid,
	})

	if err != nil {
		if apperr.IsNotFound(err) {
			_, err = s.repo.Upsert(ctx, UpsertParams{
				User1ID: aid,
				User2ID: pid,
				Type:    TypePending,
				ActorID: aid,
			})
			return err
		}
		return err
	}

	switch rel.Type {
	case TypeFriends:
		return apperr.NewAlreadyExists(nil, apperr.WithMsg("already friends"))

	case TypeBlocked:
		return apperr.NewPermissionDenied(nil, apperr.WithMsg("cannot interact with this user"))

	case TypePending:
		if rel.ActorID != aid {
			return s.AcceptFriendRequest(ctx, aid, pid)
		}
		return apperr.NewAlreadyExists(nil, apperr.WithMsg("friend request already pending"))
	}

	return nil
}

func (s *Service) AcceptFriendRequest(ctx context.Context, aid uuid.UUID, pid uuid.UUID) error {
	rel, err := s.repo.GetForUpdate(ctx, GetParams{
		User1ID: aid,
		User2ID: pid,
	})
	if err != nil {
		if apperr.IsNotFound(err) {
			return apperr.NewNotFound(err, apperr.WithMsg("no pending request to accept"))
		}
		return err
	}

	if rel.Type != TypePending {
		return apperr.NewFailedPrecondition(nil, apperr.WithMsg("no pending request to accept"))
	}

	if rel.ActorID == aid {
		return apperr.NewInvalidArgument(nil, apperr.WithMsg("cannot accept your own request"))
	}

	_, err = s.repo.Upsert(ctx, UpsertParams{
		User1ID: aid,
		User2ID: pid,
		Type:    TypeFriends,
		ActorID: aid,
	})
	return err
}

func (s *Service) Block(ctx context.Context, aid uuid.UUID, pid uuid.UUID) error {
	if aid == pid {
		return apperr.NewInvalidArgument(nil, apperr.WithMsg("cannot block yourself"))
	}

	rel, err := s.repo.Get(ctx, GetParams{
		User1ID: aid,
		User2ID: pid,
	})

	if err == nil && rel.Type == TypeBlocked {
		if rel.ActorID != aid {
			return nil
		}
	} else if err != nil && !apperr.IsNotFound(err) {
		return err
	}

	_, err = s.repo.Upsert(ctx, UpsertParams{
		User1ID: aid,
		User2ID: pid,
		Type:    TypeBlocked,
		ActorID: aid,
	})
	return err
}

func (s *Service) DeleteVerified(ctx context.Context, aid uuid.UUID, pid uuid.UUID) error {
	return s.repo.DeleteVerified(ctx, DeleteVerifiedParams{
		User1ID: aid,
		User2ID: pid,
		ActorID: aid,
	})
}

func (s *Service) Delete(ctx context.Context, user1ID uuid.UUID, user2ID uuid.UUID) error {
	return s.repo.Delete(ctx, DeleteParams{
		User1ID: user1ID,
		User2ID: user2ID,
	})
}
