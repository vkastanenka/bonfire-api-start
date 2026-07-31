package relationship

import (
	"context"
	"errors"
	"time"

	"bonfire-api/internal/channel"
	"bonfire-api/internal/errs"
	"bonfire-api/internal/outbox"

	"github.com/google/uuid"
)

type ChannelRepository interface {
	Create(ctx context.Context, ch *channel.Channel) (*channel.Channel, error)
	MemberAddBatch(ctx context.Context, members []*channel.Member) error
}

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
	repo    Repository
	channel ChannelRepository
	outbox  OutboxRepository
	tx      Tx
}

func NewService(repo Repository, channel ChannelRepository, outbox OutboxRepository, tx Tx) *Service {
	return &Service{
		repo:    repo,
		channel: channel,
		outbox:  outbox,
		tx:      tx,
	}
}

// AcceptFriendRequest explicitly accepts a pending incoming friend request.
func (s *Service) AcceptFriendRequest(ctx context.Context, rawActorID, rawPeerID uuid.UUID) error {
	actorID, err := NewUserID(rawActorID)
	if err != nil {
		return errs.InvalidArgument("invalid actor id")
	}

	peerID, err := NewUserID(rawPeerID)
	if err != nil {
		return errs.InvalidArgument("invalid peer id")
	}

	if actorID == peerID {
		return errs.InvalidArgument("cannot accept friend request from yourself")
	}

	u1, u2 := sortUserIDs(actorID, peerID)

	return s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		rel, err := s.repo.GetForUpdate(txCtx, u1.UUID(), u2.UUID())
		if err != nil {
			if errs.IsNotFound(err) {
				return errs.NotFound("no pending request to accept").Wrap(err)
			}
			return err
		}

		return s.acceptPendingRequestTx(txCtx, rel, actorID.UUID())
	})
}

// Block places a block on a user, overriding any existing friend or pending state.
func (s *Service) Block(ctx context.Context, rawActorID, rawPeerID uuid.UUID) error {
	actorID, err := NewUserID(rawActorID)
	if err != nil {
		return errs.InvalidArgument("invalid actor id")
	}

	peerID, err := NewUserID(rawPeerID)
	if err != nil {
		return errs.InvalidArgument("invalid peer id")
	}

	if actorID == peerID {
		return errs.InvalidArgument("cannot block yourself")
	}

	u1, u2 := sortUserIDs(actorID, peerID)

	return s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		var rel *Relationship

		fetchedRel, err := s.repo.GetForUpdate(txCtx, u1.UUID(), u2.UUID())
		if err != nil && !errs.IsNotFound(err) {
			return err
		}

		if errs.IsNotFound(err) {
			now := time.Now().UTC()
			rel, err = Reconstitute(
				u1.UUID(),
				u2.UUID(),
				actorID.UUID(),
				nil, // rawChannelID (*uuid.UUID) - nil since VariantBlocked isn't VariantFriends
				uint8(VariantBlocked),
				now,
				now,
			)
			if err != nil {
				return errs.InvalidArgument(err.Error()).Wrap(err)
			}
		} else {
			rel = fetchedRel
			// If already blocked by the OTHER party, do nothing (don't overwrite their block ownership)
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
			ActorID:  actorID.UUID(),
			TargetID: peerID.UUID(),
		})
		return err
	})
}

// DeleteVerified verifies permissions before removing a friendship or request.
func (s *Service) DeleteVerified(ctx context.Context, rawActorID, rawPeerID uuid.UUID) error {
	actorID, err := NewUserID(rawActorID)
	if err != nil {
		return errs.InvalidArgument("invalid actor id")
	}

	peerID, err := NewUserID(rawPeerID)
	if err != nil {
		return errs.InvalidArgument("invalid peer id")
	}

	if actorID == peerID {
		return errs.InvalidArgument("cannot target yourself").Wrap(ErrSelfRelationship)
	}

	u1, u2 := sortUserIDs(actorID, peerID)

	return s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		err := s.repo.DeleteVerified(txCtx, u1.UUID(), u2.UUID(), actorID.UUID())
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
			ActorID:  actorID.UUID(),
			TargetID: peerID.UUID(),
		})
		return err
	})
}

func (s *Service) GetPerspective(ctx context.Context, rawUserID, rawPeerID uuid.UUID) (*Perspective, error) {
	userID, err := NewUserID(rawUserID)
	if err != nil {
		return nil, errs.InvalidArgument("invalid user id")
	}

	peerID, err := NewUserID(rawPeerID)
	if err != nil {
		return nil, errs.InvalidArgument("invalid peer id")
	}

	if userID == peerID {
		return nil, errs.InvalidArgument("cannot get your own perspective")
	}

	perspective, err := s.repo.GetPerspective(ctx, userID.UUID(), peerID.UUID())
	if err != nil {
		return nil, err
	}

	return perspective, nil
}

// func (s *Service) ListPerspectives(ctx context.Context, userID uuid.UUID, filter *Variant) ([]Perspective, error) {
// 	if filter != nil && !filter.IsValid() {
// 		return nil, errs.InvalidArgument("invalid relationship status filter")
// 	}

// 	perspectives, err := s.repo.ListPerspectives(ctx, userID, filter)
// 	if err != nil {
// 		if errs.IsNotFound(err) {
// 			return []Perspective{}, nil
// 		}
// 		return nil, err
// 	}

// 	return perspectives, nil
// }

func (s *Service) SendFriendRequest(ctx context.Context, rawActorID, rawTargetID uuid.UUID) error {
	actorID, err := NewUserID(rawActorID)
	if err != nil {
		return errs.InvalidArgument("invalid actor id")
	}

	targetID, err := NewUserID(rawTargetID)
	if err != nil {
		return errs.InvalidArgument("invalid peer id")
	}

	if actorID == targetID {
		return errs.InvalidArgument("cannot friend yourself")
	}

	u1, u2 := sortUserIDs(actorID, targetID)

	return s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		// Fetch with row-level lock to prevent concurrent request race conditions inside the transaction
		rel, err := s.repo.GetForUpdate(txCtx, u1.UUID(), u2.UUID())
		if err != nil {
			if errs.IsNotFound(err) {
				newRel, reqErr := New(actorID, targetID)
				if reqErr != nil {
					return errs.InvalidArgument(reqErr.Error()).Wrap(reqErr)
				}

				if err := s.repo.Upsert(txCtx, newRel); err != nil {
					return err
				}

				// Emit outbox event atomically
				_, err := s.outbox.Publish(txCtx, EventFriendRequestSent, FriendRequestSentPayload{
					ActorID:  actorID.UUID(),
					TargetID: targetID.UUID(),
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
				return s.acceptPendingRequestTx(txCtx, rel, actorID.UUID())
			}
			return errs.AlreadyExists("friend request already pending")
		}

		return nil
	})
}

// Private helper for transactional acceptance, DM channel creation, and outbox event publishing.
func (s *Service) acceptPendingRequestTx(ctx context.Context, rel *Relationship, actorID uuid.UUID) error {
	actID, err := NewUserID(actorID)
	if err != nil {
		return errs.InvalidArgument("invalid actor id")
	}

	var channelID ChannelID

	// 1. Check if a DM channel already exists for this relationship (e.g., re-friending)
	if existingChID := rel.ChannelID(); existingChID != nil {
		channelID = *existingChID
	} else {
		// 2. Instantiate new 1:1 Direct Message Channel entity (TypeDirect)
		ch, err := channel.New(channel.TypeDirect, nil, nil)
		if err != nil {
			return errs.InvalidArgument("failed to construct DM channel").Wrap(err)
		}

		// 3. Persist Channel record inside current transaction
		createdCh, err := s.channel.Create(ctx, ch)
		if err != nil {
			return err
		}

		// 4. Construct & batch-add members
		chUUID := createdCh.ID().UUID()
		u1ID := rel.User1ID().UUID()
		u2ID := rel.User2ID().UUID()

		m1, err := channel.NewMember(chUUID, u1ID)
		if err != nil {
			return errs.InvalidArgument("invalid member 1").Wrap(err)
		}

		m2, err := channel.NewMember(chUUID, u2ID)
		if err != nil {
			return errs.InvalidArgument("invalid member 2").Wrap(err)
		}

		if err := s.channel.MemberAddBatch(ctx, []*channel.Member{m1, m2}); err != nil {
			return err
		}

		channelID = ChannelID(createdCh.ID())
	}

	// 5. Transition relationship state to VariantFriends with the active channel ID
	if err := rel.Accept(actID, channelID); err != nil {
		if errors.Is(err, ErrCannotAccept) {
			return errs.PermissionDenied("cannot accept your own outgoing friend request").Wrap(err)
		}
		return errs.InvalidArgument(err.Error()).Wrap(err)
	}

	// 6. Upsert updated relationship state
	if err := s.repo.Upsert(ctx, rel); err != nil {
		return err
	}

	// 7. Emit outbox event
	peerID := rel.GetPeerID(actID)
	_, err = s.outbox.Publish(ctx, EventFriendRequestAccepted, FriendRequestAcceptedPayload{
		ActorID:   actorID,
		TargetID:  peerID.UUID(),
		ChannelID: channelID.UUID(),
	})
	return err
}
