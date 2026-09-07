package channel

import (
	"bonfire-api/internal/fields"
	"bonfire-api/internal/pkg/ptr"
	"bonfire-api/internal/presence"
	"bonfire-api/internal/user"
	"context"
	"slices"
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

		outboxPayload := EventChannelCreatedPayload{
			ExcludeSessionID: sessionID,
			Channel:          ch,
			Users:            users,
			Presences:        presences,
			MemberIDs:        dedupedMemberIDs,
		}

		return s.outboxRepo.Publish(txCtx, EventChannelCreated, outboxPayload, now)
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

type GetResult struct {
	Channel       *Channel
	Member        *Member
	Messages      []*Message
	Reactions     map[fields.ID]*ReactionSummary
	Users         map[fields.ID]*user.User
	Presences     map[fields.ID]presence.Presence
	MemberIDs     []fields.ID
	HasMoreBefore bool
	HasMoreAfter  bool
}

// Get fetches all channel data needed to load a channel, including details, members, and messages.
func (s *ChannelService) Get(ctx context.Context, rawActorID, rawChannelID, rawMessageID uuid.UUID) (*GetResult, error) {
	actorID, channelID, err := validateIDs(rawActorID, rawChannelID)
	if err != nil {
		return nil, err
	}

	messageID, err := fields.ParseID(rawMessageID)
	if err != nil {
		return nil, err
	}

	members, err := s.memberRepo.GetBatchByChannelID(ctx, channelID)
	if err != nil {
		return nil, err
	}

	actorMember, err := validateMembership(actorID, members)
	if err != nil {
		return nil, err
	}

	memberIDs := getMemberIDs(members)

	var (
		channel       *Channel
		messages      []*Message
		hasMoreBefore bool
		hasMoreAfter  bool
	)

	g1, g1Ctx := errgroup.WithContext(ctx)

	g1.Go(func() error {
		var getErr error
		channel, getErr = s.repo.Get(g1Ctx, channelID)
		return getErr
	})

	g1.Go(func() error {
		cursor := getMessagesCursor(actorMember.LastReadMessageID(), messageID)

		var listErr error
		messages, hasMoreBefore, hasMoreAfter, listErr = s.messageRepo.ListAroundByChannelID(
			g1Ctx,
			channelID,
			cursor.ID(),
			cursor.BeforeLimit(),
			cursor.AfterLimit(),
		)
		return listErr
	})

	if err := g1.Wait(); err != nil {
		return nil, err
	}

	allUserIDs := getChannelUserIDs(memberIDs, messages)

	var (
		users     map[fields.ID]*user.User
		presences map[fields.ID]presence.Presence
		reactions map[fields.ID]*ReactionSummary
	)

	g2, g2Ctx := errgroup.WithContext(ctx)

	g2.Go(func() error {
		var fetchErr error
		users, fetchErr = s.userService.GetBatch(g2Ctx, allUserIDs)
		return fetchErr
	})

	g2.Go(func() error {
		var fetchErr error
		presences, fetchErr = s.presenceCache.GetBatchPresence(g2Ctx, memberIDs)
		return fetchErr
	})

	g2.Go(func() error {
		if len(messages) == 0 {
			reactions = make(map[fields.ID]*ReactionSummary)
			return nil
		}

		messageIDs, _ := getMessageIDs(messages)
		var fetchErr error
		reactions, fetchErr = s.reactionRepo.GetBatchSummaryByMessageIDs(g2Ctx, actorID, messageIDs)
		return fetchErr
	})

	if err := g2.Wait(); err != nil {
		return nil, err
	}

	sortMemberIDs(memberIDs, users)

	return &GetResult{
		Channel:       channel,
		Member:        actorMember,
		Messages:      messages,
		Reactions:     reactions,
		Users:         users,
		Presences:     presences,
		MemberIDs:     memberIDs,
		HasMoreBefore: hasMoreBefore,
		HasMoreAfter:  hasMoreAfter,
	}, nil
}

// GetSidebar fetches all sidebar related structures.
func (s *ChannelService) GetSidebar(ctx context.Context, rawActorID uuid.UUID) (
	channelMap map[fields.ID]*Channel,
	memberMap map[fields.ID]*Member,
	peerIDsMap map[fields.ID][]fields.ID,
	channelIDs []fields.ID,
	peerIDs []fields.ID,
	err error,
) {
	actorID, err := fields.ParseRequiredID("actor_id", rawActorID)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}

	userMemberships, err := s.memberRepo.ListVisibleByUserID(ctx, actorID, ChannelMaxSidebarItems)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}

	if len(userMemberships) == 0 {
		return make(map[fields.ID]*Channel), make(map[fields.ID]*Member), make(map[fields.ID][]fields.ID), []fields.ID{}, []fields.ID{}, nil
	}

	channelIDs, memberMap = indexMemberships(userMemberships)

	channelMap, err = s.repo.GetBatch(ctx, channelIDs)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}

	channelMembersMap, err := s.memberRepo.GetBatchByChannelIDs(ctx, channelIDs)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}

	channels := getChannels(channelMap)
	sortSidebar(channels, memberMap)
	channelIDs = indexChannels(channels)
	peerIDs, _ = getSidebarUserIDs(actorID, channelMap, channelMembersMap)
	peerIDsMap = getSidebarPeerIDsMap(actorID, channelMembersMap)
	return channelMap, memberMap, peerIDsMap, channelIDs, peerIDs, nil
}

// UpdateGroup updates the group channel properties name and icon_url.
func (s *ChannelService) UpdateGroup(ctx context.Context, rawActorID, rawChannelID uuid.UUID, rawName, rawIconURL *string) (*Channel, error) {
	actorID, channelID, err := validateIDs(rawActorID, rawChannelID)
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

	_, err = s.memberRepo.Require(ctx, channelID, actorID)
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

		// return s.outboxRepo.Publish(txCtx, EventChannelUpdated, ChannelUpdatedPayload{})
		return nil
	})
	if err != nil {
		return nil, err
	}

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

func getMessagesCursor(
	actorLastReadID fields.ID,
	fallbackMessageID fields.ID,
) fields.Cursor {
	if fallbackMessageID.IsValid() {
		return fields.NewCursor(fallbackMessageID, MessageListBeforeLimit, MessageListAfterLimit)
	}

	if actorLastReadID.IsValid() {
		return fields.NewCursor(actorLastReadID, MessageListBeforeLimit, MessageListAfterLimit)
	}

	return fields.NewCursor(fallbackMessageID, MessageListLimit, 0)
}

func getSidebarPeerIDsMap(actorID fields.ID, memberMap map[fields.ID][]*Member) map[fields.ID][]fields.ID {
	channelPeerIDsMap := make(map[fields.ID][]fields.ID, len(memberMap))
	for chID, members := range memberMap {
		var userIDs []fields.ID
		for _, m := range members {
			if m != nil && !m.UserID().Equals(actorID) {
				userIDs = append(userIDs, m.UserID())
			}
		}

		slices.SortFunc(userIDs, func(a, b fields.ID) int {
			return a.Compare(b)
		})

		channelPeerIDsMap[chID] = userIDs
	}
	return channelPeerIDsMap
}

func getSidebarUserIDs(
	actorID fields.ID,
	channelMap map[fields.ID]*Channel,
	memberMap map[fields.ID][]*Member,
) (peerIDs, directPeerIDs []fields.ID) {
	for chID, members := range memberMap {
		ch := channelMap[chID]
		if ch == nil {
			continue
		}

		isDirect := ch.Type().IsDirect()
		for _, m := range members {
			userID := m.UserID()
			if userID.Equals(actorID) {
				continue
			}

			peerIDs = append(peerIDs, userID)
			if isDirect {
				directPeerIDs = append(directPeerIDs, userID)
			}
		}
	}

	return fields.DedupeIDs(peerIDs), fields.DedupeIDs(directPeerIDs)
}
