package relationship

import (
	"context"
	"time"

	"bonfire-api/internal/errs"

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

func (s *Service) ListPerspectives(ctx context.Context, userID uuid.UUID, filter *Variant) ([]Perspective, error) {
	if filter != nil && !filter.IsValid() {
		return nil, errs.InvalidArgument("invalid relationship status filter")
	}

	perspectives, err := s.repo.ListPerspectives(ctx, userID, filter)
	if err != nil {
		if errs.IsNotFound(err) {
			return []Perspective{}, nil
		}
		return nil, err
	}

	return perspectives, nil
}

func (s *Service) GetPerspective(ctx context.Context, userID, peerID uuid.UUID) (*Perspective, error) {
	perspective, err := s.repo.GetPerspective(ctx, userID, peerID)
	if err != nil {
		if errs.IsNotFound(err) {
			return nil, errs.NotFound("relationship projection not found").Wrap(err)
		}
		return nil, err
	}
	return perspective, nil
}

func (s *Service) SendFriendRequest(ctx context.Context, actorID, targetID uuid.UUID) error {
	if actorID == targetID {
		return errs.InvalidArgument("cannot add yourself as a friend").Wrap(ErrSelfRelationship)
	}

	u1, u2 := sortUserIDs(actorID, targetID)

	rel, err := s.repo.Get(ctx, u1, u2)
	if err != nil {
		if errs.IsNotFound(err) {
			newRel, reqErr := Request(actorID, targetID)
			if reqErr != nil {
				return errs.InvalidArgument(reqErr.Error()).Wrap(reqErr)
			}

			return s.repo.Upsert(ctx, newRel)
		}
		return err
	}

	switch rel.Variant() {
	case VariantFriends:
		return errs.AlreadyExists("already friends")

	case VariantBlocked:
		return errs.PermissionDenied("cannot interact with this user").Wrap(ErrRelationshipBlocked)

	case VariantPending:
		if rel.ActorID() != actorID {
			return s.AcceptFriendRequest(ctx, actorID, targetID)
		}
		return errs.AlreadyExists("friend request already pending")
	}

	return nil
}

func (s *Service) AcceptFriendRequest(ctx context.Context, actorID, peerID uuid.UUID) error {
	u1, u2 := sortUserIDs(actorID, peerID)

	rel, err := s.repo.GetForUpdate(ctx, u1, u2)
	if err != nil {
		if errs.IsNotFound(err) {
			return errs.NotFound("no pending request to accept").Wrap(err)
		}
		return err
	}

	if err := rel.Accept(actorID); err != nil {
		return errs.InvalidArgument(err.Error()).Wrap(err)
	}

	return s.repo.Upsert(ctx, rel)
}

func (s *Service) Block(ctx context.Context, actorID, peerID uuid.UUID) error {
	if actorID == peerID {
		return errs.InvalidArgument("cannot block yourself").Wrap(ErrSelfRelationship)
	}

	u1, u2 := sortUserIDs(actorID, peerID)

	rel, err := s.repo.Get(ctx, u1, u2)
	if err != nil && !errs.IsNotFound(err) {
		return err
	}

	if errs.IsNotFound(err) {
		rel = Reconstitute(u1, u2, actorID, VariantBlocked, time.Now().UTC(), time.Now().UTC())
	} else {
		if err := rel.Block(actorID); err != nil {
			return errs.InvalidArgument(err.Error()).Wrap(err)
		}
	}

	return s.repo.Upsert(ctx, rel)
}

func (s *Service) DeleteVerified(ctx context.Context, actorID, peerID uuid.UUID) error {
	u1, u2 := sortUserIDs(actorID, peerID)

	err := s.repo.DeleteVerified(ctx, u1, u2, actorID)
	if err != nil {
		if errs.IsNotFound(err) {
			return errs.NotFound("relationship not found").Wrap(err)
		}
		return err
	}
	return nil
}

func (s *Service) Delete(ctx context.Context, user1ID, user2ID uuid.UUID) error {
	u1, u2 := sortUserIDs(user1ID, user2ID)

	err := s.repo.Delete(ctx, u1, u2)
	if err != nil {
		if errs.IsNotFound(err) {
			return errs.NotFound("relationship not found").Wrap(err)
		}
		return err
	}
	return nil
}
