package relation

import (
	"context"

	"github.com/google/uuid"

	"bonfire-api/internal/channel"
	"bonfire-api/internal/errs"
	"bonfire-api/internal/fields"
	"bonfire-api/internal/user"
)

type Service struct {
	repo        Repository
	userCache   UserCache
	userRepo    UserRepository
	userSvc     UserService
	channelRepo ChannelRepository
	memberRepo  MemberRepository
	outboxRepo  OutboxRepository
	tx          TX
}

func NewService(
	repo Repository,
	userCache UserCache,
	userRepo UserRepository,
	userSvc UserService,
	channelRepo ChannelRepository,
	memberRepo MemberRepository,
	outboxRepo OutboxRepository,
	tx TX,
) *Service {
	return &Service{
		repo:        repo,
		userCache:   userCache,
		userRepo:    userRepo,
		userSvc:     userSvc,
		channelRepo: channelRepo,
		memberRepo:  memberRepo,
		outboxRepo:  outboxRepo,
		tx:          tx,
	}
}

func (s *Service) GetPeers(ctx context.Context, rawUserID uuid.UUID, rawType string) (
	peerChannelMap map[fields.ID]fields.ID,
	peerIDs []fields.ID,
	err error,
) {
	userID, err := fields.ParseRequiredID("user_id", rawUserID)
	if err != nil {
		return nil, nil, err
	}

	relType, err := ParseString(rawType)
	if err != nil {
		return nil, nil, err
	}

	relations, err := s.repo.ListTypeByUserID(ctx, userID, relType, maxPeerTypeLimit)
	if err != nil {
		return nil, nil, err
	}

	if len(relations) == 0 {
		return make(map[fields.ID]fields.ID), []fields.ID{}, nil
	}

	peerIDs = make([]fields.ID, 0, len(relations))
	peerChannelMap = make(map[fields.ID]fields.ID, len(relations))

	for _, rel := range relations {
		if rel == nil {
			continue
		}
		pID := rel.PeerID(userID)
		peerIDs = append(peerIDs, pID)
		peerChannelMap[pID] = rel.ChannelID()
	}

	return peerChannelMap, peerIDs, nil
}

func (s *Service) TransitionPending(ctx context.Context, rawActorID, rawPeerID uuid.UUID) error {
	actorID, _, u1, u2, err := validateIDs(rawActorID, rawPeerID)
	if err != nil {
		return err
	}

	channelID, err := fields.NewID()
	if err != nil {
		return err
	}

	now := fields.Now()
	rel := NewPending(u1, u2, actorID, channelID, now)

	actor, err := s.userSvc.Get(ctx, rawActorID)
	if err != nil {
		return err
	}

	return s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		relLock, err := s.repo.GetForUpdate(txCtx, u1, u2)
		if errs.IsNotFound(err) {
			if rel, err = s.repo.Save(txCtx, rel); err != nil {
				return err
			}

			payload := FriendRequestSentPayload{
				ActorID:   actorID.UUID().String(),
				PeerID:    rel.PeerID(actorID).UUID().String(),
				Actor:     user.ParseSummary(actor),
				CreatedAt: now.String(),
			}

			return s.outboxRepo.Publish(txCtx, EventFriendRequestSent, payload, now)
		}
		if err != nil {
			return err
		}

		if relLock.Type().IsFriends() {
			return ErrAlreadyFriends()
		}

		if relLock.Type().IsPending() {
			if err := validateAccept(actorID, relLock); err == nil {
				return s.acceptPendingRequestTx(txCtx, actorID, relLock, now)
			}
			return ErrAlreadyPending()
		}

		return nil
	})
}

func (s *Service) TransitionFriends(ctx context.Context, rawActorID, rawPeerID uuid.UUID) error {
	actorID, _, u1, u2, err := validateIDs(rawActorID, rawPeerID)
	if err != nil {
		return err
	}

	now := fields.Now()

	return s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		rel, err := s.repo.GetForUpdate(txCtx, u1, u2)
		if err != nil {
			return err
		}

		return s.acceptPendingRequestTx(txCtx, actorID, rel, now)
	})
}

func (s *Service) DeleteByUserID(ctx context.Context, rawActorID, rawPeerID uuid.UUID) error {
	actorID, _, u1, u2, err := validateIDs(rawActorID, rawPeerID)
	if err != nil {
		return err
	}

	now := fields.Now()
	var rel *Relation

	err = s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		var dbErr error
		rel, dbErr = s.repo.GetForUpdate(txCtx, u1, u2)
		if err != nil {
			return dbErr
		}

		if dbErr = validateBlockedActor(actorID, rel); err != nil {
			return dbErr
		}

		if dbErr = s.repo.DeleteByUserID(txCtx, u1, u2, actorID); err != nil {
			return dbErr
		}

		if rel.IsFriends() {
			payload := FriendDeletedPayload{
				ActorID: actorID.String(),
				PeerID:  rel.PeerID(actorID).String(),
			}
			return s.outboxRepo.Publish(txCtx, EventRelationDeleted, payload, now)
		}

		return nil
	})
	if err != nil {
		return err
	}

	if rel.IsFriends() {
		_ = s.userCache.RemoveFriendPair(ctx, u1, u2)
	}

	return nil
}

func (s *Service) TransitionBlocked(ctx context.Context, rawActorID, rawPeerID uuid.UUID) error {
	actorID, _, u1, u2, err := validateIDs(rawActorID, rawPeerID)
	if err != nil {
		return err
	}

	now := fields.Now()
	var wasFriends bool

	err = s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		relLock, getErr := s.repo.GetForUpdate(txCtx, u1, u2)
		if getErr != nil && !errs.IsNotFound(getErr) {
			return getErr
		}

		if err := validateBlockedActor(actorID, relLock); err != nil {
			return err
		}

		if relLock != nil && relLock.Type().IsFriends() {
			wasFriends = true
		}

		if errs.IsNotFound(getErr) {
			relLock = NewBlocked(u1, u2, actorID, now)
		} else {
			if relLock.Type().IsBlocked() {
				return nil
			}
			relLock.Block(actorID, now)
		}

		_, err := s.repo.Save(txCtx, relLock)
		if err != nil {
			return err
		}

		if wasFriends {
			payload := FriendDeletedPayload{
				ActorID: actorID.String(),
				PeerID:  relLock.PeerID(actorID).String(),
			}
			return s.outboxRepo.Publish(txCtx, EventRelationDeleted, payload, now)
		}

		return nil
	})
	if err != nil {
		return err
	}

	if wasFriends {
		_ = s.userCache.RemoveFriendPair(ctx, u1, u2)
	}

	return nil
}

func (s *Service) acceptPendingRequestTx(txCtx context.Context, actorID fields.ID, rel *Relation, now fields.Timestamp) error {
	if err := validateBlockedActor(actorID, rel); err != nil {
		return err
	}

	if err := validateAccept(actorID, rel); err != nil {
		return err
	}

	ch := channel.ReconstituteChannel(
		rel.ChannelID(),
		channel.NewChannelTypeDirect(),
		channel.ChannelName{},
		fields.URL{},
		fields.ID{},
		fields.Timestamp{},
		now,
		now,
	)

	newCh, err := s.channelRepo.Create(txCtx, ch)
	if err != nil {
		return err
	}

	members := channel.NewMembers(newCh.ID(), actorID, rel.PeerIDs(actorID), now)
	if _, err := s.memberRepo.CreateBatch(txCtx, members); err != nil {
		return err
	}

	rel.Accept(actorID, newCh.ID(), now)

	if _, err := s.repo.Save(txCtx, rel); err != nil {
		return err
	}

	// return s.outboxRepo.Publish(txCtx, EventFriendRequestAccepted, FriendRequestAcceptedPayload{})
	return nil
}
