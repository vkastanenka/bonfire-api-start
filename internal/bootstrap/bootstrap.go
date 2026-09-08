package bootstrap

import (
	"bonfire-api/internal/channel"
	"bonfire-api/internal/fields"
	"context"

	"github.com/google/uuid"
)

type Service struct {
	channelRepo       ChannelRepository
	cachedChannelRepo CachedChannelRepository
	memberRepo        MemberRepository
	cachedMemberRepo  CachedMemberRepository
	messageRepo       MessageRepository
	cachedMessageRepo CachedMessageRepository
	reactionRepo      ReactionRepository
	outboxRepo        OutboxRepository
	relationRepo      RelationRepository
	userRepo          UserRepository
	cachedUserRepo    CachedUserRepository
	tx                TX
}

func NewService(
	channelRepo ChannelRepository,
	cachedChannelRepo CachedChannelRepository,
	memberRepo MemberRepository,
	cachedMemberRepo CachedMemberRepository,
	messageRepo MessageRepository,
	cachedMessageRepo CachedMessageRepository,
	reactionRepo ReactionRepository,
	outboxRepo OutboxRepository,
	relationRepo RelationRepository,
	userRepo UserRepository,
	cachedUserRepo CachedUserRepository,
	tx TX,
) *Service {
	return &Service{
		channelRepo:       channelRepo,
		cachedChannelRepo: cachedChannelRepo,
		memberRepo:        memberRepo,
		cachedMemberRepo:  cachedMemberRepo,
		messageRepo:       messageRepo,
		cachedMessageRepo: cachedMessageRepo,
		reactionRepo:      reactionRepo,
		outboxRepo:        outboxRepo,
		relationRepo:      relationRepo,
		userRepo:          userRepo,
		cachedUserRepo:    cachedUserRepo,
		tx:                tx,
	}
}

// type BootstrapResult struct {
// 	Channel       *Channel
// 	Member        *Member
// 	Messages      []*Message
// 	Reactions     map[fields.ID]*ReactionSummary
// 	Users         map[fields.ID]*user.User
// 	Presences     map[fields.ID]presence.Presence
// 	MemberIDs     []fields.ID
// 	HasMoreBefore bool
// 	HasMoreAfter  bool
// }

// Bootstrap fetches all channel data needed to load a channel, including details, members, and messages.
func (s *Service) Bootstrap(ctx context.Context, rawUserID, rawChannelID, rawMessageID uuid.UUID) (*any, error) {
	userID, err := fields.ParseRequiredID("", rawUserID)
	if err != nil {
		return nil, err
	}

	// user, err := s.cachedUserRepo.Get(ctx, userID)

	userMemberships, err := s.memberRepo.ListVisibleByUserID(ctx, userID, 100)
	if err != nil {
		return nil, err
	}

	// if len(userMemberships) == 0 {}

	channelIDs := getMemberChannelIDs(userMemberships)

	channels, err := s.cachedChannelRepo.GetBatch(ctx, channelIDs)
	if err != nil {
		return nil, err
	}

	channelMembersMap, err := s.cachedMemberRepo.GetBatchByChannelIDs(ctx, channelIDs)
	if err != nil {
		return nil, err
	}

	// actorID, channelID, err := validateIDs(rawActorID, rawChannelID)
	// if err != nil {
	// 	return nil, err
	// }

	// messageID, err := fields.ParseID(rawMessageID)
	// if err != nil {
	// 	return nil, err
	// }

	// members, err := s.memberRepo.GetBatchByChannelID(ctx, channelID)
	// if err != nil {
	// 	return nil, err
	// }

	// actorMember, err := validateMembership(actorID, members)
	// if err != nil {
	// 	return nil, err
	// }

	// memberIDs := getMemberIDs(members)

	// var (
	// 	channel       *Channel
	// 	messages      []*Message
	// 	hasMoreBefore bool
	// 	hasMoreAfter  bool
	// )

	// g1, g1Ctx := errgroup.WithContext(ctx)

	// g1.Go(func() error {
	// 	var getErr error
	// 	channel, getErr = s.Get(g1Ctx, channelID.UUID())
	// 	return getErr
	// })

	// g1.Go(func() error {
	// 	cursor := getMessagesCursor(actorMember.LastReadMessageID(), messageID)

	// 	var listErr error
	// 	messages, hasMoreBefore, hasMoreAfter, listErr = s.messageRepo.ListAroundByChannelID(
	// 		g1Ctx,
	// 		channelID,
	// 		cursor.ID(),
	// 		cursor.BeforeLimit(),
	// 		cursor.AfterLimit(),
	// 	)
	// 	return listErr
	// })

	// if err := g1.Wait(); err != nil {
	// 	return nil, err
	// }

	// allUserIDs := getChannelUserIDs(memberIDs, messages)

	// var (
	// 	users     map[fields.ID]*user.User
	// 	presences map[fields.ID]presence.Presence
	// 	reactions map[fields.ID]*ReactionSummary
	// )

	// g2, g2Ctx := errgroup.WithContext(ctx)

	// g2.Go(func() error {
	// 	var fetchErr error
	// 	users, fetchErr = s.userService.GetBatch(g2Ctx, allUserIDs)
	// 	return fetchErr
	// })

	// g2.Go(func() error {
	// 	var fetchErr error
	// 	presences, fetchErr = s.presenceCache.GetBatchPresence(g2Ctx, memberIDs)
	// 	return fetchErr
	// })

	// g2.Go(func() error {
	// 	if len(messages) == 0 {
	// 		reactions = make(map[fields.ID]*ReactionSummary)
	// 		return nil
	// 	}

	// 	messageIDs, _ := getMessageIDs(messages)
	// 	var fetchErr error
	// 	reactions, fetchErr = s.reactionRepo.GetBatchSummaryByMessageIDs(g2Ctx, actorID, messageIDs)
	// 	return fetchErr
	// })

	// if err := g2.Wait(); err != nil {
	// 	return nil, err
	// }

	// sortMemberIDs(memberIDs, users)

	// return &BootstrapResult{
	// 	Channel:       channel,
	// 	Member:        actorMember,
	// 	Messages:      messages,
	// 	Reactions:     reactions,
	// 	Users:         users,
	// 	Presences:     presences,
	// 	MemberIDs:     memberIDs,
	// 	HasMoreBefore: hasMoreBefore,
	// 	HasMoreAfter:  hasMoreAfter,
	// }, nil
}

// // GetSidebar fetches all sidebar related structures.
// func (s *ChannelService) GetSidebar(ctx context.Context, rawActorID uuid.UUID) (
// 	channelMap map[fields.ID]*Channel,
// 	memberMap map[fields.ID]*Member,
// 	peerIDsMap map[fields.ID][]fields.ID,
// 	channelIDs []fields.ID,
// 	peerIDs []fields.ID,
// 	err error,
// ) {
// 	actorID, err := fields.ParseRequiredID("actor_id", rawActorID)
// 	if err != nil {
// 		return nil, nil, nil, nil, nil, err
// 	}

// 	userMemberships, err := s.memberRepo.ListVisibleByUserID(ctx, actorID, ChannelMaxSidebarItems)
// 	if err != nil {
// 		return nil, nil, nil, nil, nil, err
// 	}

// 	if len(userMemberships) == 0 {
// 		return make(map[fields.ID]*Channel), make(map[fields.ID]*Member), make(map[fields.ID][]fields.ID), []fields.ID{}, []fields.ID{}, nil
// 	}

// 	channelIDs, memberMap = indexMemberships(userMemberships)

// 	channelMap, err = s.repo.GetBatch(ctx, channelIDs)
// 	if err != nil {
// 		return nil, nil, nil, nil, nil, err
// 	}

// 	channelMembersMap, err := s.memberRepo.GetBatchByChannelIDs(ctx, channelIDs)
// 	if err != nil {
// 		return nil, nil, nil, nil, nil, err
// 	}

// 	channels := getChannels(channelMap)
// 	sortSidebar(channels, memberMap)
// 	channelIDs = indexChannels(channels)
// 	peerIDs, _ = getSidebarUserIDs(actorID, channelMap, channelMembersMap)
// 	peerIDsMap = getSidebarPeerIDsMap(actorID, channelMembersMap)
// 	return channelMap, memberMap, peerIDsMap, channelIDs, peerIDs, nil
// }

// type UserGetMeResponse struct {
// 	Me             user.UserMeView                `json:"me"`
// 	Users          map[fields.ID]*user.UserView   `json:"users"`
// 	Presences      map[fields.ID]user.Presence    `json:"presences"`
// 	Channels       map[fields.ID]*channel.Channel `json:"channels"`
// 	Members        map[fields.ID]*channel.Member  `json:"members"`
// 	PeerIDs        map[fields.ID][]fields.ID      `json:"peerIDs"`
// 	FriendChannels map[fields.ID]fields.ID        `json:"friendChannels"` // friendID -> channelID
// 	ChannelIDs     []fields.ID                    `json:"channelIds"`     // ordered sidebar channels
// 	FriendIDs      []fields.ID                    `json:"friendIds"`      // sorted friend IDs
// }

// func (h *UserHandler) GetMe(w http.ResponseWriter, r *http.Request) error {
// 	ctx := r.Context()
// 	userID, err := httpio.CtxGetUserID(ctx)
// 	if err != nil {
// 		return err
// 	}

// 	g1, gCtx1 := errgroup.WithContext(ctx)

// 	var (
// 		me                *user.User
// 		channelMap        map[fields.ID]*channel.Channel
// 		memberMap         map[fields.ID]*channel.Member
// 		peerIDsMap        map[fields.ID][]fields.ID
// 		channelIDs        []fields.ID
// 		peerIDs           []fields.ID
// 		friendChannelsMap map[fields.ID]fields.ID
// 		friendIDs         []fields.ID
// 	)

// 	g1.Go(func() error {
// 		var err error
// 		me, err = h.service.Get(gCtx1, userID.UUID())
// 		return err
// 	})

// 	g1.Go(func() error {
// 		var err error
// 		channelMap, memberMap, peerIDsMap, channelIDs, peerIDs, _, err = h.chanService.GetSidebar(gCtx1, userID.UUID())
// 		return err
// 	})

// 	g1.Go(func() error {
// 		var err error
// 		friendChannelsMap, friendIDs, err = h.relService.GetPeers(gCtx1, userID.UUID(), relation.NewTypeFriends().String())
// 		return err
// 	})

// 	if err := g1.Wait(); err != nil {
// 		return err
// 	}

// 	allUserIDs := make([]fields.ID, 0, len(peerIDs)+len(friendIDs)+1)
// 	allUserIDs = append(allUserIDs, userID)
// 	allUserIDs = append(allUserIDs, peerIDs...)
// 	allUserIDs = append(allUserIDs, friendIDs...)
// 	dedupedUserIDs := fields.DedupeIDs(allUserIDs)

// 	g2, gCtx2 := errgroup.WithContext(ctx)

// 	var (
// 		usersMap     map[fields.ID]*user.User
// 		presencesMap map[fields.ID]user.Presence
// 	)

// 	g2.Go(func() error {
// 		var err error
// 		usersMap, err = h.service.GetBatch(gCtx2, dedupedUserIDs)
// 		return err
// 	})

// 	g2.Go(func() error {
// 		var err error
// 		presencesMap, err = h.service.GetBatchPresence(gCtx2, dedupedUserIDs)
// 		return err
// 	})

// 	if err := g2.Wait(); err != nil {
// 		return err
// 	}

// 	relation.SortFriendIDs(friendIDs, usersMap)

// 	userViews := make(map[fields.ID]*user.UserView, len(usersMap))
// 	for id, u := range usersMap {
// 		if u == nil {
// 			continue
// 		}
// 		p, exists := presencesMap[id]
// 		if !exists {
// 			p = user.NewPresenceOffline()
// 		}
// 		view := user.ToUserView(u, p, fields.Now())
// 		userViews[id] = &view
// 	}

// 	response := UserGetMeResponse{
// 		Me:             user.ToUserMeView(me),
// 		Users:          userViews,
// 		Presences:      presencesMap,
// 		Channels:       channelMap,
// 		Members:        memberMap,
// 		PeerIDs:        peerIDsMap,
// 		FriendChannels: friendChannelsMap,
// 		ChannelIDs:     channelIDs,
// 		FriendIDs:      friendIDs,
// 	}

// 	httpio.RespondOK(w, r, response)
// 	return nil
// }

// func getMessagesCursor(
// 	actorLastReadID fields.ID,
// 	fallbackMessageID fields.ID,
// ) fields.Cursor {
// 	if fallbackMessageID.IsValid() {
// 		return fields.NewCursor(fallbackMessageID, MessageListBeforeLimit, MessageListAfterLimit)
// 	}

// 	if actorLastReadID.IsValid() {
// 		return fields.NewCursor(actorLastReadID, MessageListBeforeLimit, MessageListAfterLimit)
// 	}

// 	return fields.NewCursor(fallbackMessageID, MessageListLimit, 0)
// }

// func getSidebarPeerIDsMap(actorID fields.ID, memberMap map[fields.ID][]*Member) map[fields.ID][]fields.ID {
// 	channelPeerIDsMap := make(map[fields.ID][]fields.ID, len(memberMap))
// 	for chID, members := range memberMap {
// 		var userIDs []fields.ID
// 		for _, m := range members {
// 			if m != nil && !m.UserID().Equals(actorID) {
// 				userIDs = append(userIDs, m.UserID())
// 			}
// 		}

// 		slices.SortFunc(userIDs, func(a, b fields.ID) int {
// 			return a.Compare(b)
// 		})

// 		channelPeerIDsMap[chID] = userIDs
// 	}
// 	return channelPeerIDsMap
// }

// func getSidebarUserIDs(
// 	actorID fields.ID,
// 	channelMap map[fields.ID]*Channel,
// 	memberMap map[fields.ID][]*Member,
// ) (peerIDs, directPeerIDs []fields.ID) {
// 	for chID, members := range memberMap {
// 		ch := channelMap[chID]
// 		if ch == nil {
// 			continue
// 		}

// 		isDirect := ch.Type().IsDirect()
// 		for _, m := range members {
// 			userID := m.UserID()
// 			if userID.Equals(actorID) {
// 				continue
// 			}

// 			peerIDs = append(peerIDs, userID)
// 			if isDirect {
// 				directPeerIDs = append(directPeerIDs, userID)
// 			}
// 		}
// 	}

// 	return fields.DedupeIDs(peerIDs), fields.DedupeIDs(directPeerIDs)
// }

func getMemberChannelIDs(members []*channel.Member) []fields.ID {
	channelIDs := make([]fields.ID, 0, len(members))

	for _, m := range members {
		if m == nil {
			continue
		}
		channelIDs = append(channelIDs, m.ChannelID())
	}

	return channelIDs
}
