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
	repo              Repository
	channelRepo       ChannelRepository
	cachedChannelRepo CachedChannelRepository
	memberRepo        MemberRepository
	cachedMemberRepo  CachedMemberRepository
	outboxRepo        OutboxRepository
	userCache         UserCache
	userRepo          UserRepository
	cachedUserRepo    CachedUserRepository
	tx                TX
}

func NewService(
	repo Repository,
	channelRepo ChannelRepository,
	cachedChannelRepo CachedChannelRepository,
	memberRepo MemberRepository,
	cachedMemberRepo CachedMemberRepository,
	outboxRepo OutboxRepository,
	userCache UserCache,
	userRepo UserRepository,
	cachedUserRepo CachedUserRepository,
	tx TX,
) *Service {
	return &Service{
		repo:              repo,
		channelRepo:       channelRepo,
		cachedChannelRepo: cachedChannelRepo,
		memberRepo:        memberRepo,
		cachedMemberRepo:  cachedMemberRepo,
		outboxRepo:        outboxRepo,
		userCache:         userCache,
		userRepo:          userRepo,
		cachedUserRepo:    cachedUserRepo,
		tx:                tx,
	}
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

			payload := EventFriendRequestSentPayload{
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

		// TODO
		if relLock.Type().IsPending() {
			if err := validateAccept(actorID, relLock); err == nil {
				return s.acceptPendingRequestTx(txCtx, actorID, relLock, now)
			}
			return ErrAlreadyPending()
		}

		return nil
	})
}

// TODO
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
			payload := EventFriendDeletedPayload{
				ActorID: actorID.String(),
				PeerID:  rel.PeerID(actorID).String(),
			}
			return s.outboxRepo.Publish(txCtx, EventFriendDeleted, payload, now)
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
			payload := EventFriendDeletedPayload{
				ActorID: actorID.String(),
				PeerID:  relLock.PeerID(actorID).String(),
			}
			return s.outboxRepo.Publish(txCtx, EventFriendDeleted, payload, now)
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

// TODO
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
