package channel

import (
	"bonfire-api/internal/fields"
	"bonfire-api/internal/pkg/ptr"
	"bonfire-api/internal/presence"
	"bonfire-api/internal/user"
	"context"

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
	err := ValidateMaxPeers(rawPeerIDs)
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

	peerIDs := FilterPeerIDs(actorID, memberIDs)

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

	actorMember, err := ValidateMembership(actorID, members)
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
		cursor := NewMessageCursor(actorMember.LastReadMessageID(), messageID)

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

	memberIDs := GetMemberIDs(members)

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
		messageIDs := GetMessageIDs(messages)
		var err error
		reactionMap, err = s.reactionRepo.GetBatchSummaryByMessageIDs(ctx2, actorID, messageIDs)
		return err
	})

	if err := g2.Wait(); err != nil {
		return nil, nil, nil, err
	}

	SortMembers(members, userMap)
	SortMessages(messages)

	memberViews := HydrateMemberViews(members, userMap, presenceMap)
	messageViews := HydrateMessageViews(messages, userMap, reactionMap)

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

	channelIDs, actorMembershipMap := IndexMemberships(userMemberships)

	channelMap, err := s.repo.GetBatch(ctx, channelIDs)
	if err != nil {
		return nil, err
	}

	memberMap, err := s.memberRepo.GetBatchByChannelIDs(ctx, channelIDs)
	if err != nil {
		return nil, err
	}

	peerIDs, directPeerIDs := GetSidebarUserIDs(actorID, channelMap, memberMap)

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

	SortSidebar(channels, actorMembershipMap)

	return HydrateSidebarViews(actorID, channels, actorMembershipMap, memberMap, userMap, presenceMap), nil
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

		systemMessages, err := BuildUpdateGroupSystemMessages(channel.ID(), actorID, name, iconURL, now)
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
