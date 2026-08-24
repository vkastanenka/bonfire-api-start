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

		if _, err = s.outboxRepo.Publish(txCtx, EventChannelCreated, ChannelCreatedPayload{}); err != nil {
			return err
		}

		return nil
	})
}

// Get fetches all channel data needed to load a channel, including details, members, and messages.
func (s *ChannelService) Get(ctx context.Context, rawActorID, rawChannelID, rawMessageID uuid.UUID) (*Channel, []MemberView, []MessageView, error) {
	actorID, err := fields.ParseRequiredID("actor_id", rawActorID)
	if err != nil {
		return nil, nil, nil, err
	}

	channelID, err := fields.ParseRequiredID("channel_id", rawChannelID)
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
		cursor := getMessageCursor(actorMember.LastReadMessageID(), messageID)

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
		messageIDs := getMessageIDs(messages)
		var err error
		reactionMap, err = s.reactionRepo.GetBatchSummaryByMessageIDs(ctx2, actorID, messageIDs)
		return err
	})

	if err := g2.Wait(); err != nil {
		return nil, nil, nil, err
	}

	sortMembers(members, userMap)
	sortMessages(messages)

	memberViews := hydrateMemberViews(members, userMap, presenceMap)
	messageViews := hydrateMessageViews(messages, userMap, reactionMap)

	return channel, memberViews, messageViews, nil
}

// GetSidebar fetches all sidebar needed to load a user's sidebar, including details, members, and presences.
func (s *ChannelService) GetSidebar(ctx context.Context, rawActorID uuid.UUID) ([]SidebarView, error) {
	actorID, err := fields.ParseRequiredID("actor_id", rawActorID)
	if err != nil {
		return nil, err
	}

	userMemberships, err := s.memberRepo.ListVisibleByUserID(ctx, actorID, ChannelMaxSidebarItems)
	if err != nil {
		return nil, err
	}

	if len(userMemberships) == 0 {
		return []SidebarView{}, nil
	}

	channelIDs, actorMembershipMap := indexMemberships(userMemberships)

	channelMap, err := s.repo.GetBatch(ctx, channelIDs)
	if err != nil {
		return nil, err
	}

	memberMap, err := s.memberRepo.GetBatchByChannelIDs(ctx, channelIDs)
	if err != nil {
		return nil, err
	}

	peerIDs, directPeerIDs := getSidebarUserIDs(actorID, channelMap, memberMap)

	var (
		userMap     map[fields.ID]*user.User
		presenceMap map[fields.ID]presence.Presence
	)

	g, gCtx := errgroup.WithContext(ctx)

	if len(peerIDs) > 0 {
		g.Go(func() error {
			var err error
			userMap, err = s.userRepo.GetBatch(gCtx, peerIDs)
			return err
		})
	}

	if len(directPeerIDs) > 0 {
		g.Go(func() error {
			var err error
			presenceMap, err = s.userCache.GetBatchPresence(gCtx, directPeerIDs)
			return err
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	channels := make([]*Channel, 0, len(channelMap))
	for _, ch := range channelMap {
		if ch != nil {
			channels = append(channels, ch)
		}
	}

	sortSidebar(channels, actorMembershipMap)

	return hydrateSidebarViews(actorID, channels, actorMembershipMap, memberMap, userMap, presenceMap), nil
}

// UpdateGroup updates the group channel properties name and icon_url.
func (s *ChannelService) UpdateGroup(ctx context.Context, rawActorID, rawChannelID uuid.UUID, rawName, rawIconURL *string) (*Channel, error) {
	actorID, err := fields.ParseRequiredID("actor_id", rawActorID)
	if err != nil {
		return nil, err
	}

	channelID, err := fields.ParseRequiredID("channel_id", rawChannelID)
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

	now := fields.Now()

	var channel *Channel

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
			if _, err := s.messageRepo.CreateBatch(txCtx, systemMessages); err != nil {
				return err
			}
		}

		_, err = s.outboxRepo.Publish(txCtx, EventChannelUpdated, ChannelUpdatedPayload{})
		if err != nil {
			return err
		}

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

func getMessageCursor(
	actorLastReadID fields.ID,
	fallbackMessageID fields.ID,
) fields.Cursor {
	cursorID := fallbackMessageID
	beforeLimit := MessageListBeforeLimit
	afterLimit := MessageListAfterLimit

	if !cursorID.IsValid() {
		if actorLastReadID.IsValid() {
			cursorID = actorLastReadID
		}

		if !cursorID.IsValid() {
			beforeLimit = MessageListLimit
			afterLimit = 0
		}
	}

	return fields.NewCursor(cursorID, beforeLimit, afterLimit)
}

func getSidebarUserIDs(actorID fields.ID, channelMap map[fields.ID]*Channel, memberMap map[fields.ID][]*Member) (peerIDs []fields.ID, directPeerIDs []fields.ID) {
	var rawPeerIDs []fields.ID
	var rawDirectPeerIDs []fields.ID

	for chID, members := range memberMap {
		ch, exists := channelMap[chID]
		if !exists || ch == nil {
			continue
		}
		for _, m := range members {
			if m.UserID().Equals(actorID) {
				continue
			}
			rawPeerIDs = append(rawPeerIDs, m.UserID())
			if ch.Type().IsDirect() {
				rawDirectPeerIDs = append(rawDirectPeerIDs, m.UserID())
			}
		}
	}

	return fields.DedupeIDs(rawPeerIDs), fields.DedupeIDs(rawDirectPeerIDs)
}
