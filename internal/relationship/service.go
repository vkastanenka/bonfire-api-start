package relationship

import (
	"context"
	"errors"
	"time"

	"bonfire-api/internal/errs"

	"github.com/google/uuid"
)

type Repository interface {
	Get(ctx context.Context, user1ID uuid.UUID, user2ID uuid.UUID) (*Relationship, error)
	GetForUpdate(ctx context.Context, user1ID uuid.UUID, user2ID uuid.UUID) (*Relationship, error)
	Upsert(ctx context.Context, rel *Relationship) error
	Delete(ctx context.Context, user1ID uuid.UUID, user2ID uuid.UUID) error
	DeleteVerified(ctx context.Context, user1ID uuid.UUID, user2ID uuid.UUID, actorID uuid.UUID) error
	GetPerspective(ctx context.Context, userID uuid.UUID, peerID uuid.UUID) (*Perspective, error)
	ListPerspectives(ctx context.Context, userID uuid.UUID, filterVariant *Variant) ([]Perspective, error)
}

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{
		repo: repo,
	}
}

// ListPerspectives retrieves all relationship projections for a given user (e.g., Friends list, Pending list, Blocked list).
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

// GetPerspective fetches a specific UI projection showing how userID views peerID.
func (s *Service) GetPerspective(ctx context.Context, userID, peerID uuid.UUID) (*Perspective, error) {
	if userID == peerID {
		return nil, errs.InvalidArgument("cannot get perspective for oneself")
	}

	perspective, err := s.repo.GetPerspective(ctx, userID, peerID)
	if err != nil {
		if errs.IsNotFound(err) {
			return nil, errs.NotFound("relationship projection not found").Wrap(err)
		}
		return nil, err
	}
	return perspective, nil
}

// SendFriendRequest initiates a request or auto-accepts an incoming request from the target user.
func (s *Service) SendFriendRequest(ctx context.Context, actorID, targetID uuid.UUID) error {
	if actorID == targetID {
		return errs.InvalidArgument("cannot add yourself as a friend").Wrap(ErrSelfRelationship)
	}

	u1, u2 := sortUserIDs(actorID, targetID)

	// Fetch with row-level lock to prevent concurrent request race conditions
	rel, err := s.repo.GetForUpdate(ctx, u1, u2)
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
		return errs.AlreadyExists("already friends with this user")

	case VariantBlocked:
		// Discord rule: You cannot send requests to someone if either party has blocked the other
		return errs.PermissionDenied("cannot interact with this user").Wrap(ErrRelationshipBlocked)

	case VariantPending:
		// Cross-request scenario: Peer already sent a request to actor, auto-accept it!
		if rel.ActorID() != actorID {
			return s.acceptPendingRequest(ctx, rel, actorID)
		}
		return errs.AlreadyExists("friend request already pending")
	}

	return nil
}

// AcceptFriendRequest explicitly accepts a pending incoming friend request.
func (s *Service) AcceptFriendRequest(ctx context.Context, actorID, peerID uuid.UUID) error {
	if actorID == peerID {
		return errs.InvalidArgument("cannot accept friend request from yourself").Wrap(ErrSelfRelationship)
	}

	u1, u2 := sortUserIDs(actorID, peerID)

	rel, err := s.repo.GetForUpdate(ctx, u1, u2)
	if err != nil {
		if errs.IsNotFound(err) {
			return errs.NotFound("no pending request to accept").Wrap(err)
		}
		return err
	}

	return s.acceptPendingRequest(ctx, rel, actorID)
}

// Block places a block on a user, overriding any existing friend or pending state.
func (s *Service) Block(ctx context.Context, actorID, peerID uuid.UUID) error {
	if actorID == peerID {
		return errs.InvalidArgument("cannot block yourself").Wrap(ErrSelfRelationship)
	}

	u1, u2 := sortUserIDs(actorID, peerID)

	rel, err := s.repo.GetForUpdate(ctx, u1, u2)
	if err != nil && !errs.IsNotFound(err) {
		return err
	}

	if errs.IsNotFound(err) {
		rel = Reconstitute(u1, u2, actorID, VariantBlocked, time.Now().UTC(), time.Now().UTC())
	} else {
		// Invariant Guard: If already blocked by the OTHER user, do not overwrite their block actor ID
		if rel.IsBlocked() && rel.ActorID() != actorID {
			// Maintain the original blocker's authority
			return nil
		}

		if err := rel.Block(actorID); err != nil {
			return errs.InvalidArgument(err.Error()).Wrap(err)
		}
	}

	return s.repo.Upsert(ctx, rel)
}

// UnfriendOrCancelRequest verifies permissions before removing a friendship or canceling an outgoing/incoming request.
func (s *Service) DeleteVerified(ctx context.Context, actorID, peerID uuid.UUID) error {
	if actorID == peerID {
		return errs.InvalidArgument("cannot target yourself").Wrap(ErrSelfRelationship)
	}

	u1, u2 := sortUserIDs(actorID, peerID)

	err := s.repo.DeleteVerified(ctx, u1, u2, actorID)
	if err != nil {
		if errs.IsNotFound(err) {
			return errs.NotFound("relationship not found").Wrap(err)
		}
		// If SQL query rejected deletion because the actor was blocked by peer
		if errors.Is(err, ErrRelationshipBlocked) {
			return errs.PermissionDenied("cannot modify blocked relationship").Wrap(err)
		}
		return err
	}
	return nil
}

// Delete forcefully removes a relationship (System/Admin scope).
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

// Private helper to encapsulate acceptance logic and mutation persistence.
func (s *Service) acceptPendingRequest(ctx context.Context, rel *Relationship, actorID uuid.UUID) error {
	if err := rel.Accept(actorID); err != nil {
		if errors.Is(err, ErrCannotAccept) {
			return errs.PermissionDenied("cannot accept your own outgoing friend request").Wrap(err)
		}
		return errs.InvalidArgument(err.Error()).Wrap(err)
	}

	return s.repo.Upsert(ctx, rel)
}
