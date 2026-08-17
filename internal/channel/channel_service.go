package channel

import (
	"bonfire-api/internal/errs"
	"bonfire-api/internal/fields"
	"bonfire-api/internal/pkg/ptr"
	"bonfire-api/internal/presence"
	"bonfire-api/internal/user"
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"
)

type ChannelRepository interface {
	Create(ctx context.Context, ch *Channel) (*Channel, error)
	Delete(ctx context.Context, id fields.ID) error
	Get(ctx context.Context, id fields.ID) (*Channel, error)
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
		return errs.InvalidArgument(fmt.Sprintf("Member list cannot exceed %d items.", ChannelMaxPeers))
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

		_, err = s.outboxRepo.Publish(txCtx, EventChannelCreated, ChannelCreated{})
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
	memberViews := s.hydrateMembers(members, userMap, presenceMap)
	messageViews := s.hydrateMessages(messages, reactionMap, userMap, userID)

	return ch, memberViews, messageViews, nil
}

func (s *ChannelService) hydrateMembers(
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

func (s *ChannelService) hydrateMessages(
	messages []*Message,
	reactionMap map[fields.ID][]*Reaction,
	userMap map[fields.ID]*user.User,
	currentUserID fields.ID,
) []MessageView {
	views := make([]MessageView, 0, len(messages))
	for _, msg := range messages {
		u, ok := userMap[msg.AuthorID()]
		if !ok || u == nil {
			u = &user.User{}
		}

		rxList := reactionMap[msg.ID()]
		emojiCounts := make(map[ReactionEmoji][]*Reaction)
		for _, r := range rxList {
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

		views = append(views, MessageView{
			id:                 msg.ID(),
			authorID:           msg.AuthorID(),
			displayName:        u.DisplayName(),
			avatarURL:          u.AvatarURL(),
			msgType:            msg.Type(),
			content:            msg.Content(),
			systemMetadata:     msg.SystemMetadata(),
			replyToMessageID:   msg.ReplyToMessageID(),
			forwardedMessageID: msg.ForwardedMessageID(),
			forwardedChannelID: msg.ForwardedChannelID(),
			createdAt:          msg.CreatedAt(),
			editedAt:           msg.EditedAt(),
			reactions:          reactionsView,
		})
	}
	return views
}

// Get User Sidebar

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

		updatedChannel = channelRow
		return nil
	})
	if err != nil {
		return nil, err
	}

	s.cache.Delete(ctx, updatedChannel.ID())

	return updatedChannel, nil
}
