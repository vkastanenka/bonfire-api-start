package channel

import (
	"bonfire-api/internal/errs"
	"bonfire-api/internal/fields"
	"bonfire-api/internal/pkg/ptr"
	"bonfire-api/internal/presence"
	"bonfire-api/internal/user"
	"context"
	"log/slog"
	"slices"
	"time"

	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"
)

type ChannelService struct {
	repo          ChannelRepository
	cache         ChannelCache
	memberRepo    MemberRepository
	memberCache   MemberCache
	messageRepo   MessageRepository
	messageCache  MessageCache
	reactionRepo  ReactionRepository
	reactionCache ReactionCache
	userRepo      UserRepository
	userCache     UserCache
	outboxRepo    OutboxRepository
	relationRepo  RelationRepository
	presenceCache PresenceCache
	tx            TX
}

func NewChannelService(
	repo ChannelRepository,
	cache ChannelCache,
	memberRepo MemberRepository,
	memberCache MemberCache,
	messageRepo MessageRepository,
	messageCache MessageCache,
	reactionRepo ReactionRepository,
	reactionCache ReactionCache,
	userRepo UserRepository,
	userCache UserCache,
	outboxRepo OutboxRepository,
	relationRepo RelationRepository,
	presenceCache PresenceCache,
	tx TX,
) *ChannelService {
	return &ChannelService{
		repo:          repo,
		cache:         cache,
		memberRepo:    memberRepo,
		memberCache:   memberCache,
		messageRepo:   messageRepo,
		messageCache:  messageCache,
		reactionRepo:  reactionRepo,
		reactionCache: reactionCache,
		userRepo:      userRepo,
		userCache:     userCache,
		outboxRepo:    outboxRepo,
		relationRepo:  relationRepo,
		presenceCache: presenceCache,
		tx:            tx,
	}
}

func (s *ChannelService) CreateGroup(ctx context.Context, rawUserID uuid.UUID, rawPeerIDs []uuid.UUID) error {
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

	peerIDs := fields.RemoveID(fields.DedupeIDs(memberIDs), userID)

	allMemberIDs := make([]fields.ID, 0, len(peerIDs)+1)
	allMemberIDs = append(allMemberIDs, userID)
	allMemberIDs = append(allMemberIDs, peerIDs...)

	err = s.ensureNoIncomingBlocks(ctx, userID, peerIDs)
	if err != nil {
		return err
	}

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

	var newChannel *Channel
	var newChannelMembers []*Member

	err = s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		ch, err := s.repo.Create(txCtx, parsedChannel)
		if err != nil {
			return err
		}

		mem, err := s.memberRepo.CreateBatch(txCtx, parsedMembers)
		if err != nil {
			return err
		}

		_, err = s.outboxRepo.Publish(
			txCtx,
			EventChannelCreated,
			ChannelCreatedPayload{},
		)
		if err != nil {
			return err
		}

		newChannel = ch
		newChannelMembers = mem

		return nil
	})
	if err != nil {
		return err
	}

	cacheCtx := context.WithoutCancel(ctx)

	if err := s.cache.Set(cacheCtx, newChannel); err != nil {
		slog.WarnContext(cacheCtx, "failed to cache channel entity",
			"channel_id", channelID.String(),
			"error", err,
		)
	}

	if err := s.cache.SetMemberIDs(cacheCtx, channelID, allMemberIDs); err != nil {
		slog.WarnContext(cacheCtx, "failed to cache channel member ids",
			"channel_id", channelID.String(),
			"count", len(allMemberIDs),
			"error", err,
		)
	}

	if err := s.cache.SetLoaded(cacheCtx, channelID); err != nil {
		slog.WarnContext(cacheCtx, "failed to set channel loaded flag",
			"channel_id", channelID.String(),
			"error", err,
		)
	}

	if err := s.memberCache.SetBatch(cacheCtx, newChannelMembers); err != nil {
		slog.WarnContext(cacheCtx, "failed to batch cache channel members",
			"channel_id", channelID.String(),
			"count", len(newChannelMembers),
			"error", err,
		)
	}

	if err := s.userCache.DeleteChannelIDsBatch(cacheCtx, allMemberIDs); err != nil {
		slog.WarnContext(cacheCtx, "failed to invalidate user channel ids batch",
			"count", len(allMemberIDs),
			"error", err,
		)
	}

	return nil
}

func (s *ChannelService) Get(ctx context.Context, rawUserID, rawChannelID, rawMessageID uuid.UUID) (*Channel, []MemberView, []MessageView, error) {
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

	memberMap, err := s.memberRepo.GetBatchByChannelIDs(ctx, []fields.ID{channelID})
	if err != nil {
		return nil, nil, nil, err
	}

	members, ok := memberMap[channelID]
	if !ok || len(members) == 0 {
		return nil, nil, nil, errs.NotFound("Channel members not found.")
	}

	currentMember, err := ValidateMembership(members, userID)
	if err != nil {
		return nil, nil, nil, err
	}

	var (
		channel  *Channel
		messages []*Message
	)

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
		presenceMap, err = s.presenceCache.GetBatch(ctx2, userIDs)
		return err
	})

	if err := g2.Wait(); err != nil {
		return nil, nil, nil, err
	}

	return channel, HydrateMemberViews(members, userMap, presenceMap), HydrateMessageViews(messages, userMap, reactionMap), nil
}

func (s *ChannelService) GetSidebar(ctx context.Context, rawUserID uuid.UUID) ([]ChannelSidebarView, error) {
	userID, err := fields.ParseRequiredID("user_id", rawUserID)
	if err != nil {
		return nil, err
	}

	userMemberships, err := s.memberRepo.ListVisibleByUserID(ctx, userID, ChannelMaxSidebarItems)
	if err != nil {
		return nil, err
	}
	if len(userMemberships) == 0 {
		return []ChannelSidebarView{}, nil
	}

	channelIDs := make([]fields.ID, len(userMemberships))
	membershipMap := make(map[fields.ID]*Member, len(userMemberships))
	for i, m := range userMemberships {
		channelIDs[i] = m.ChannelID()
		membershipMap[m.ChannelID()] = m
	}

	// --- PHASE 1: Fetch Channels & Resolve Peer IDs ---
	channelMap, err := s.repo.GetBatch(ctx, channelIDs)
	if err != nil {
		return nil, err
	}

	var membersMap map[fields.ID][]*Member
	if len(channelIDs) > 0 {
		membersMap, err = s.memberRepo.GetBatchByChannelIDs(ctx, channelIDs)
		if err != nil {
			return nil, err
		}
	}

	// Deduplicate User IDs for profile hydration and direct presence lookups
	userIDSet := make(map[fields.ID]struct{})
	directPeerIDSet := make(map[fields.ID]struct{})

	for chID, members := range membersMap {
		ch, exists := channelMap[chID]
		if !exists || ch == nil {
			continue
		}
		for _, m := range members {
			if m.UserID() == userID {
				continue // Exclude current user
			}
			userIDSet[m.UserID()] = struct{}{}
			if ChannelTypeValue(ch.Type().Value) == ChannelTypeDirect {
				directPeerIDSet[m.UserID()] = struct{}{}
			}
		}
	}

	userIDs := make([]fields.ID, 0, len(userIDSet))
	for id := range userIDSet {
		userIDs = append(userIDs, id)
	}

	directPeerIDs := make([]fields.ID, 0, len(directPeerIDSet))
	for id := range directPeerIDSet {
		directPeerIDs = append(directPeerIDs, id)
	}

	// --- PHASE 2: Concurrent Users & Presence Fetch ---
	var (
		userMap     map[fields.ID]*user.User
		presenceMap map[fields.ID]presence.Presence
	)

	g, gCtx := errgroup.WithContext(ctx)

	// Fetch Users (only if userIDs is non-empty)
	if len(userIDs) > 0 {
		g.Go(func() error {
			var err error
			userMap, err = s.userRepo.GetBatch(gCtx, userIDs)
			return err
		})
	}

	// Fetch Presences (1:1 Direct Peers only, if non-empty)
	if len(directPeerIDs) > 0 {
		g.Go(func() error {
			var err error
			presenceMap, err = s.presenceCache.GetBatch(gCtx, directPeerIDs)
			return err
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	// --- PHASE 3: In-Memory Sort ---
	channels := make([]*Channel, 0, len(channelMap))
	for _, ch := range channelMap {
		if ch != nil {
			channels = append(channels, ch)
		}
	}

	slices.SortFunc(channels, func(a, b *Channel) int {
		mA := membershipMap[a.ID()]
		mB := membershipMap[b.ID()]

		// 1. Pinned priority
		aPinned := mA != nil && mA.PinnedAt().IsValid()
		bPinned := mB != nil && mB.PinnedAt().IsValid()
		if aPinned != bPinned {
			if aPinned {
				return -1
			}
			return 1
		}
		if aPinned && bPinned {
			if mA.PinnedAt().After(mB.PinnedAt()) {
				return -1
			}
			if mB.PinnedAt().After(mA.PinnedAt()) {
				return 1
			}
		}

		// 2. Activity (lastMessageAt)
		aLast := a.LastMessageAt()
		bLast := b.LastMessageAt()
		if !aLast.Equals(bLast) {
			if aLast.After(bLast) {
				return -1
			}
			return 1
		}

		// 3. Creation date
		if a.CreatedAt().After(b.CreatedAt()) {
			return -1
		}
		if b.CreatedAt().After(a.CreatedAt()) {
			return 1
		}

		// 4. Guaranteed deterministic ID tie-breaker
		return a.ID().Compare(b.ID())
	})

	// --- PHASE 4: Hydrate Views ---
	return s.hydrateSidebarViews(channels, membershipMap, membersMap, userMap, presenceMap), nil
}

func (s *ChannelService) hydrateSidebarViews(
	channels []*Channel,
	membershipMap map[fields.ID]*Member,
	peerMembersMap map[fields.ID][]*Member,
	userMap map[fields.ID]*user.User,
	presenceMap map[fields.ID]presence.Presence,
) []ChannelSidebarView {
	views := make([]ChannelSidebarView, 0, len(channels))

	for _, ch := range channels {
		mem := membershipMap[ch.ID()]
		if mem == nil {
			continue
		}

		rawPeers := peerMembersMap[ch.ID()]
		peersView := make([]ChannelSidebarPeerView, 0, len(rawPeers))

		for _, pMem := range rawPeers {
			if pMem.UserID() == mem.UserID() {
				continue
			}

			u, ok := userMap[pMem.UserID()]
			if !ok || u == nil {
				continue
			}

			p, ok := presenceMap[pMem.UserID()]
			if !ok {
				p = presence.New(presence.PresenceOffline)
			}

			peersView = append(peersView, ChannelSidebarPeerView{
				id:          u.ID(),
				displayName: u.DisplayName(),
				avatarURL:   u.AvatarURL(),
				presence:    p,
			})
		}

		memberTotal := int16(len(rawPeers))

		views = append(views, ChannelSidebarView{
			id:                ch.ID(),
			chType:            ch.Type(),
			name:              ch.Name(),
			iconURL:           ch.IconURL(),
			lastMessageID:     ch.LastMessageID(),
			lastMessageAt:     ch.LastMessageAt(),
			lastReadMessageID: mem.LastReadMessageID(),
			pinnedAt:          mem.PinnedAt(),
			mutedUntil:        mem.MutedUntil(),
			mentionCount:      mem.MentionCount(),
			peers:             peersView,
			memberTotal:       memberTotal,
		})
	}

	return views
}

func (s *ChannelService) UpdateGroup(ctx context.Context, rawUserID, rawChannelID uuid.UUID, rawName, rawIconURL *string) (*Channel, error) {
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

	_, err = s.memberRepo.Get(ctx, channelID, userID)
	if err != nil {
		if errs.IsNotFound(err) {
			return nil, errs.PermissionDenied("You are not a member of this channel.")
		}
		return nil, err
	}

	updatedAt := fields.NewTimestamp(time.Now())
	var updatedChannel *Channel

	err = s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		channelRow, err := s.repo.UpdateGroup(txCtx, channelID, name, iconURL, updatedAt)
		if err != nil {
			return err
		}

		_, err = s.outboxRepo.Publish(txCtx, EventChannelUpdated, ChannelUpdatedPayload{})
		if err != nil {
			return err
		}

		// TODO: Create system message for name change if not null
		// TODO: Create system message for icon change if not null

		updatedChannel = channelRow
		return nil
	})
	if err != nil {
		return nil, err
	}

	s.cache.Delete(ctx, updatedChannel.ID())

	return updatedChannel, nil
}

func (s *ChannelService) ensureNoIncomingBlocks(ctx context.Context, userID fields.ID, peerIDs []fields.ID) error {
	if len(peerIDs) == 0 {
		return nil
	}

	hasBlock, err := s.relationRepo.HasIncomingBlock(ctx, userID, peerIDs)
	if err != nil {
		return err
	}
	if hasBlock {
		return errs.InvalidArgument("Cannot interact with users who have blocked you.").
			Reason("INCOMING_BLOCK_DETECTED")
	}

	return nil
}
