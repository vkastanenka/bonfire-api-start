package channel

import (
	"bonfire-api/internal/errs"
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
func (s *ChannelService) CreateGroup(ctx context.Context, rawUserID uuid.UUID, rawPeerIDs []uuid.UUID) error {
	// Validate
	err := ValidateMaxPeers(rawPeerIDs)
	if err != nil {
		return err
	}

	userID, err := fields.ParseRequiredID("user_id", rawUserID)
	if err != nil {
		return err
	}

	memberIDs, err := fields.ParseIDs("member_id", rawPeerIDs)
	if err != nil {
		return err
	}

	// Dedupe peers and remove requesting user
	peerIDs := fields.RemoveID(fields.DedupeIDs(memberIDs), userID)

	allMemberIDs := make([]fields.ID, 0, len(peerIDs)+1)
	allMemberIDs = append(allMemberIDs, userID)
	allMemberIDs = append(allMemberIDs, peerIDs...)

	// Verify blocks
	if len(peerIDs) > 0 {
		hasBlock, err := s.relationRepo.HasIncomingBlock(ctx, userID, peerIDs)
		if err != nil {
			return err
		}
		if hasBlock {
			return errs.InvalidArgument("Cannot interact with users who have blocked you.").
				Reason("INCOMING_BLOCK_DETECTED")
		}
	}

	// Parse models
	channelID, err := fields.NewID()
	if err != nil {
		return err
	}

	now := fields.Now()

	parsedChannel := ParseChannel(
		channelID,
		NewChannelTypeGroup(),
		ChannelName{},
		fields.URL{},
		fields.ID{},
		fields.Timestamp{},
		now,
		now,
	)

	creatorMember := ParseMember(
		channelID,
		userID,
		fields.ID{},
		fields.Timestamp{},
		fields.Timestamp{},
		fields.Timestamp{},
		0,
		true,
		now,
		now,
	)

	peerMembers := ParseMembers(
		channelID,
		peerIDs,
		fields.ID{},
		fields.Timestamp{},
		fields.Timestamp{},
		fields.Timestamp{},
		1,
		true,
		now,
		now,
	)

	parsedMembers := make([]*Member, 0, len(allMemberIDs))
	parsedMembers = append(parsedMembers, creatorMember)
	parsedMembers = append(parsedMembers, peerMembers...)

	err = s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		// Create channel
		_, err = s.repo.Create(txCtx, parsedChannel)
		if err != nil {
			return err
		}

		// Create members
		_, err = s.memberRepo.CreateBatch(txCtx, parsedMembers)
		if err != nil {
			return err
		}

		// Publish event
		_, err = s.outboxRepo.Publish(
			txCtx,
			EventChannelCreated,
			ChannelCreatedPayload{},
		)
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

// Get fetches all channel data needed to load a channel, including details, members, and messages.
func (s *ChannelService) Get(ctx context.Context, rawUserID, rawChannelID, rawMessageID uuid.UUID) (*Channel, []MemberView, []MessageView, error) {
	// Validate
	userID, err := fields.ParseRequiredID("user_id", rawUserID)
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

	// Fetch members
	memberMap, err := s.memberRepo.GetBatchByChannelIDs(ctx, []fields.ID{channelID})
	if err != nil {
		return nil, nil, nil, err
	}

	members, ok := memberMap[channelID]
	if !ok || len(members) == 0 {
		return nil, nil, nil, ErrMembersNotFound()
	}

	// Validate membership
	currentMember, err := ValidateMembership(userID, members)
	if err != nil {
		return nil, nil, nil, err
	}

	var (
		channel  *Channel
		messages []*Message
	)

	// Fetch channel and messages
	g1, ctx1 := errgroup.WithContext(ctx)

	g1.Go(func() error {
		channel, err = s.repo.Get(ctx1, channelID)
		if err != nil {
			return err
		}
		return nil
	})

	g1.Go(func() error {
		anchorMessageID := messageID
		beforeLimit := MessageListBeforeLimit
		afterLimit := MessageListAfterLimit

		if !anchorMessageID.IsValid() {
			anchorMessageID = currentMember.LastReadMessageID()

			if !anchorMessageID.IsValid() {
				beforeLimit = MessageListLimit
				afterLimit = 0
			}
		}

		var err error
		messages, err = s.messageRepo.ListAroundByChannelID(
			ctx1,
			channelID,
			anchorMessageID,
			beforeLimit,
			afterLimit,
		)
		return err
	})

	if err := g1.Wait(); err != nil {
		return nil, nil, nil, err
	}

	// Dedupe IDs
	rawUserIDs := make([]fields.ID, 0, len(members)+len(messages))

	for _, m := range members {
		rawUserIDs = append(rawUserIDs, m.UserID())
	}
	for _, msg := range messages {
		rawUserIDs = append(rawUserIDs, msg.AuthorID())
	}

	userIDs := fields.DedupeIDs(rawUserIDs)

	var (
		reactionMap map[fields.ID]*ReactionSummary
		userMap     map[fields.ID]*user.User
		presenceMap map[fields.ID]presence.Presence
	)

	// Fetch reactions, profiles, and presences
	g2, ctx2 := errgroup.WithContext(ctx)

	g2.Go(func() error {
		if len(messages) == 0 {
			reactionMap = make(map[fields.ID]*ReactionSummary)
			return nil
		}
		messageIDs := make([]fields.ID, len(messages))
		for i, msg := range messages {
			messageIDs[i] = msg.ID()
		}
		var err error
		reactionMap, err = s.reactionRepo.GetBatchSummaryByMessageIDs(ctx2, userID, messageIDs)
		return err
	})

	g2.Go(func() error {
		var err error
		userMap, err = s.userRepo.GetBatch(ctx2, userIDs)
		return err
	})

	g2.Go(func() error {
		var err error
		presenceMap, err = s.userCache.GetBatchPresence(ctx2, userIDs)
		return err
	})

	if err := g2.Wait(); err != nil {
		return nil, nil, nil, err
	}

	// Sort
	SortMembers(members, userMap)
	SortMessages(messages)

	// Hydrate
	memberViews := HydrateMemberViews(members, userMap, presenceMap)
	messageViews := HydrateMessageViews(messages, userMap, reactionMap)

	return channel, memberViews, messageViews, nil
}

// GetSidebar fetches all sidebar needed to load a user's sidebar, including details, members, and presences.
func (s *ChannelService) GetSidebar(ctx context.Context, rawUserID uuid.UUID) ([]SidebarView, error) {
	// Validate
	userID, err := fields.ParseRequiredID("user_id", rawUserID)
	if err != nil {
		return nil, err
	}

	// List user memberships
	userMemberships, err := s.memberRepo.ListVisibleByUserID(ctx, userID, ChannelMaxSidebarItems)
	if err != nil {
		return nil, err
	}

	if len(userMemberships) == 0 {
		return []SidebarView{}, nil
	}

	// Index user memberships
	channelIDs, userMembersMap := IndexMemberships(userMemberships)

	// Fetch channels and members
	channelMap, err := s.repo.GetBatch(ctx, channelIDs)
	if err != nil {
		return nil, err
	}

	channelMembersMap, err := s.memberRepo.GetBatchByChannelIDs(ctx, channelIDs)
	if err != nil {
		return nil, err
	}

	// Collect peer IDs
	var rawUserIDs []fields.ID
	var rawDirectPeerIDs []fields.ID

	for chID, members := range channelMembersMap {
		ch, exists := channelMap[chID]
		if !exists || ch == nil {
			continue
		}
		for _, m := range members {
			if m.UserID() == userID {
				continue
			}
			rawUserIDs = append(rawUserIDs, m.UserID())
			if ch.Type().IsDirect() {
				rawDirectPeerIDs = append(rawDirectPeerIDs, m.UserID())
			}
		}
	}

	// Dedupe IDs
	userIDs := fields.DedupeIDs(rawUserIDs)
	directPeerIDs := fields.DedupeIDs(rawDirectPeerIDs)

	var (
		userMap     map[fields.ID]*user.User
		presenceMap map[fields.ID]presence.Presence
	)

	// Fetch users and presences
	g, gCtx := errgroup.WithContext(ctx)

	if len(userIDs) > 0 {
		g.Go(func() error {
			var err error
			userMap, err = s.userRepo.GetBatch(gCtx, userIDs)
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

	// Prepare channels
	channels := make([]*Channel, 0, len(channelMap))
	for _, ch := range channelMap {
		if ch != nil {
			channels = append(channels, ch)
		}
	}

	// Sort channels
	SortSidebar(channels, userMembersMap)

	// Hydrate views
	sidebarViews := HydrateSidebarViews(userID, channels, userMembersMap, channelMembersMap, userMap, presenceMap)

	return sidebarViews, nil
}

// UpdateGroup updates the group channel properties name and icon_url.
func (s *ChannelService) UpdateGroup(ctx context.Context, rawUserID, rawChannelID uuid.UUID, rawName, rawIconURL *string) (*Channel, error) {
	// Validate inputs
	userID, err := fields.ParseRequiredID("user_id", rawUserID)
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

	// Validate membership
	_, err = s.memberRepo.Get(ctx, channelID, userID)
	if err != nil {
		if errs.IsNotFound(err) {
			return nil, errs.PermissionDenied("You are not a member of this channel.")
		}
		return nil, err
	}

	now := fields.Now()

	var channel *Channel
	var systemMessages []*Message

	err = s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		// Update group
		channel, err = s.repo.UpdateGroup(txCtx, channelID, name, iconURL, now)
		if err != nil {
			return err
		}

		// Create system messages
		if rawName != nil && !name.IsZero() {
			messageNameChangeID, err := fields.NewID()
			if err != nil {
				return err
			}

			msg := ParseMessageNameChange(
				messageNameChangeID,
				channel.ID(),
				userID,
				name,
				now,
			)
			systemMessages = append(systemMessages, msg)
		}

		if rawIconURL != nil {
			messageIconChangeID, err := fields.NewID()
			if err != nil {
				return err
			}

			iconTime := now
			if len(systemMessages) > 0 {
				iconTime = now.Add(time.Microsecond)
			}

			msg := ParseMessageIconChange(
				messageIconChangeID,
				channel.ID(),
				userID,
				iconTime,
			)
			systemMessages = append(systemMessages, msg)
		}

		if len(systemMessages) > 0 {
			_, err = s.messageRepo.CreateBatch(txCtx, systemMessages)
			if err != nil {
				return err
			}
		}

		// Publish event
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
