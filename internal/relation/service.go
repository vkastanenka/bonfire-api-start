package relation

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"bonfire-api/internal/appctx"
	"bonfire-api/internal/channel"
	"bonfire-api/internal/outbox"
	"bonfire-api/internal/pkg/errs"
	"bonfire-api/internal/pkg/helpers"
	"bonfire-api/internal/presence"
	"bonfire-api/internal/user"
)

type Service struct {
	cache             Cache
	repo              Repository
	cachedRepo        CachedRepository
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
	cache Cache,
	repo Repository,
	cachedRepo CachedRepository,
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
		cache:             cache,
		repo:              repo,
		cachedRepo:        cachedRepo,
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

func (s *Service) TransitionPending(ctx context.Context, peerID uuid.UUID) error {
	claims, err := appctx.GetClaims(ctx)
	if err != nil {
		return err
	}

	actorID := claims.UserID
	if actorID == peerID {
		return ErrCannotRequestSelf()
	}

	if s.checkBlockedCache(ctx, actorID, peerID) {
		return nil
	}

	outgoingPendings, err := s.cachedRepo.GetOutgoingPendingIDs(ctx, actorID)
	if err == nil {
		if helpers.ContainsID(outgoingPendings, peerID) {
			return nil
		}
		if len(outgoingPendings) >= MaxPeerTypeLimit {
			return ErrMaxPendingRequestsReached()
		}
	}

	if incomingPendings, err := s.cache.GetIncomingPendingIDs(ctx, actorID); err == nil {
		if helpers.ContainsID(incomingPendings, peerID) {
			return nil
		}
	} else {
		slog.WarnContext(ctx, "failed to get incoming pendings from cache",
			slog.String("user_id", actorID.String()),
			slog.String("peer_id", peerID.String()),
			slog.Any("error", err),
		)
	}

	if peerIncomingPendings, err := s.cachedRepo.GetIncomingPendingIDs(ctx, peerID); err == nil && len(peerIncomingPendings) >= MaxPeerTypeLimit {
		return nil
	}

	friends, err := s.cachedRepo.GetFriendIDs(ctx, actorID)
	if err == nil {
		if helpers.ContainsID(friends, peerID) {
			return nil
		}
		if len(friends) >= MaxPeerTypeLimit {
			return ErrMaxFriendsReached()
		}
	}

	actor, err := s.cachedUserRepo.Get(ctx, actorID)
	if err != nil {
		return err
	}

	u1, u2 := sortIDPair(actorID, peerID)
	var newRel *Relation

	err = s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		existing, err := s.repo.GetForUpdate(txCtx, u1, u2)
		if err != nil && !errs.IsNotFound(err) {
			return err
		}

		if existing != nil {
			return nil
		}

		now := time.Now()
		newRel = NewPending(u1, u2, actorID, now)

		if _, err = s.repo.Save(txCtx, newRel); err != nil {
			return err
		}

		payload := EventFriendRequestSentPayload{
			ActorID:   actorID,
			PeerID:    peerID,
			Actor:     user.ParseSummary(actor),
			CreatedAt: now,
		}

		event, err := outbox.New(
			actorID,
			claims.SessionID,
			appctx.GetTraceID(ctx),
			[]uuid.UUID{peerID},
			EventFriendRequestSent,
			payload,
			now,
		)
		if err != nil {
			return err
		}

		return s.outboxRepo.Create(txCtx, event)
	})
	if err != nil {
		return err
	}

	s.invalidatePairCache(ctx, actorID, peerID, newRel)

	return nil
}

type TransitionFriendsResult struct {
	Channel      *channel.Channel
	ActorMember  *channel.Member
	PeerUser     *user.User
	PeerPresence presence.Presence
}

func (s *Service) TransitionFriends(ctx context.Context, peerID uuid.UUID) (*TransitionFriendsResult, error) {
	claims, err := appctx.GetClaims(ctx)
	if err != nil {
		return nil, err
	}

	actorID := claims.UserID
	if actorID == peerID {
		return nil, ErrCannotRequestSelf()
	}

	if s.checkBlockedCache(ctx, actorID, peerID) {
		return nil, ErrPendingRequestNotFound()
	}

	if incomingPendings, err := s.cache.GetIncomingPendingIDs(ctx, actorID); err == nil {
		if !helpers.ContainsID(incomingPendings, peerID) {
			return nil, ErrPendingRequestNotFound()
		}
	} else {
		slog.WarnContext(ctx, "failed to get incoming pendings from cache",
			slog.String("user_id", actorID.String()),
			slog.String("peer_id", peerID.String()),
			slog.Any("error", err),
		)
	}

	actorFriends, err := s.cachedRepo.GetFriendIDs(ctx, actorID)
	if err == nil {
		if helpers.ContainsID(actorFriends, peerID) {
			return nil, ErrAlreadyFriends()
		}
		if len(actorFriends) >= MaxPeerTypeLimit {
			return nil, ErrMaxFriendsReached()
		}
	}

	peerFriends, err := s.cachedRepo.GetFriendIDs(ctx, peerID)
	if err == nil && len(peerFriends) >= MaxPeerTypeLimit {
		return nil, ErrPendingRequestNotFound()
	}

	actorUser, err := s.cachedUserRepo.Get(ctx, actorID)
	if err != nil {
		return nil, err
	}

	peerUser, err := s.cachedUserRepo.Get(ctx, peerID)
	if err != nil {
		return nil, err
	}

	presences, _ := s.presenceCache.GetBatch(ctx, []uuid.UUID{actorID, peerID})
	actorPresence := presences[actorID]
	peerPresence := presences[peerID]

	var createdChannel *channel.Channel
	var actorMember *channel.Member
	var updatedRel *Relation

	u1, u2 := sortIDPair(actorID, peerID)

	err = s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		rel, err := s.repo.GetForUpdate(txCtx, u1, u2)
		if err != nil {
			return err
		}

		if err := validateNonBlockedActor(actorID, rel); err != nil {
			return ErrPendingRequestNotFound()
		}

		if err := validateAccept(actorID, rel); err != nil {
			return err
		}

		now := time.Now()

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
		updatedRel = rel

		actorPayload := EventFriendAddedPayload{
			Friend:         user.ParseView(peerUser),
			FriendPresence: peerPresence,
			Channel:        newCh,
			Member:         actorMember,
			CreatedAt:      now,
		}

		actorEvent, err := outbox.New(
			actorID,
			claims.SessionID,
			appctx.GetTraceID(ctx),
			[]uuid.UUID{actorID, peerID},
			EventFriendAdded,
			actorPayload,
			now,
		)
		if err != nil {
			return err
		}
		if err := s.outboxRepo.Create(txCtx, actorEvent); err != nil {
			return err
		}

		peerPayload := EventFriendAddedPayload{
			Friend:         user.ParseView(actorUser),
			FriendPresence: actorPresence,
			Channel:        newCh,
			Member:         peerMember,
			CreatedAt:      now,
		}

		peerEvent, err := outbox.New(
			actorID,
			claims.SessionID,
			appctx.GetTraceID(ctx),
			[]uuid.UUID{actorID, peerID},
			EventFriendAdded,
			peerPayload,
			now,
		)
		if err != nil {
			return err
		}

		return s.outboxRepo.Create(txCtx, peerEvent)
	})
	if err != nil {
		return nil, err
	}

	s.invalidatePairCache(ctx, actorID, peerID, &Relation{Type: TypePending})
	s.invalidatePairCache(ctx, actorID, peerID, updatedRel)

	return &TransitionFriendsResult{
		Channel:      createdChannel,
		ActorMember:  actorMember,
		PeerUser:     peerUser,
		PeerPresence: peerPresence,
	}, nil
}

func (s *Service) TransitionBlocked(ctx context.Context, peerID uuid.UUID) error {
	claims, err := appctx.GetClaims(ctx)
	if err != nil {
		return err
	}

	actorID := claims.UserID
	if actorID == peerID {
		return ErrCannotRequestSelf()
	}

	if s.checkBlockedCache(ctx, actorID, peerID) {
		return nil
	}

	u1, u2 := sortIDPair(actorID, peerID)
	var prevRel *Relation
	now := time.Now()

	err = s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		relLock, getErr := s.repo.GetForUpdate(txCtx, u1, u2)
		if getErr != nil && !errs.IsNotFound(getErr) {
			return getErr
		}

		if err := validateNonBlockedActor(actorID, relLock); err != nil {
			return nil
		}

		if errs.IsNotFound(getErr) {
			relLock = NewBlocked(u1, u2, actorID, now)
		} else {
			if relLock.Type.IsBlocked() {
				return nil
			}
			prevRel = relLock
			relLock.Block(actorID, now)
		}

		if _, err := s.repo.Save(txCtx, relLock); err != nil {
			return err
		}

		if prevRel != nil {
			if prevRel.IsFriends() {
				payload := EventFriendDeletedPayload{
					ActorID: actorID,
					PeerID:  peerID,
				}

				event, err := outbox.New(
					actorID,
					claims.SessionID,
					appctx.GetTraceID(ctx),
					[]uuid.UUID{actorID, peerID},
					EventFriendDeleted,
					payload,
					now,
				)
				if err != nil {
					return err
				}
				if err := s.outboxRepo.Create(txCtx, event); err != nil {
					return err
				}
			} else if prevRel.IsPending() {
				payload := EventFriendRequestDeletedPayload{
					ActorID: actorID,
					PeerID:  peerID,
				}

				event, err := outbox.New(
					actorID,
					claims.SessionID,
					appctx.GetTraceID(ctx),
					[]uuid.UUID{peerID},
					EventFriendRequestDeleted,
					payload,
					now,
				)
				if err != nil {
					return err
				}
				if err := s.outboxRepo.Create(txCtx, event); err != nil {
					return err
				}
			}
		}

		return nil
	})
	if err != nil {
		return err
	}

	s.invalidatePairCache(ctx, actorID, peerID, &Relation{Type: TypeBlocked})
	s.invalidatePairCache(ctx, actorID, peerID, prevRel)

	return nil
}

func (s *Service) DeleteByUserID(ctx context.Context, peerID uuid.UUID) error {
	claims, err := appctx.GetClaims(ctx)
	if err != nil {
		return err
	}

	actorID := claims.UserID
	if actorID == peerID {
		return ErrCannotRequestSelf()
	}

	if s.checkBlockedCache(ctx, actorID, peerID) {
		return nil
	}

	u1, u2 := sortIDPair(actorID, peerID)
	var deletedRel *Relation

	err = s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		rel, err := s.repo.GetForUpdate(txCtx, u1, u2)
		if err != nil {
			if errs.IsNotFound(err) {
				return nil
			}
			return err
		}

		if err := validateNonBlockedActor(actorID, rel); err != nil {
			return nil
		}

		if err := s.repo.DeleteByUserID(txCtx, u1, u2, actorID); err != nil {
			return err
		}

		deletedRel = rel
		now := time.Now()

		if rel.IsFriends() {
			payload := EventFriendDeletedPayload{
				ActorID: actorID,
				PeerID:  rel.PeerID(actorID),
			}

			event, err := outbox.New(
				actorID,
				claims.SessionID,
				appctx.GetTraceID(ctx),
				[]uuid.UUID{actorID, peerID},
				EventFriendDeleted,
				payload,
				now,
			)
			if err != nil {
				return err
			}
			if err := s.outboxRepo.Create(txCtx, event); err != nil {
				return err
			}
		} else if rel.IsPending() {
			payload := EventFriendRequestDeletedPayload{
				ActorID: actorID,
				PeerID:  peerID,
			}

			event, err := outbox.New(
				actorID,
				claims.SessionID,
				appctx.GetTraceID(ctx),
				[]uuid.UUID{peerID},
				EventFriendRequestDeleted,
				payload,
				now,
			)
			if err != nil {
				return err
			}
			if err := s.outboxRepo.Create(txCtx, event); err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return err
	}

	s.invalidatePairCache(ctx, actorID, peerID, deletedRel)

	return nil
}

func (s *Service) checkBlockedCache(ctx context.Context, actorID, peerID uuid.UUID) bool {
	if incomingBlocks, err := s.cache.GetIncomingBlockIDs(ctx, actorID); err == nil {
		if helpers.ContainsID(incomingBlocks, peerID) {
			return true
		}
	} else {
		slog.WarnContext(ctx, "failed to get incoming blocks from cache",
			slog.String("user_id", actorID.String()),
			slog.String("peer_id", peerID.String()),
			slog.Any("error", err),
		)
	}

	if outgoingBlocks, err := s.cache.GetOutgoingBlockIDs(ctx, actorID); err == nil {
		if helpers.ContainsID(outgoingBlocks, peerID) {
			return true
		}
	} else {
		slog.WarnContext(ctx, "failed to get outgoing blocks from cache",
			slog.String("user_id", actorID.String()),
			slog.String("peer_id", peerID.String()),
			slog.Any("error", err),
		)
	}

	return false
}

func (s *Service) invalidatePairCache(ctx context.Context, actorID, peerID uuid.UUID, rel *Relation) {
	if rel == nil {
		return
	}

	cacheCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
	defer cancel()

	var err error
	var cacheType string

	switch {
	case rel.IsFriends():
		cacheType = "friends"
		err = s.cache.DeleteFriendsPair(cacheCtx, actorID, peerID)
	case rel.IsPending():
		cacheType = "pending"
		err = s.cache.DeletePendingPair(cacheCtx, actorID, peerID)
	case rel.IsBlocked():
		cacheType = "blocks"
		err = s.cache.DeleteBlocksPair(cacheCtx, actorID, peerID)
	}

	if err != nil {
		slog.WarnContext(cacheCtx, "failed to invalidate cache pair",
			slog.String("cache_type", cacheType),
			slog.String("user_id", actorID.String()),
			slog.String("peer_id", peerID.String()),
			slog.Any("error", err),
		)
	}
}
