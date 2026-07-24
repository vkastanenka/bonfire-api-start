package relationship

import (
	"context"
	"errors"
	"time"

	"bonfire-api/internal/errs"
	"bonfire-api/internal/outbox"

	"github.com/google/uuid"
)

type OutboxRepository interface {
	Publish(ctx context.Context, variant string, payload any) (*outbox.Event, error)
}

type Repository interface {
	Get(ctx context.Context, user1ID uuid.UUID, user2ID uuid.UUID) (*Relationship, error)
	GetForUpdate(ctx context.Context, user1ID uuid.UUID, user2ID uuid.UUID) (*Relationship, error)
	Upsert(ctx context.Context, rel *Relationship) error
	Delete(ctx context.Context, user1ID uuid.UUID, user2ID uuid.UUID) error
	DeleteVerified(ctx context.Context, user1ID uuid.UUID, user2ID uuid.UUID, actorID uuid.UUID) error
	GetPerspective(ctx context.Context, userID uuid.UUID, peerID uuid.UUID) (*Perspective, error)
	ListPerspectives(ctx context.Context, userID uuid.UUID, filterVariant *Variant) ([]Perspective, error)
}

type Tx interface {
	ExecTx(ctx context.Context, fn func(txCtx context.Context) error) error
}

type Service struct {
	repo   Repository
	outbox OutboxRepository
	tx     Tx
}

func NewService(repo Repository, outbox OutboxRepository, tx Tx) *Service {
	return &Service{
		repo:   repo,
		outbox: outbox,
		tx:     tx,
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

func (s *Service) SendFriendRequest(ctx context.Context, actorID, targetID uuid.UUID) error {
	if actorID == targetID {
		return errs.InvalidArgument("cannot add yourself as a friend").Wrap(ErrSelfRelationship)
	}

	u1, u2 := sortUserIDs(actorID, targetID)

	return s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		// Fetch with row-level lock to prevent concurrent request race conditions inside the transaction
		rel, err := s.repo.GetForUpdate(txCtx, u1, u2)
		if err != nil {
			if errs.IsNotFound(err) {
				newRel, reqErr := Request(actorID, targetID)
				if reqErr != nil {
					return errs.InvalidArgument(reqErr.Error()).Wrap(reqErr)
				}

				if err := s.repo.Upsert(txCtx, newRel); err != nil {
					return err
				}

				// Emit outbox event atomically
				_, err := s.outbox.Publish(txCtx, EventFriendRequestSent, FriendRequestSentPayload{
					ActorID:  actorID,
					TargetID: targetID,
				})
				return err
			}
			return err
		}

		switch rel.Variant() {
		case VariantFriends:
			return errs.AlreadyExists("already friends with this user")

		case VariantBlocked:
			return errs.PermissionDenied("cannot interact with this user").Wrap(ErrRelationshipBlocked)

		case VariantPending:
			// Cross-request scenario: Peer already sent a request to actor, auto-accept it!
			if rel.ActorID() != actorID {
				return s.acceptPendingRequestTx(txCtx, rel, actorID)
			}
			return errs.AlreadyExists("friend request already pending")
		}

		return nil
	})
}

// AcceptFriendRequest explicitly accepts a pending incoming friend request.
func (s *Service) AcceptFriendRequest(ctx context.Context, actorID, peerID uuid.UUID) error {
	if actorID == peerID {
		return errs.InvalidArgument("cannot accept friend request from yourself").Wrap(ErrSelfRelationship)
	}

	u1, u2 := sortUserIDs(actorID, peerID)

	return s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		rel, err := s.repo.GetForUpdate(txCtx, u1, u2)
		if err != nil {
			if errs.IsNotFound(err) {
				return errs.NotFound("no pending request to accept").Wrap(err)
			}
			return err
		}

		return s.acceptPendingRequestTx(txCtx, rel, actorID)
	})
}

// Private helper for transactional acceptance and outbox event publishing.
func (s *Service) acceptPendingRequestTx(ctx context.Context, rel *Relationship, actorID uuid.UUID) error {
	if err := rel.Accept(actorID); err != nil {
		if errors.Is(err, ErrCannotAccept) {
			return errs.PermissionDenied("cannot accept your own outgoing friend request").Wrap(err)
		}
		return errs.InvalidArgument(err.Error()).Wrap(err)
	}

	if err := s.repo.Upsert(ctx, rel); err != nil {
		return err
	}

	// Emit outbox event notifying that the request was accepted
	_, err := s.outbox.Publish(ctx, EventFriendRequestAccepted, FriendRequestAcceptedPayload{
		ActorID:  actorID,
		TargetID: rel.GetPeerID(actorID), // Sends to the person who originated the request
	})
	return err
}

// Block places a block on a user, overriding any existing friend or pending state.
func (s *Service) Block(ctx context.Context, actorID, peerID uuid.UUID) error {
	if actorID == peerID {
		return errs.InvalidArgument("cannot block yourself").Wrap(ErrSelfRelationship)
	}

	u1, u2 := sortUserIDs(actorID, peerID)

	return s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		rel, err := s.repo.GetForUpdate(txCtx, u1, u2)
		if err != nil && !errs.IsNotFound(err) {
			return err
		}

		if errs.IsNotFound(err) {
			rel = Reconstitute(u1, u2, actorID, VariantBlocked, time.Now().UTC(), time.Now().UTC())
		} else {
			if rel.IsBlocked() && rel.ActorID() != actorID {
				return nil
			}

			if err := rel.Block(actorID); err != nil {
				return errs.InvalidArgument(err.Error()).Wrap(err)
			}
		}

		if err := s.repo.Upsert(txCtx, rel); err != nil {
			return err
		}

		// Emit outbox event for blocking
		_, err = s.outbox.Publish(txCtx, EventUserBlocked, UserBlockedPayload{
			ActorID:  actorID,
			TargetID: peerID,
		})
		return err
	})
}

// DeleteVerified verifies permissions before removing a friendship or request.
func (s *Service) DeleteVerified(ctx context.Context, actorID, peerID uuid.UUID) error {
	if actorID == peerID {
		return errs.InvalidArgument("cannot target yourself").Wrap(ErrSelfRelationship)
	}

	u1, u2 := sortUserIDs(actorID, peerID)

	return s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		err := s.repo.DeleteVerified(txCtx, u1, u2, actorID)
		if err != nil {
			if errs.IsNotFound(err) {
				return errs.NotFound("relationship not found").Wrap(err)
			}
			if errors.Is(err, ErrRelationshipBlocked) {
				return errs.PermissionDenied("cannot modify blocked relationship").Wrap(err)
			}
			return err
		}

		// Emit outbox event for removal (unfriend / cancel request)
		_, err = s.outbox.Publish(txCtx, EventRelationshipRemoved, RelationshipRemovedPayload{
			ActorID:  actorID,
			TargetID: peerID,
		})
		return err
	})
}

// Delete forcefully removes a relationship (System/Admin scope).
func (s *Service) Delete(ctx context.Context, user1ID, user2ID uuid.UUID) error {
	u1, u2 := sortUserIDs(user1ID, user2ID)

	return s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		err := s.repo.Delete(txCtx, u1, u2)
		if err != nil {
			if errs.IsNotFound(err) {
				return errs.NotFound("relationship not found").Wrap(err)
			}
			return err
		}

		// Optional: Admin deletion event broadcasted to both parties
		_, err = s.outbox.Publish(txCtx, EventRelationshipRemoved, RelationshipRemovedPayload{
			ActorID:  user1ID,
			TargetID: user2ID,
		})
		return err
	})
}
