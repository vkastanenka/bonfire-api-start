package channel

import (
	"bonfire-api/internal/fields"
	"bonfire-api/internal/pkg/ptr"
	"bonfire-api/internal/presence"
	"bonfire-api/internal/user"
	"context"
	"time"

	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"
)

type ChannelService struct {
	cache         ChannelCache
	repo          ChannelRepository
	memberRepo    MemberRepository
	messageRepo   MessageRepository
	reactionRepo  ReactionRepository
	presenceCache PresenceCache
	userRepo      UserRepository
	userService   UserService
	outboxRepo    OutboxRepository
	relationRepo  RelationRepository
	tx            TX
}

func NewChannelService(
	cache ChannelCache,
	repo ChannelRepository,
	memberRepo MemberRepository,
	messageRepo MessageRepository,
	reactionRepo ReactionRepository,
	presenceCache PresenceCache,
	userRepo UserRepository,
	outboxRepo OutboxRepository,
	relationRepo RelationRepository,
	tx TX,
) *ChannelService {
	return &ChannelService{
		cache:         cache,
		repo:          repo,
		memberRepo:    memberRepo,
		messageRepo:   messageRepo,
		reactionRepo:  reactionRepo,
		presenceCache: presenceCache,
		userRepo:      userRepo,
		outboxRepo:    outboxRepo,
		relationRepo:  relationRepo,
		tx:            tx,
	}
}

type CreateGroupResult struct {
	Channel     *Channel
	ActorMember *Member
	Users       map[fields.ID]*user.User
	Presences   map[fields.ID]presence.Presence
	MemberIDs   []fields.ID
}

// CreateGroup creates a new group channel with members.
func (s *ChannelService) CreateGroup(ctx context.Context, rawActorID, rawSessionID uuid.UUID, rawPeerIDs []uuid.UUID) (*CreateGroupResult, error) {
	if err := validateMaxPeers(rawPeerIDs); err != nil {
		return nil, err
	}

	actorID, err := fields.ParseRequiredID("actor_id", rawActorID)
	if err != nil {
		return nil, err
	}

	sessionID, err := fields.ParseRequiredID("session_id", rawSessionID)
	if err != nil {
		return nil, err
	}

	peerMemberIDs, err := fields.ParseIDs(rawPeerIDs)
	if err != nil {
		return nil, err
	}

	dedupedMemberIDs := fields.DedupeIDs(append(peerMemberIDs, actorID))
	peerIDs := fields.RemoveID(dedupedMemberIDs, actorID)

	if len(peerIDs) > 0 {
		if err = s.relationRepo.HasIncomingBlock(ctx, actorID, peerIDs); err != nil {
			return nil, err
		}
	}

	now := fields.Now()

	ch, err := NewGroupChannel(now)
	if err != nil {
		return nil, err
	}

	membs := NewMembers(ch.ID(), actorID, peerIDs, now)

	g, gCtx := errgroup.WithContext(ctx)

	var (
		users     map[fields.ID]*user.User
		presences map[fields.ID]presence.Presence
	)

	g.Go(func() error {
		var fetchErr error
		users, fetchErr = s.userService.GetBatch(gCtx, dedupedMemberIDs)
		return fetchErr
	})

	g.Go(func() error {
		var fetchErr error
		presences, fetchErr = s.presenceCache.GetBatchPresence(gCtx, dedupedMemberIDs)
		return fetchErr
	})

	if err := g.Wait(); err != nil {
		return nil, err
	}

	sortMemberIDs(dedupedMemberIDs, users)

	txErr := s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		var repoErr error
		if ch, repoErr = s.repo.Create(txCtx, ch); repoErr != nil {
			return repoErr
		}

		if membs, repoErr = s.memberRepo.CreateBatch(txCtx, membs); repoErr != nil {
			return repoErr
		}

		membersMap := make(map[fields.ID]*Member, len(membs))
		for _, m := range membs {
			membersMap[m.UserID()] = m
		}

		payload := EventChannelCreatedPayload{
			ExcludeSessionID: sessionID,
			Channel:          ch,
			Users:            users,
			Presences:        presences,
			MemberIDs:        dedupedMemberIDs,
		}

		return s.outboxRepo.Publish(txCtx, EventChannelCreated, payload, now)
	})
	if txErr != nil {
		return nil, txErr
	}

	_ = s.cache.CreateGroup(ctx, ch, membs)

	return &CreateGroupResult{
		Channel:     ch,
		ActorMember: filterMembership(actorID, membs),
		Users:       users,
		Presences:   presences,
		MemberIDs:   dedupedMemberIDs,
	}, nil
}

func (s *ChannelService) Get(ctx context.Context, rawID uuid.UUID) (*Channel, error) {
	id, err := fields.ParseRequiredID("id", rawID)
	if err != nil {
		return nil, err
	}

	ch, err := s.cache.Get(ctx, id)
	if err != nil {
		// Non-fatal cache error
	}
	if ch != nil {
		return ch, nil
	}

	ch, err = s.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	_ = s.cache.Set(ctx, ch)

	return ch, nil
}

// UpdateGroup updates the group channel properties name and icon_url.
func (s *ChannelService) UpdateGroup(ctx context.Context, rawActorID, rawSessionID, rawChannelID uuid.UUID, rawName, rawIconURL *string) (*Channel, error) {
	actorID, sessionID, channelID, err := validateIDs(rawActorID, rawSessionID, rawChannelID)
	if err != nil {
		return nil, err
	}

	name, err := ParseChannelName(ptr.From(rawName))
	if err != nil {
		return nil, err
	}

	iconURL, err := fields.ParseURL("icon_url", ptr.From(rawIconURL))
	if err != nil {
		return nil, err
	}

	members, err := s.memberRepo.GetBatchByChannelID(ctx, channelID)
	if err != nil {
		return nil, err
	}

	_, err = validateMembership(actorID, members)
	if err != nil {
		return nil, err
	}

	var channel *Channel
	now := fields.Now()

	err = s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		channel, err = s.repo.UpdateGroup(txCtx, channelID, name, iconURL, now)
		if err != nil {
			return err
		}

		systemMessages, err := buildUpdateGroupSystemMessages(channel.ID(), actorID, name, iconURL, now)
		if err != nil {
			return err
		}

		if len(systemMessages) > 0 {
			if _, err := s.messageRepo.CreateBatchAndMention(
				txCtx,
				systemMessages,
				channel.ID(),
				actorID,
				now,
			); err != nil {
				return err
			}
		}

		payload := EventChannelUpdatedPayload{
			ExcludeSessionID: sessionID,
			Channel:          channel,
			MemberIDs:        getMemberIDs(members),
		}

		return s.outboxRepo.Publish(txCtx, EventChannelUpdated, payload, now)
	})
	if err != nil {
		return nil, err
	}

	_ = s.cache.Delete(ctx, channel.ID())

	return channel, nil
}

func buildUpdateGroupSystemMessages(
	channelID, actorID fields.ID,
	name ChannelName,
	iconURL fields.URL,
	now fields.Timestamp,
) ([]*Message, error) {
	var systemMessages []*Message

	if name.IsValid() {
		msg, err := NewMessageNameChange(channelID, actorID, name, now)
		if err != nil {
			return nil, err
		}
		systemMessages = append(systemMessages, msg)
	}

	if iconURL.IsValid() {
		iconTime := now
		if len(systemMessages) > 0 {
			iconTime = now.Add(time.Microsecond)
		}

		msg, err := NewMessageIconChange(channelID, actorID, iconTime)
		if err != nil {
			return nil, err
		}
		systemMessages = append(systemMessages, msg)
	}

	return systemMessages, nil
}
