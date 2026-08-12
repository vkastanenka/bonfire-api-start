package relation

import (
	"bonfire-api/internal/channel"
	"bonfire-api/internal/errs"
	"bonfire-api/internal/fields"
	"bonfire-api/internal/outbox"
	"bonfire-api/internal/presence"
	"bonfire-api/internal/user"
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"
)

const maxPeerLimit int32 = 1000

type Cache interface {
	Get(ctx context.Context, u1, u2 fields.ID) (*Relation, error)
	GetUserRelations(ctx context.Context, userID fields.ID) (map[uuid.UUID]Type, error)
	TransitionRelation(ctx context.Context, u1, u2 fields.ID, rel *Relation) error
	RemoveRelation(ctx context.Context, u1, u2 fields.ID) error
	SetUserRelations(ctx context.Context, userID fields.ID, relations map[uuid.UUID]Type) error
	InvalidateUser(ctx context.Context, userID fields.ID) error
}

type PresenceCache interface {
	Get(ctx context.Context, userID uuid.UUID) (presence.Presence, error)
	GetBatch(ctx context.Context, userIDs []uuid.UUID) (map[uuid.UUID]presence.Presence, error)
	Set(ctx context.Context, userID uuid.UUID, p presence.Presence) error
}

type UserCache interface {
	Delete(ctx context.Context, userID fields.ID) error
	DeleteBatch(ctx context.Context, userIDs []fields.ID) error
	Get(ctx context.Context, userID fields.ID) (*user.User, error)
	GetBatch(ctx context.Context, userIDs []fields.ID) (map[fields.ID]*user.User, []fields.ID, error)
	Set(ctx context.Context, usr *user.User) error
	SetBatch(ctx context.Context, users []*user.User) error
}

type Repository interface {
	DeleteByUser(ctx context.Context, user1ID fields.ID, user2ID fields.ID, actorID fields.ID) error
	Get(ctx context.Context, user1ID fields.ID, user2ID fields.ID) (*Relation, error)
	GetByChannel(ctx context.Context, channelID fields.ID) (*Relation, error)
	GetForUpdate(ctx context.Context, user1ID fields.ID, user2ID fields.ID) (*Relation, error)
	ListTypeByUser(ctx context.Context, userID fields.ID, relType Type, limit int32) ([]*Relation, error)
	Save(ctx context.Context, rel *Relation) (*Relation, error)
}

type ChannelRepository interface {
	Create(ctx context.Context, ch *channel.Channel) (*channel.Channel, error)
	MemberAddBatch(ctx context.Context, members []*channel.Member) error
}

type OutboxRepository interface {
	Publish(ctx context.Context, variant string, payload any) (*outbox.Event, error)
}

type UserRepository interface {
	Availability(ctx context.Context, email *user.Email, username *user.Username) (bool, bool, error)
	Create(ctx context.Context, u *user.User) (*user.User, error)
	Get(ctx context.Context, id fields.ID) (*user.User, error)
	GetByEmail(ctx context.Context, email user.Email) (*user.User, error)
	GetCached(ctx context.Context, id fields.ID) (*user.User, error)
	GetDeleteScheduledBatch(ctx context.Context, currentTime fields.Timestamp, batchLimit int32) ([]*user.User, error)
	Update(ctx context.Context, u *user.User) (*user.User, error)
	UpdateBatch(ctx context.Context, usersJson []byte) ([]*user.User, error)
}

type Tx interface {
	ExecTx(ctx context.Context, fn func(txCtx context.Context) error) error
}

type Service struct {
	cache     Cache
	presence  PresenceCache
	userCache UserCache
	repo      Repository
	channel   ChannelRepository
	outbox    OutboxRepository
	user      UserRepository
	tx        Tx
}

func NewService(
	cache Cache,
	presence PresenceCache,
	userCache UserCache,
	repo Repository,
	channel ChannelRepository,
	outbox OutboxRepository,
	user UserRepository,
	tx Tx,
) *Service {
	return &Service{
		cache:     cache,
		presence:  presence,
		userCache: userCache,
		repo:      repo,
		channel:   channel,
		outbox:    outbox,
		user:      user,
		tx:        tx,
	}
}

// // AcceptFriendRequest explicitly accepts a pending incoming friend request.
// func (s *Service) AcceptFriendRequest(ctx context.Context, rawActorID, rawPeerID uuid.UUID) error {
// 	actorID, err := NewUserID(rawActorID)
// 	if err != nil {
// 		return errs.InvalidArgument("invalid actor id")
// 	}

// 	peerID, err := NewUserID(rawPeerID)
// 	if err != nil {
// 		return errs.InvalidArgument("invalid peer id")
// 	}

// 	if actorID == peerID {
// 		return errs.InvalidArgument("cannot accept friend request from yourself")
// 	}

// 	u1, u2 := sortUserIDs(actorID, peerID)

// 	return s.tx.ExecTx(ctx, func(txCtx context.Context) error {
// 		rel, err := s.repo.GetForUpdate(txCtx, u1.UUID(), u2.UUID())
// 		if err != nil {
// 			if errs.IsNotFound(err) {
// 				return errs.NotFound("no pending request to accept").Wrap(err)
// 			}
// 			return err
// 		}

// 		return s.acceptPendingRequestTx(txCtx, rel, actorID.UUID())
// 	})
// }

// // Block places a block on a user, overriding any existing friend or pending state.
// func (s *Service) Block(ctx context.Context, rawActorID, rawPeerID uuid.UUID) error {
// 	actorID, err := NewUserID(rawActorID)
// 	if err != nil {
// 		return errs.InvalidArgument("invalid actor id")
// 	}

// 	peerID, err := NewUserID(rawPeerID)
// 	if err != nil {
// 		return errs.InvalidArgument("invalid peer id")
// 	}

// 	if actorID == peerID {
// 		return errs.InvalidArgument("cannot block yourself")
// 	}

// 	u1, u2 := sortUserIDs(actorID, peerID)

// 	return s.tx.ExecTx(ctx, func(txCtx context.Context) error {
// 		var rel *Relation

// 		fetchedRel, err := s.repo.GetForUpdate(txCtx, u1.UUID(), u2.UUID())
// 		if err != nil && !errs.IsNotFound(err) {
// 			return err
// 		}

// 		if errs.IsNotFound(err) {
// 			now := time.Now().UTC()
// 			rel, err = Reconstitute(
// 				u1.UUID(),
// 				u2.UUID(),
// 				actorID.UUID(),
// 				nil, // rawChannelID (*uuid.UUID) - nil since VariantBlocked isn't VariantFriends
// 				uint8(VariantBlocked),
// 				now,
// 				now,
// 			)
// 			if err != nil {
// 				return errs.InvalidArgument(err.Error()).Wrap(err)
// 			}
// 		} else {
// 			rel = fetchedRel
// 			// If already blocked by the OTHER party, do nothing (don't overwrite their block ownership)
// 			if rel.IsBlocked() && rel.ActorID() != actorID {
// 				return nil
// 			}

// 			if err := rel.Block(actorID); err != nil {
// 				return errs.InvalidArgument(err.Error()).Wrap(err)
// 			}
// 		}

// 		if err := s.repo.Upsert(txCtx, rel); err != nil {
// 			return err
// 		}

// 		// Emit outbox event for blocking
// 		_, err = s.outbox.Publish(txCtx, EventUserBlocked, UserBlockedPayload{
// 			ActorID:  actorID.UUID(),
// 			TargetID: peerID.UUID(),
// 		})
// 		return err
// 	})
// }

// // DeleteVerified verifies permissions before removing a friendship or request.
// func (s *Service) DeleteVerified(ctx context.Context, rawActorID, rawPeerID uuid.UUID) error {
// 	actorID, err := NewUserID(rawActorID)
// 	if err != nil {
// 		return errs.InvalidArgument("invalid actor id")
// 	}

// 	peerID, err := NewUserID(rawPeerID)
// 	if err != nil {
// 		return errs.InvalidArgument("invalid peer id")
// 	}

// 	if actorID == peerID {
// 		return errs.InvalidArgument("cannot target yourself").Wrap(ErrSelfRelation)
// 	}

// 	u1, u2 := sortUserIDs(actorID, peerID)

// 	return s.tx.ExecTx(ctx, func(txCtx context.Context) error {
// 		err := s.repo.DeleteVerified(txCtx, u1.UUID(), u2.UUID(), actorID.UUID())
// 		if err != nil {
// 			if errs.IsNotFound(err) {
// 				return errs.NotFound("relationship not found").Wrap(err)
// 			}
// 			if errors.Is(err, ErrRelationBlocked) {
// 				return errs.PermissionDenied("cannot modify blocked relationship").Wrap(err)
// 			}
// 			return err
// 		}

// 		// Emit outbox event for removal (unfriend / cancel request)
// 		_, err = s.outbox.Publish(txCtx, EventRelationRemoved, RelationRemovedPayload{
// 			ActorID:  actorID.UUID(),
// 			TargetID: peerID.UUID(),
// 		})
// 		return err
// 	})
// }

func (s *Service) GetPeer(ctx context.Context, rawUserID, rawPeerID uuid.UUID) (*Peer, error) {
	userID, err := fields.ParseRequiredID("user_id", rawUserID)
	if err != nil {
		return nil, err
	}

	peerID, err := fields.ParseRequiredID("peer_id", rawPeerID)
	if err != nil {
		return nil, err
	}

	if userID.Equals(peerID) {
		return nil, errs.InvalidArgument("Relation ids cannot match.").
			FieldViolation("peer_id", "ID is the same as user ID", "PEER_ID_INVALID")
	}

	u1, u2 := SortUserIDs(userID, peerID)

	rel, err := s.cache.Get(ctx, u1, u2)
	if err != nil {
		slog.WarnContext(ctx, "failed to read relation cache", "error", err)
	}

	if rel == nil {
		rel, err = s.repo.Get(ctx, u1, u2)
		if err != nil {
			return nil, err
		}

		if cacheErr := s.cache.TransitionRelation(ctx, u1, u2, rel); cacheErr != nil {
			slog.WarnContext(ctx, "failed to backfill relation cache", "error", cacheErr)
		}
	}

	var (
		peerUser     *user.User
		peerPresence presence.Presence
	)

	g, gCtx := errgroup.WithContext(ctx)

	g.Go(func() error {
		u, uErr := s.user.GetCached(gCtx, peerID)
		if uErr != nil {
			return uErr
		}
		peerUser = u
		return nil
	})

	g.Go(func() error {
		p, pErr := s.presence.Get(gCtx, peerID.UUID())
		if pErr != nil {
			slog.WarnContext(gCtx, "failed to fetch presence", "peer_id", peerID.String(), "error", pErr)
			peerPresence = presence.PresenceOffline
			return nil
		}
		peerPresence = p
		return nil
	})

	if err := g.Wait(); err != nil {
		return nil, err
	}

	return NewPeer(
		peerID,
		rel.ChannelID(),
		rel.ActorID(),
		peerUser.AvatarURL(),
		peerUser.Username(),
		peerUser.DisplayName(),
		rel.Type(),
		peerPresence,
	), nil
}

func (s *Service) GetPeers(ctx context.Context, rawUserID uuid.UUID, rawType string) (*[]Peer, error) {
	userID, err := fields.ParseRequiredID("user_id", rawUserID)
	if err != nil {
		return nil, err
	}

	// Parse type
	relType, err := Parse(rawType)
	if err != nil {
		return nil, err
	}

	// Get from cache, ensure 1000 limit
	peerIDs, err := s.cache.GetUserRelations(ctx, userID, peerType, maxPeerLimit)
	if err != nil {
		// 3. Fallback to repo if cache miss/failure (enforcing max 1000 limit)
		peerIDs, err = s.relationshipRepo.GetPeerIDs(ctx, userID, peerType, maxPeerLimit)
		if err != nil {
			return nil, err
		}

		// Asynchronously backfill peer IDs cache on miss
		go func(ids []fields.ID) {
			_ = s.peerCache.SetPeerIDs(context.WithoutCancel(ctx), userID, peerType, ids)
		}(peerIDs)
	}

	// Fallback to repo if miss, ensure 1000 limit

	// Order ids by asc display_name

	// Start goroutine

	// Get batch presences, fallback to offline if issues

	// Get batch users from cache

	// Fallback to db batch get for missing users from cache

	// End goroutine

	// Build array of peers

	// return
}

// // func (s *Service) ListPeers(ctx context.Context, userID uuid.UUID, filter *Variant) ([]Perspective, error) {
// // 	if filter != nil && !filter.IsValid() {
// // 		return nil, errs.InvalidArgument("invalid relationship status filter")
// // 	}

// // 	perspectives, err := s.repo.ListPerspectives(ctx, userID, filter)
// // 	if err != nil {
// // 		if errs.IsNotFound(err) {
// // 			return []Perspective{}, nil
// // 		}
// // 		return nil, err
// // 	}

// // 	return perspectives, nil
// // }

func (s *Service) SendFriendRequest(ctx context.Context, rawActorID, rawPeerID uuid.UUID) error {
	actorID, err := fields.ParseRequiredID("actor_id", rawActorID)
	if err != nil {
		return err
	}

	peerID, err := fields.ParseRequiredID("peer_id", rawPeerID)
	if err != nil {
		return err
	}

	if actorID.Equals(peerID) {
		return errs.InvalidArgument("Cannot friend yourself.").
			FieldViolation("peer_id", "ID is the same as actor ID", "PEER_ID_INVALID")
	}

	u1, u2 := SortUserIDs(actorID, peerID)
	now := fields.NewTimestampFromTime(time.Now())
	channelID, _ := fields.ParseID("channel_id", uuid.Nil)
	newRel := New(u1, u2, actorID, channelID, TypePending, now, now)

	var updatedRel *Relation

	err = s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		rel, err := s.repo.GetForUpdate(txCtx, u1, u2)
		if err != nil {
			if errs.IsNotFound(err) {
				relRow, err := s.repo.Save(txCtx, newRel)
				if err != nil {
					return err
				}

				updatedRel = relRow

				_, err = s.outbox.Publish(txCtx, EventFriendRequestSent, FriendRequestSentPayload{
					ActorID:  actorID.UUID(),
					TargetID: peerID.UUID(),
				})
				return err
			}
			return err
		}

		switch rel.Type() {
		case TypeFriends:
			return errs.AlreadyExists("Already friends with this user.")

		case TypeBlocked:
			return errs.PermissionDenied("Cannot interact with this user.")

		case TypePending:
			if !rel.ActorID().Equals(actorID) {
				acceptedRel, acceptErr := s.acceptPendingRequestTx(txCtx, actorID, rel)
				if acceptErr != nil {
					return acceptErr
				}
				updatedRel = acceptedRel
				return nil
			}
			return errs.AlreadyExists("Friend request already pending.")
		}

		return nil
	})

	if err != nil {
		return err
	}

	// Post-commit cache write
	if updatedRel != nil {
		if cacheErr := s.cache.TransitionRelation(ctx, u1, u2, updatedRel); cacheErr != nil {
			slog.WarnContext(ctx, "failed to transition relation cache",
				"user1_id", u1.String(),
				"user2_id", u2.String(),
				"actor_id", actorID.String(),
				"error", cacheErr,
				"scope", "relation",
			)
		}
	}

	return nil
}

// Private helper for transactional acceptance, DM channel creation, and outbox event publishing.
func (s *Service) acceptPendingRequestTx(ctx context.Context, actorID fields.ID, rel *Relation) (*Relation, error) {
	// var channelID fields.ID

	// // 1. Check if a DM channel already exists for this relationship (e.g., re-friending)
	// if existingChID := rel.ChannelID(); existingChID.UUIDPtr() != nil {
	// 	channelID = existingChID
	// } else {
	// 	// 2. Instantiate new 1:1 Direct Message Channel entity (TypeDirect)
	// 	ch, err := channel.New(channel.TypeDirect, nil, nil)
	// 	if err != nil {
	// 		return errs.InvalidArgument("failed to construct DM channel").Wrap(err)
	// 	}

	// 	// 3. Persist Channel record inside current transaction
	// 	createdCh, err := s.channel.Create(ctx, ch)
	// 	if err != nil {
	// 		return err
	// 	}

	// 	// 4. Construct & batch-add members
	// 	chUUID := createdCh.ID().UUID()
	// 	u1ID := rel.User1ID().UUID()
	// 	u2ID := rel.User2ID().UUID()

	// 	m1, err := channel.NewMember(chUUID, u1ID)
	// 	if err != nil {
	// 		return errs.InvalidArgument("invalid member 1").Wrap(err)
	// 	}

	// 	m2, err := channel.NewMember(chUUID, u2ID)
	// 	if err != nil {
	// 		return errs.InvalidArgument("invalid member 2").Wrap(err)
	// 	}

	// 	if err := s.channel.MemberAddBatch(ctx, []*channel.Member{m1, m2}); err != nil {
	// 		return err
	// 	}

	// 	channelID = ChannelID(createdCh.ID())
	// }

	// // 5. Transition relationship state to VariantFriends with the active channel ID
	// if err := rel.Accept(actID, channelID); err != nil {
	// 	if errors.Is(err, ErrCannotAccept) {
	// 		return errs.PermissionDenied("cannot accept your own outgoing friend request").Wrap(err)
	// 	}
	// 	return errs.InvalidArgument(err.Error()).Wrap(err)
	// }

	// // 6. Upsert updated relationship state
	// if err := s.repo.Upsert(ctx, rel); err != nil {
	// 	return err
	// }

	// // 7. Emit outbox event
	// peerID := rel.GetPeerID(actID)
	// _, err = s.outbox.Publish(ctx, EventFriendRequestAccepted, FriendRequestAcceptedPayload{
	// 	ActorID:   actorID,
	// 	TargetID:  peerID.UUID(),
	// 	ChannelID: channelID.UUID(),
	// })
	return nil, nil
}
