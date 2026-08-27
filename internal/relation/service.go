package relation

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"

	"bonfire-api/internal/channel"
	"bonfire-api/internal/errs"
	"bonfire-api/internal/fields"
	"bonfire-api/internal/user"
)

type Service struct {
	repo        Repository
	userRepo    UserRepository
	userCache   UserCache
	channelRepo ChannelRepository
	memberRepo  MemberRepository
	outboxRepo  OutboxRepository
	tx          TX
}

func NewService(
	repo Repository,
	userRepo UserRepository,
	userCache UserCache,
	channelRepo ChannelRepository,
	memberRepo MemberRepository,
	outboxRepo OutboxRepository,
	tx TX,
) *Service {
	return &Service{
		repo:        repo,
		userRepo:    userRepo,
		userCache:   userCache,
		channelRepo: channelRepo,
		memberRepo:  memberRepo,
		outboxRepo:  outboxRepo,
		tx:          tx,
	}
}

func (s *Service) GetPeer(ctx context.Context, rawActorID, rawPeerID uuid.UUID) (Peer, error) {
	_, peerID, u1, u2, err := validateIDs(rawActorID, rawPeerID)
	if err != nil {
		return Peer{}, err
	}

	rel, err := s.repo.Get(ctx, u1, u2)
	if err != nil {
		return Peer{}, err
	}

	var (
		peerUser     *user.User
		peerPresence user.Presence
	)

	g, gCtx := errgroup.WithContext(ctx)

	g.Go(func() error {
		u, err := s.userRepo.Get(gCtx, peerID)
		if err != nil {
			return err
		}
		peerUser = u
		return nil
	})

	g.Go(func() error {
		p, err := s.userCache.GetPresence(gCtx, peerID)
		if err != nil {
			slog.WarnContext(gCtx, "failed to fetch presence", "peer_id", peerID.String(), "error", err)
			peerPresence = user.NewPresenceOffline()
			return nil
		}
		peerPresence = p
		return nil
	})

	if err := g.Wait(); err != nil {
		return Peer{}, err
	}

	peer, _ := hydratePeer(peerID, rel, peerUser, peerPresence)
	return peer, nil
}

func (s *Service) GetPeers(ctx context.Context, rawUserID uuid.UUID, rawType string) ([]Peer, error) {
	userID, err := fields.ParseRequiredID("user_id", rawUserID)
	if err != nil {
		return nil, err
	}

	relType, err := ParseString(rawType)
	if err != nil {
		return nil, err
	}

	relations, err := s.repo.ListTypeByUserID(ctx, userID, relType, maxPeerTypeLimit)
	if err != nil {
		return nil, err
	}

	if len(relations) == 0 {
		return []Peer{}, nil
	}

	peerIDs := make([]fields.ID, len(relations))
	for i, rel := range relations {
		peerIDs[i] = rel.PeerID(userID)
	}

	var (
		usersMap    map[fields.ID]*user.User
		presenceMap map[fields.ID]user.Presence
	)

	g, gCtx := errgroup.WithContext(ctx)

	g.Go(func() error {
		var err error
		usersMap, err = s.userRepo.GetBatch(gCtx, peerIDs)
		return err
	})

	g.Go(func() error {
		var err error
		presenceMap, err = s.userCache.GetBatchPresence(gCtx, peerIDs)
		if err != nil {
			slog.WarnContext(gCtx, "presence batch fetch failed, defaulting peers to offline",
				"user_id", userID.String(),
				"count", len(peerIDs),
				"error", err,
			)
			presenceMap = nil
		}
		return nil
	})

	if err := g.Wait(); err != nil {
		return nil, err
	}

	return hydratePeers(userID, relations, usersMap, presenceMap), nil
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

	return s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		relLock, err := s.repo.GetForUpdate(txCtx, u1, u2)
		if errs.IsNotFound(err) {
			if _, err := s.repo.Save(txCtx, rel); err != nil {
				return err
			}

			return s.outboxRepo.Publish(txCtx, EventFriendRequestSent, FriendRequestSentPayload{})
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

// TransitionFriends explicitly accepts a pending incoming friend request.
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

// TransitionBlocked places a block on a user, overriding any existing friend or pending state.
func (s *Service) TransitionBlocked(ctx context.Context, rawActorID, rawPeerID uuid.UUID) error {
	actorID, _, u1, u2, err := validateIDs(rawActorID, rawPeerID)
	if err != nil {
		return err
	}

	now := fields.Now()

	return s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		relLock, getErr := s.repo.GetForUpdate(txCtx, u1, u2)
		if getErr != nil && !errs.IsNotFound(getErr) {
			return getErr
		}

		if err := validateBlockedActor(actorID, relLock); err != nil {
			return err
		}

		if errs.IsNotFound(getErr) {
			relLock = NewBlocked(u1, u2, actorID, now)
		} else {
			if relLock.Type().IsBlocked() {
				return nil
			}
			relLock.Block(actorID, now)
		}

		if _, err := s.repo.Save(txCtx, relLock); err != nil {
			return err
		}

		return s.outboxRepo.Publish(txCtx, EventUserBlocked, UserBlockedPayload{})
	})
}

// DeleteByUserID verifies permissions before removing a friendship or friend request.
func (s *Service) DeleteByUserID(ctx context.Context, rawActorID, rawPeerID uuid.UUID) error {
	actorID, _, u1, u2, err := validateIDs(rawActorID, rawPeerID)
	if err != nil {
		return err
	}

	return s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		rel, err := s.repo.GetForUpdate(txCtx, u1, u2)
		if err != nil {
			return err
		}

		if err := validateBlockedActor(actorID, rel); err != nil {
			return err
		}

		if err := s.repo.DeleteByUserID(txCtx, u1, u2, actorID); err != nil {
			return err
		}

		return s.outboxRepo.Publish(txCtx, EventRelationRemoved, RelationRemovedPayload{})
	})
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

	return s.outboxRepo.Publish(txCtx, EventFriendRequestAccepted, FriendRequestAcceptedPayload{})
}
