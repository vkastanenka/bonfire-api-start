package relationship

import (
	"context"
	"time"

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

// ListPerspectives retrieves user-centric relationship projections (read models).
func (s *Service) ListPerspectives(ctx context.Context, userID uuid.UUID, filter *Variant) ([]Perspective, error) {
	if filter != nil && !filter.IsValid() {
		return nil, apperr.NewInvalidArgument(nil, apperr.WithMsg("invalid relationship status filter"))
	}

	perspectives, err := s.repo.ListPerspectives(ctx, userID, filter)
	if err != nil {
		if apperr.IsNotFound(err) {
			return []Perspective{}, nil
		}
		return nil, err
	}

	return perspectives, nil
}

// GetPerspective retrieves a single relationship projection for a specific user and peer.
func (s *Service) GetPerspective(ctx context.Context, userID, peerID uuid.UUID) (*Perspective, error) {
	perspective, err := s.repo.GetPerspective(ctx, userID, peerID)
	if err != nil {
		if apperr.IsNotFound(err) {
			return nil, apperr.NewNotFound(err, apperr.WithMsg("relationship projection not found"))
		}
		return nil, err
	}
	return perspective, nil
}

// SendFriendRequest initiates a request or auto-accepts an existing inverse pending request.
func (s *Service) SendFriendRequest(ctx context.Context, actorID, targetID uuid.UUID) error {
	if actorID == targetID {
		return apperr.NewInvalidArgument(ErrSelfRelationship, apperr.WithMsg("cannot add yourself as a friend"))
	}

	u1, u2 := sortUserIDs(actorID, targetID)

	rel, err := s.repo.Get(ctx, u1, u2)
	if err != nil {
		if apperr.IsNotFound(err) {
			// Construct fresh relationship aggregate
			newRel, reqErr := Request(actorID, targetID)
			if reqErr != nil {
				return apperr.NewInvalidArgument(reqErr, apperr.WithMsg(reqErr.Error()))
			}

			return s.repo.Upsert(ctx, newRel)
		}
		return err
	}

	switch rel.Variant() {
	case VariantFriends:
		return apperr.NewAlreadyExists(nil, apperr.WithMsg("already friends"))

	case VariantBlocked:
		return apperr.NewPermissionDenied(ErrRelationshipBlocked, apperr.WithMsg("cannot interact with this user"))

	case VariantPending:
		// If the recipient sends a request back, turn it into an acceptance
		if rel.ActorID() != actorID {
			return s.AcceptFriendRequest(ctx, actorID, targetID)
		}
		return apperr.NewAlreadyExists(nil, apperr.WithMsg("friend request already pending"))
	}

	return nil
}

// AcceptFriendRequest transitions a pending relationship into a friendship.
func (s *Service) AcceptFriendRequest(ctx context.Context, actorID, peerID uuid.UUID) error {
	u1, u2 := sortUserIDs(actorID, peerID)

	rel, err := s.repo.GetForUpdate(ctx, u1, u2)
	if err != nil {
		if apperr.IsNotFound(err) {
			return apperr.NewNotFound(err, apperr.WithMsg("no pending request to accept"))
		}
		return err
	}

	// Apply aggregate transition logic
	if err := rel.Accept(actorID); err != nil {
		return apperr.NewInvalidArgument(err, apperr.WithMsg(err.Error()))
	}

	return s.repo.Upsert(ctx, rel)
}

// Block transitions a relationship state into blocked by the acting user.
func (s *Service) Block(ctx context.Context, actorID, peerID uuid.UUID) error {
	if actorID == peerID {
		return apperr.NewInvalidArgument(ErrSelfRelationship, apperr.WithMsg("cannot block yourself"))
	}

	u1, u2 := sortUserIDs(actorID, peerID)

	rel, err := s.repo.Get(ctx, u1, u2)
	if err != nil && !apperr.IsNotFound(err) {
		return err
	}

	if apperr.IsNotFound(err) {
		// Create a new relationship aggregate directly in blocked state
		rel = Reconstitute(u1, u2, actorID, VariantBlocked, time.Now().UTC(), time.Now().UTC())
	} else {
		// Mutate existing relationship aggregate
		if err := rel.Block(actorID); err != nil {
			return apperr.NewInvalidArgument(err, apperr.WithMsg(err.Error()))
		}
	}

	return s.repo.Upsert(ctx, rel)
}

// DeleteVerified removes a relationship while enforcing blocking safeguards at the database level.
func (s *Service) DeleteVerified(ctx context.Context, actorID, peerID uuid.UUID) error {
	u1, u2 := sortUserIDs(actorID, peerID)

	err := s.repo.DeleteVerified(ctx, u1, u2, actorID)
	if err != nil {
		if apperr.IsNotFound(err) {
			return apperr.NewNotFound(err, apperr.WithMsg("relationship not found"))
		}
		return err
	}
	return nil
}

// Delete removes a relationship aggregate given two participant IDs.
func (s *Service) Delete(ctx context.Context, user1ID, user2ID uuid.UUID) error {
	u1, u2 := sortUserIDs(user1ID, user2ID)

	err := s.repo.Delete(ctx, u1, u2)
	if err != nil {
		if apperr.IsNotFound(err) {
			return apperr.NewNotFound(err, apperr.WithMsg("relationship not found"))
		}
		return err
	}
	return nil
}
