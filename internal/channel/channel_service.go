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
	repo         ChannelRepository
	memberRepo   MemberRepository
	messageRepo  MessageRepository
	reactionRepo ReactionRepository
	userRepo     UserRepository
	userCache    UserCache
	outboxRepo   OutboxRepository
	relationRepo RelationRepository
	tx           TX
}

func NewChannelService(
	repo ChannelRepository,
	memberRepo MemberRepository,
	messageRepo MessageRepository,
	reactionRepo ReactionRepository,
	userRepo UserRepository,
	userCache UserCache,
	outboxRepo OutboxRepository,
	relationRepo RelationRepository,
	tx TX,
) *ChannelService {
	return &ChannelService{
		repo:         repo,
		memberRepo:   memberRepo,
		messageRepo:  messageRepo,
		reactionRepo: reactionRepo,
		userRepo:     userRepo,
		userCache:    userCache,
		outboxRepo:   outboxRepo,
		relationRepo: relationRepo,
		tx:           tx,
	}
}

// CreateGroup creates a new group channel with members.
func (s *ChannelService) CreateGroup(ctx context.Context, rawActorID uuid.UUID, rawPeerIDs []uuid.UUID) error {
	err := validateMaxPeers(rawPeerIDs)
	if err != nil {
		return err
	}

	actorID, err := fields.ParseRequiredID("actor_id", rawActorID)
	if err != nil {
		return err
	}

	memberIDs, err := fields.ParseIDs(rawPeerIDs)
	if err != nil {
		return err
	}

	peerIDs := filterPeerIDs(actorID, memberIDs)

	if len(peerIDs) > 0 {
		err = s.relationRepo.HasIncomingBlock(ctx, actorID, peerIDs)
		if err != nil {
			return err
		}
	}

	now := fields.Now()

	ch, err := NewGroupChannel(now)
	if err != nil {
		return err
	}

	membs := NewMembers(ch.ID(), actorID, peerIDs, now)

	return s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		if _, err = s.repo.Create(txCtx, ch); err != nil {
			return err
		}

		if _, err = s.memberRepo.CreateBatch(txCtx, membs); err != nil {
			return err
		}

		// return s.outboxRepo.Publish(txCtx, EventChannelCreated, ChannelCreatedPayload{})
		return nil
	})
}

// Get fetches all channel data needed to load a channel, including details, members, and messages.
func (s *ChannelService) Get(ctx context.Context, rawActorID, rawChannelID, rawMessageID uuid.UUID) (*Channel, []MemberView, []MessageView, error) {
	actorID, channelID, err := validateIDs(rawActorID, rawChannelID)
	if err != nil {
		return nil, nil, nil, err
	}

	messageID, err := fields.ParseID(rawMessageID)
	if err != nil {
		return nil, nil, nil, err
	}

	members, err := s.memberRepo.GetBatchByChannelID(ctx, channelID)
	if err != nil {
		return nil, nil, nil, err
	}

	actorMember, err := validateMembership(actorID, members)
	if err != nil {
		return nil, nil, nil, err
	}

	var (
		channel  *Channel
		messages []*Message
	)

	g1, ctx1 := errgroup.WithContext(ctx)

	g1.Go(func() error {
		if channel, err = s.repo.Get(ctx1, channelID); err != nil {
			return err
		}
		return nil
	})

	g1.Go(func() error {
		cursor := getMessagesCursor(actorMember.LastReadMessageID(), messageID)

		var err error
		messages, err = s.messageRepo.ListAroundByChannelID(
			ctx1,
			channelID,
			cursor.ID(),
			cursor.BeforeLimit(),
			cursor.AfterLimit(),
		)
		return err
	})

	if err := g1.Wait(); err != nil {
		return nil, nil, nil, err
	}

	memberIDs := getMemberIDs(members)

	var (
		userMap     map[fields.ID]*user.User
		presenceMap map[fields.ID]presence.Presence
		reactionMap map[fields.ID]*ReactionSummary
	)

	g2, ctx2 := errgroup.WithContext(ctx)

	g2.Go(func() error {
		var err error
		userMap, err = s.userRepo.GetBatch(ctx2, memberIDs)
		return err
	})

	g2.Go(func() error {
		var err error
		presenceMap, err = s.userCache.GetBatchPresence(ctx2, memberIDs)
		return err
	})

	g2.Go(func() error {
		if len(messages) == 0 {
			reactionMap = make(map[fields.ID]*ReactionSummary)
			return nil
		}
		messageIDs, _ := getMessageIDs(messages)
		var err error
		reactionMap, err = s.reactionRepo.GetBatchSummaryByMessageIDs(ctx2, actorID, messageIDs)
		return err
	})

	if err := g2.Wait(); err != nil {
		return nil, nil, nil, err
	}

	sortMembers(members, userMap)
	sortMessages(messages)

	return channel, hydrateMemberViews(members, userMap, presenceMap), hydrateMessageViews(messages, userMap, reactionMap), nil
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
