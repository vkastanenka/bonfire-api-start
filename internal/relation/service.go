package relation

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"bonfire-api/internal/channel"
	"bonfire-api/internal/errs"
	"bonfire-api/internal/presence"
	"bonfire-api/internal/user"
)

type Service struct {
	repo              Repository
	channelRepo       ChannelRepository
	cachedChannelRepo CachedChannelRepository
	memberRepo        MemberRepository
	cachedMemberRepo  CachedMemberRepository
	outboxRepo        OutboxRepository
	presenceCache     PresenceCache
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
	presenceCache PresenceCache,
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
		presenceCache:     presenceCache,
		userCache:         userCache,
		userRepo:          userRepo,
		cachedUserRepo:    cachedUserRepo,
		tx:                tx,
	}
}

func (s *Service) TransitionPending(ctx context.Context, actorID, peerID uuid.UUID) error {
	u1, u2 := sortIDPair(actorID, peerID)
	now := time.Now()
	rel := NewPending(u1, u2, actorID, now)

	actor, err := s.cachedUserRepo.Get(ctx, actorID)
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
				ActorID:   actorID,
				PeerID:    rel.PeerID(actorID),
				Actor:     user.ParseSummary(actor),
				CreatedAt: now,
			}

			return s.outboxRepo.Publish(txCtx, EventFriendRequestSent, payload, now)
		}
		if err != nil {
			return err
		}

		if relLock.Type.IsFriends() {
			return ErrAlreadyFriends()
		}

		if relLock.Type.IsPending() {
			return ErrAlreadyPending()
		}

		return nil
	})
}

type TransitionFriendsResult struct {
	Channel      *channel.Channel
	ActorMember  *channel.Member
	PeerUser     *user.User
	PeerPresence presence.Presence
}

func (s *Service) TransitionFriends(ctx context.Context, actorID, peerID uuid.UUID) (*TransitionFriendsResult, error) {
	actorUser, err := s.cachedUserRepo.Get(ctx, actorID)
	if err != nil {
		return nil, err
	}

	peerUser, err := s.cachedUserRepo.Get(ctx, peerID)
	if err != nil {
		return nil, err
	}

	actorPresence, _ := s.presenceCache.GetPresence(ctx, actorID)
	peerPresence, _ := s.presenceCache.GetPresence(ctx, peerID)

	var createdChannel *channel.Channel
	var actorMember *channel.Member
	u1, u2 := sortIDPair(actorID, peerID)
	now := time.Now()

	err = s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		rel, err := s.repo.GetForUpdate(txCtx, u1, u2)
		if err != nil {
			return err
		}

		if err := validateNonBlockedActor(actorID, rel); err != nil {
			return err
		}

		if err := validateAccept(actorID, rel); err != nil {
			return err
		}

		ch, err := channel.NewDirectChannel(now)
		if err != nil {
			return err
		}

		newCh, err := s.channelRepo.Create(txCtx, ch)
		if err != nil {
			return err
		}
		createdChannel = newCh

		newMembers := channel.NewMembers(newCh.ID, actorID, rel.PeerIDs(actorID), now)
		createdMembers, err := s.memberRepo.CreateBatch(txCtx, newMembers)
		if err != nil {
			return err
		}

		var peerMember *channel.Member
		for _, m := range createdMembers {
			if m.UserID == actorID {
				actorMember = m
			} else if m.UserID == peerID {
				peerMember = m
			}
		}

		rel.Accept(actorID, newCh.ID, now)

		if _, err := s.repo.Save(txCtx, rel); err != nil {
			return err
		}

		actorPayload := EventFriendAddedPayload{
			Friend:         peerUser,
			FriendPresence: peerPresence,
			Channel:        newCh,
			Member:         actorMember,
			CreatedAt:      now,
		}
		if err := s.outboxRepo.Publish(txCtx, EventFriendAdded, actorPayload, now); err != nil {
			return err
		}

		peerPayload := EventFriendAddedPayload{
			Friend:         actorUser,
			FriendPresence: actorPresence,
			Channel:        newCh,
			Member:         peerMember,
			CreatedAt:      now,
		}
		return s.outboxRepo.Publish(txCtx, EventFriendAdded, peerPayload, now)
	})
	if err != nil {
		return nil, err
	}

	if err := s.userCache.AddFriendPair(ctx, actorID, peerID, createdChannel.ID); err != nil {
		slog.WarnContext(ctx, "failed to update friend pair cache", "actor_id", actorID, "peer_id", peerID, "err", err)
	}

	// TODO: Handle cache

	return &TransitionFriendsResult{
		Channel:      createdChannel,
		ActorMember:  actorMember,
		PeerUser:     peerUser,
		PeerPresence: peerPresence,
	}, nil
}

func (s *Service) DeleteByUserID(ctx context.Context, actorID, peerID uuid.UUID) error {
	var rel *Relation
	u1, u2 := sortIDPair(actorID, peerID)
	now := time.Now()

	err := s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		var dbErr error
		rel, dbErr = s.repo.GetForUpdate(txCtx, u1, u2)
		if dbErr != nil {
			return dbErr
		}

		if dbErr = validateNonBlockedActor(actorID, rel); dbErr != nil {
			return dbErr
		}

		if dbErr = s.repo.DeleteByUserID(txCtx, u1, u2, actorID); dbErr != nil {
			return dbErr
		}

		if rel.IsFriends() {
			payload := EventFriendDeletedPayload{
				ActorID: actorID,
				PeerID:  rel.PeerID(actorID),
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

func (s *Service) TransitionBlocked(ctx context.Context, actorID, peerID uuid.UUID) error {
	var wasFriends bool
	u1, u2 := sortIDPair(actorID, peerID)
	now := time.Now()

	err := s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		relLock, getErr := s.repo.GetForUpdate(txCtx, u1, u2)
		if getErr != nil && !errs.IsNotFound(getErr) {
			return getErr
		}

		if err := validateNonBlockedActor(actorID, relLock); err != nil {
			return err
		}

		if relLock != nil && relLock.Type.IsFriends() {
			wasFriends = true
		}

		if errs.IsNotFound(getErr) {
			relLock = NewBlocked(u1, u2, actorID, now)
		} else {
			if relLock.Type.IsBlocked() {
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
				ActorID: actorID,
				PeerID:  relLock.PeerID(actorID),
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
