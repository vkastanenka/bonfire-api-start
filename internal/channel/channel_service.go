package channel

import (
	"bonfire-api/internal/errs"
	"bonfire-api/internal/fields"
	"bonfire-api/internal/pkg/ptr"
	"bonfire-api/internal/presence"
	"bonfire-api/internal/user"
	"context"
	"fmt"
	"slices"
	"sort"
	"time"

	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"
)

type ChannelRepository interface {
	Create(ctx context.Context, ch *Channel) (*Channel, error)
	Delete(ctx context.Context, id fields.ID) error
	Get(ctx context.Context, id fields.ID) (*Channel, error)
	GetForUpdate(ctx context.Context, id fields.ID) (*Channel, error)
	GetBatch(ctx context.Context, ids []fields.ID) (map[fields.ID]*Channel, error)
	UpdateGroup(ctx context.Context, id fields.ID, name ChannelName, iconURL fields.URL, updatedAt fields.Timestamp) (*Channel, error)
	UpdateLastMessage(ctx context.Context, id fields.ID, lastMessageID fields.ID, lastMessageAt fields.Timestamp, updatedAt fields.Timestamp) (*Channel, error)
}

type ChannelCache interface {
	Delete(ctx context.Context, key fields.ID) error
	DeleteBatch(ctx context.Context, keys []fields.ID) error
	Get(ctx context.Context, id fields.ID) (*Channel, error)
	GetBatch(ctx context.Context, ids []fields.ID) (map[fields.ID]*Channel, []fields.ID, error)
	Set(ctx context.Context, ch *Channel) error
	SetBatch(ctx context.Context, channels []*Channel) error
}

type RelationRepository interface {
	HasIncomingBlock(ctx context.Context, actorID fields.ID, peerIDs []fields.ID) (bool, error)
}

type UserRepository interface {
	Get(ctx context.Context, id fields.ID) (*user.User, error)
	GetBatch(ctx context.Context, ids []fields.ID) (map[fields.ID]*user.User, error)
}

type UserCache interface {
	SetBatch(ctx context.Context, users []*user.User) error
}

type PresenceCache interface {
	GetBatch(ctx context.Context, userIDs []fields.ID) (map[fields.ID]presence.Presence, error)
}

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

func (s *ChannelService) CreateGroup(ctx context.Context, rawUserID uuid.UUID, rawMemberIDs []uuid.UUID) error {
	if len(rawMemberIDs) > ChannelMaxPeers {
		return errs.InvalidArgument(fmt.Sprintf("Peer list cannot exceed %d items.", ChannelMaxPeers))
	}

	userID, err := fields.ParseRequiredID("user_id", rawUserID)
	if err != nil {
		return err
	}

	memberIDs, err := fields.ParseIDs("user_id", rawMemberIDs)
	if err != nil {
		return err
	}

	peerIDs := make([]fields.ID, 0, len(memberIDs))
	seen := map[fields.ID]struct{}{
		userID: {},
	}

	for _, id := range memberIDs {
		if _, exists := seen[id]; !exists {
			seen[id] = struct{}{}
			peerIDs = append(peerIDs, id)
		}
	}

	if len(peerIDs) > 0 {
		hasBlock, err := s.relationRepo.HasIncomingBlock(ctx, userID, peerIDs)
		if err != nil {
			return err
		}
		if hasBlock {
			return errs.InvalidArgument("Cannot create group DM with users who have blocked you.")
		}
	}

	allMemberIDs := make([]fields.ID, 0, len(peerIDs)+1)
	allMemberIDs = append(allMemberIDs, userID)
	allMemberIDs = append(allMemberIDs, peerIDs...)

	now := fields.NewTimestamp(time.Now())
	channelID := fields.NewID(uuid.New())

	parsedChannel := ParseChannel(
		channelID,
		NewChannelType(ChannelTypeGroup),
		ChannelName{},
		fields.URL{},
		fields.ID{},
		fields.Timestamp{},
		now,
		now,
	)

	parsedMembers := make([]*Member, 0, len(allMemberIDs))
	for _, id := range allMemberIDs {
		m := ParseMember(
			channelID,
			id,
			fields.ID{},
			fields.Timestamp{},
			fields.Timestamp{},
			fields.Timestamp{},
			1,
			true,
			now,
			now,
		)
		parsedMembers = append(parsedMembers, m)
	}

	var newChannel *Channel
	var newChannelMembers []*Member

	err = s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		channelRow, err := s.repo.Create(txCtx, parsedChannel)
		if err != nil {
			return err
		}

		memberRows, err := s.memberRepo.CreateBatch(txCtx, parsedMembers)
		if err != nil {
			return err
		}

		_, err = s.outboxRepo.Publish(txCtx, EventChannelCreated, ChannelCreatedPayload{})
		if err != nil {
			return err
		}

		newChannel = channelRow
		newChannelMembers = memberRows
		return nil
	})
	if err != nil {
		return err
	}

	s.cache.Set(ctx, newChannel)
	s.memberCache.SetBatch(ctx, newChannelMembers)

	// Set channelID: memberID[]?

	return nil
}

func (s *ChannelService) Get(ctx context.Context, rawUserID, rawChannelID uuid.UUID) (*Channel, []MemberView, []MessageView, error) {
	userID, err := fields.ParseRequiredID("user_id", rawUserID)
	if err != nil {
		return nil, nil, nil, err
	}

	channelID, err := fields.ParseRequiredID("channel_id", rawChannelID)
	if err != nil {
		return nil, nil, nil, err
	}

	// 1. Members (Required first for authorization & message anchor ID)
	memberMap, err := s.memberRepo.GetBatchByChannelIDs(ctx, []fields.ID{channelID})
	if err != nil {
		return nil, nil, nil, err
	}

	members := memberMap[channelID]

	var currentMember *Member
	for _, m := range members {
		if m.UserID() == userID {
			currentMember = m
			break
		}
	}

	if currentMember == nil {
		return nil, nil, nil, errs.PermissionDenied("You are not a member of this channel.")
	}

	// --- PHASE 1: Concurrent Channel + Messages fetch ---
	var (
		ch       *Channel
		messages []*Message
	)

	g1, ctx1 := errgroup.WithContext(ctx)

	// Fetch Channel
	g1.Go(func() error {
		channelMap, err := s.repo.GetBatch(ctx1, []fields.ID{channelID})
		if err != nil {
			return err
		}
		var ok bool
		ch, ok = channelMap[channelID]
		if !ok || ch == nil {
			return errs.NotFound("Channel not found.")
		}
		return nil
	})

	// Fetch Messages
	g1.Go(func() error {
		anchorMessageID := currentMember.LastReadMessageID()
		beforeLimit := MessageListBeforeLimit
		afterLimit := MessageListAfterLimit

		// Default anchor fallback logic
		if !anchorMessageID.IsValid() {
			beforeLimit = MessageListLimit
			afterLimit = 0
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

	// Collect known User IDs from Members & Messages
	userIDMap := make(map[fields.ID]struct{}, len(members)+len(messages))
	for _, m := range members {
		if id := m.UserID(); id.IsValid() {
			userIDMap[id] = struct{}{}
		}
	}
	for _, msg := range messages {
		if id := msg.AuthorID(); id.IsValid() {
			userIDMap[id] = struct{}{}
		}
	}

	userIDs := make([]fields.ID, 0, len(userIDMap))
	for id := range userIDMap {
		userIDs = append(userIDs, id)
	}

	// --- PHASE 2: Concurrent Reactions, Users, & Presence fetches ---
	var (
		reactionMap map[fields.ID][]*Reaction
		userMap     map[fields.ID]*user.User
		presenceMap map[fields.ID]presence.Presence
	)

	g2, ctx2 := errgroup.WithContext(ctx)

	// Fetch Reactions
	g2.Go(func() error {
		if len(messages) == 0 {
			reactionMap = make(map[fields.ID][]*Reaction)
			return nil
		}
		messageIDs := make([]fields.ID, len(messages))
		for i, msg := range messages {
			messageIDs[i] = msg.ID()
		}
		var err error
		reactionMap, err = s.reactionRepo.GetBatchByMessageIDs(ctx2, messageIDs)
		return err
	})

	// Fetch Users
	g2.Go(func() error {
		if len(userIDs) == 0 {
			userMap = make(map[fields.ID]*user.User)
			return nil
		}
		var err error
		userMap, err = s.userRepo.GetBatch(ctx2, userIDs)
		return err
	})

	// Fetch Presences (Redis/Cache)
	g2.Go(func() error {
		if len(userIDs) == 0 {
			presenceMap = make(map[fields.ID]presence.Presence)
			return nil
		}
		var err error
		presenceMap, err = s.presenceCache.GetBatch(ctx2, userIDs)
		return err
	})

	if err := g2.Wait(); err != nil {
		return nil, nil, nil, err
	}

	// Hydrate Views
	memberViews := HydrateMembers(members, userMap, presenceMap)
	messageViews := HydrateMessages(messages, reactionMap, userMap, userID)

	// ==========================================
	// CACHING ARCHITECTURE CHECKLIST FOR Get()
	// ==========================================
	// 1. Channel Cache:
	//    - Key: channel:{channel_id}
	//    - Implementation: s.channelCache.GetBatch / SetBatch (stores single channel entity state).
	//
	// 2. Member Cache:
	//    - Keys: member:{channel_id}:{user_id} (individual member DTOs) & channel:{channel_id}:members (Redis Set index)
	//    - Implementation: s.memberCache.GetByChannelID (checks the index set, handles cache hits/misses, and falls back to s.memberRepo.GetBatchByChannelIDs).
	//
	// 3. Message Cache (Two-Tier Approach):
	//    - Individual DTOs: message:{message_id} (via MessageCache.GetBatch / SetBatch)
	//    - Feed Index: channel:{channel_id}:messages (Redis ZSET ordered by creation timestamp via MessageCache.ListAround / SetBatchChannelIDs)
	//    - Implementation Flow on Cache Miss: Fall back to s.messageRepo.ListAroundByChannelID, then populate both the individual message keys and the channel ZSET feed index.
	//    - Add reactions to the cache to prevent needing its own keys.
	//
	// 4. Presence Cache:
	//    - Keys: presence:{user_id}
	//    - Implementation: Already wired to s.presenceCache.GetBatch in Phase 2.
	// ==========================================

	return ch, memberViews, messageViews, nil
}

func HydrateMembers(
	members []*Member,
	userMap map[fields.ID]*user.User,
	presenceMap map[fields.ID]presence.Presence,
) []MemberView {
	views := make([]MemberView, 0, len(members))
	for _, m := range members {
		u, ok := userMap[m.UserID()]
		if !ok || u == nil {
			continue
		}

		p, ok := presenceMap[m.UserID()]
		if !ok {
			p = presence.New(presence.PresenceOffline)
		}

		views = append(views, MemberView{
			id:          m.UserID(),
			displayName: u.DisplayName(),
			avatarURL:   u.AvatarURL(),
			presence:    p,
		})
	}
	return views
}

func HydrateMessage(
	msg *Message,
	reactions []*Reaction,
	author *user.User,
	currentUserID fields.ID,
) MessageView {
	if author == nil {
		author = &user.User{}
	}

	emojiCounts := make(map[ReactionEmoji][]*Reaction)
	for _, r := range reactions {
		emojiCounts[r.Emoji()] = append(emojiCounts[r.Emoji()], r)
	}

	reactionsView := make([]ReactionView, 0, len(emojiCounts))
	for emoji, list := range emojiCounts {
		isReacted := false
		for _, r := range list {
			if r.UserID() == currentUserID {
				isReacted = true
				break
			}
		}
		reactionsView = append(reactionsView, ReactionView{
			emoji:     emoji,
			count:     len(list),
			isReacted: isReacted,
		})
	}

	sort.Slice(reactionsView, func(i, j int) bool {
		return reactionsView[i].emoji.String() < reactionsView[j].emoji.String()
	})

	return MessageView{
		id:                 msg.ID(),
		authorID:           msg.AuthorID(),
		displayName:        author.DisplayName(),
		avatarURL:          author.AvatarURL(),
		msgType:            msg.Type(),
		content:            msg.Content(),
		systemMetadata:     msg.SystemMetadata(),
		replyToMessageID:   msg.ReplyToMessageID(),
		forwardedMessageID: msg.ForwardedMessageID(),
		forwardedChannelID: msg.ForwardedChannelID(),
		createdAt:          msg.CreatedAt(),
		editedAt:           msg.EditedAt(),
		reactions:          reactionsView,
	}
}

func HydrateMessages(
	messages []*Message,
	reactionMap map[fields.ID][]*Reaction,
	userMap map[fields.ID]*user.User,
	currentUserID fields.ID,
) []MessageView {
	views := make([]MessageView, 0, len(messages))
	for _, msg := range messages {
		views = append(views, HydrateMessage(
			msg,
			reactionMap[msg.ID()],
			userMap[msg.AuthorID()],
			currentUserID,
		))
	}
	return views
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
