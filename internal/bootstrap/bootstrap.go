package bootstrap

import (
	"bytes"
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"bonfire-api/internal/channel"
	"bonfire-api/internal/presence"
	"bonfire-api/internal/user"

	"github.com/google/uuid"
)

type Result struct {
	User                  *user.User                      `json:"user"`
	Friends               map[uuid.UUID]uuid.UUID         `json:"friends"`
	UserMembers           map[uuid.UUID]*channel.Member   `json:"userMembers"`
	Channels              map[uuid.UUID]*channel.Channel  `json:"channels"`
	ChannelPeerIDs        map[uuid.UUID][]uuid.UUID       `json:"channelPeerIDs"`
	ChannelMemberIDs      map[uuid.UUID][]uuid.UUID       `json:"channelMemberIDs"`
	FriendIDs             []uuid.UUID                     `json:"friendIDs"`
	PendingIDs            []uuid.UUID                     `json:"pendingIDs"`
	SidebarIDs            []uuid.UUID                     `json:"sidebarIDs"`
	Peers                 map[uuid.UUID]*user.User        `json:"peers"`
	PeerPresences         map[uuid.UUID]presence.Presence `json:"peerPresences"`
	Messages              []*channel.Message              `json:"messages"`
	HasMoreMessagesBefore bool                            `json:"hasMoreMessagesBefore"`
	HasMoreMessagesAfter  bool                            `json:"hasMoreMessagesAfter"`
}

type Service struct {
	channelRepo        ChannelRepository
	cachedChannelRepo  CachedChannelRepository
	memberRepo         MemberRepository
	cachedMemberRepo   CachedMemberRepository
	messageRepo        MessageRepository
	cachedMessageRepo  CachedMessageRepository
	reactionRepo       ReactionRepository
	outboxRepo         OutboxRepository
	presenceCache      PresenceCache
	relationRepo       RelationRepository
	cachedRelationRepo CachedRelationRepository
	userRepo           UserRepository
	cachedUserRepo     CachedUserRepository
	tx                 TX
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
	presenceCache PresenceCache,
	relationRepo RelationRepository,
	cachedRelationRepo CachedRelationRepository,
	userRepo UserRepository,
	cachedUserRepo CachedUserRepository,
	tx TX,
) *Service {
	return &Service{
		channelRepo:        channelRepo,
		cachedChannelRepo:  cachedChannelRepo,
		memberRepo:         memberRepo,
		cachedMemberRepo:   cachedMemberRepo,
		messageRepo:        messageRepo,
		cachedMessageRepo:  cachedMessageRepo,
		reactionRepo:       reactionRepo,
		outboxRepo:         outboxRepo,
		presenceCache:      presenceCache,
		relationRepo:       relationRepo,
		cachedRelationRepo: cachedRelationRepo,
		userRepo:           userRepo,
		cachedUserRepo:     cachedUserRepo,
		tx:                 tx,
	}
}

// Bootstrap fetches all channel data needed to load a channel, including details, members, and messages.
func (s *Service) Bootstrap(ctx context.Context, userID uuid.UUID, channelID, messageID *uuid.UUID) (*Result, error) {
	user, err := s.cachedUserRepo.Get(ctx, userID)
	if err != nil {
		return nil, err
	}

	friends, err := s.cachedRelationRepo.GetFriends(ctx, userID)
	if err != nil {
		return nil, err
	}

	friendIDs := make([]uuid.UUID, 0, len(friends))
	seenPeers := make(map[uuid.UUID]struct{}, len(friends))

	for friendID := range friends {
		friendIDs = append(friendIDs, friendID)
		seenPeers[friendID] = struct{}{}
	}

	var (
		pendingIDs []uuid.UUID
		pendingErr error
	)

	pendingIDs, pendingErr = s.cachedRelationRepo.GetPendingIDs(ctx, userID)
	if pendingErr != nil {
		return nil, fmt.Errorf("failed to fetch pending IDs: %w", pendingErr)
	}

	userMembers, err := s.cachedMemberRepo.ListVisibleByUserID(ctx, userID, 100)
	if err != nil {
		return nil, err
	}

	memberChannelIDs := getMemberChannelIDs(userMembers)
	channels, err := s.cachedChannelRepo.GetBatch(ctx, memberChannelIDs)
	if err != nil {
		return nil, err
	}

	channelMembersMap, err := s.cachedMemberRepo.GetBatchByChannelIDs(ctx, memberChannelIDs)
	if err != nil {
		return nil, err
	}

	channelPeerIDsMap := make(map[uuid.UUID][]uuid.UUID, len(channelMembersMap))
	channelMemberIDsMap := make(map[uuid.UUID][]uuid.UUID, len(channelMembersMap))

	for channelID, members := range channelMembersMap {
		peerIDs := make([]uuid.UUID, 0, len(members))
		memberIDs := make([]uuid.UUID, 0, len(members))

		for _, member := range members {
			mID := member.UserID
			memberIDs = append(memberIDs, mID)

			if mID == userID {
				continue
			}

			peerIDs = append(peerIDs, mID)
			seenPeers[mID] = struct{}{}
		}

		slices.SortFunc(peerIDs, func(a, b uuid.UUID) int {
			return bytes.Compare(a[:], b[:])
		})

		channelPeerIDsMap[channelID] = peerIDs
		channelMemberIDsMap[channelID] = memberIDs
	}

	// Extract all unique peer IDs (friends + channel peers)
	dedupedPeerIDs := make([]uuid.UUID, 0, len(seenPeers))
	for peerID := range seenPeers {
		dedupedPeerIDs = append(dedupedPeerIDs, peerID)
	}

	peers, err := s.cachedUserRepo.GetBatch(ctx, dedupedPeerIDs)
	if err != nil {
		return nil, err
	}

	peerPresences, err := s.presenceCache.GetBatchPresence(ctx, dedupedPeerIDs)
	if err != nil {
		return nil, err
	}

	getDisplayName := func(id uuid.UUID) string {
		if id == userID && user != nil {
			return user.DisplayName
		}
		if u, ok := peers[id]; ok {
			return u.DisplayName
		}
		return ""
	}

	for _, memberIDs := range channelMemberIDsMap {
		slices.SortFunc(memberIDs, func(a, b uuid.UUID) int {
			nameA := strings.ToLower(getDisplayName(a))
			nameB := strings.ToLower(getDisplayName(b))

			if nameA != nameB {
				return strings.Compare(nameA, nameB)
			}
			// Fallback tie-breaker by UUIDv7 ascending
			return bytes.Compare(a[:], b[:])
		})
	}

	// 1. Sort friendIDs by display name ascending (case-insensitive)
	slices.SortFunc(friendIDs, func(a, b uuid.UUID) int {
		nameA := strings.ToLower(getDisplayName(a))
		nameB := strings.ToLower(getDisplayName(b))

		if nameA != nameB {
			return strings.Compare(nameA, nameB)
		}
		// Fallback tie-breaker by UUIDv7 ascending
		return bytes.Compare(a[:], b[:])
	})

	// 2. Sort pendingIDs by UUIDv7 descending (newest first)
	slices.SortFunc(pendingIDs, func(a, b uuid.UUID) int {
		return bytes.Compare(b[:], a[:])
	})

	// 3. Map userMembers by ChannelID for O(1) lookup in sortSidebar
	userMembersMap := make(map[uuid.UUID]*channel.Member, len(userMembers))
	for _, m := range userMembers {
		userMembersMap[m.ChannelID] = m
	}

	// 4. Convert channels map to a slice for sorting
	sidebarChannels := make([]*channel.Channel, 0, len(channels))
	for _, c := range channels {
		sidebarChannels = append(sidebarChannels, c)
	}

	// 5. In-place sort channels according to sidebar criteria
	sortSidebar(sidebarChannels, userMembersMap)

	// 6. Extract sorted channel IDs
	sidebarIDs := make([]uuid.UUID, len(sidebarChannels))
	for i, c := range sidebarChannels {
		sidebarIDs[i] = c.ID
	}

	var (
		messages      []*channel.Message
		hasMoreBefore bool
		hasMoreAfter  bool
		messagesErr   error
	)

	if channelID != nil {
		userMember := userMembersMap[*channelID]
		var actorLastReadID uuid.UUID
		if userMember != nil {
			actorLastReadID = *userMember.LastReadMessageID
		}

		msgCursor, beforeLimit, afterLimit := getMessagesCursor(&actorLastReadID, messageID)

		messages, hasMoreBefore, hasMoreAfter, messagesErr = s.cachedMessageRepo.ListAroundByChannelID(
			ctx,
			*channelID,
			*msgCursor,
			beforeLimit,
			afterLimit,
		)
		if messagesErr != nil {
			return nil, fmt.Errorf("failed to fetch messages: %w", messagesErr)
		}
	}

	return &Result{
		User:                  user,
		Friends:               friends,
		UserMembers:           userMembersMap,
		Channels:              channels,
		ChannelPeerIDs:        channelPeerIDsMap,
		ChannelMemberIDs:      channelMemberIDsMap,
		FriendIDs:             friendIDs,
		PendingIDs:            pendingIDs,
		SidebarIDs:            sidebarIDs,
		Peers:                 peers,
		PeerPresences:         peerPresences,
		Messages:              messages,
		HasMoreMessagesBefore: hasMoreBefore,
		HasMoreMessagesAfter:  hasMoreAfter,
	}, nil
}

func sortSidebar(channels []*channel.Channel, userMembersMap map[uuid.UUID]*channel.Member) {
	pinnedAt := func(c *channel.Channel) time.Time {
		if c == nil {
			return time.Time{}
		}
		if m := userMembersMap[c.ID]; m != nil && m.PinnedAt != nil {
			return *m.PinnedAt
		}
		return time.Time{}
	}

	slices.SortFunc(channels, func(a, b *channel.Channel) int {
		pinA, pinB := pinnedAt(a), pinnedAt(b)
		isPinnedA, isPinnedB := !pinA.IsZero(), !pinB.IsZero()

		// Pinned channels come first
		if isPinnedA != isPinnedB {
			if isPinnedA {
				return -1
			}
			return 1
		}
		if isPinnedA && isPinnedB {
			if cmp := pinB.Compare(pinA); cmp != 0 {
				return cmp
			}
		}

		// Fallback for LastMessageAt if it's a *time.Time pointer
		var tA, tB time.Time
		if a != nil && a.LastMessageAt != nil {
			tA = *a.LastMessageAt
		}
		if b != nil && b.LastMessageAt != nil {
			tB = *b.LastMessageAt
		}
		if cmp := tB.Compare(tA); cmp != 0 {
			return cmp
		}

		// CreatedAt comparison
		var cA, cB time.Time
		if a != nil {
			cA = a.CreatedAt
		}
		if b != nil {
			cB = b.CreatedAt
		}
		if cmp := cB.Compare(cA); cmp != 0 {
			return cmp
		}

		// UUID comparison using byte slices
		var idA, idB uuid.UUID
		if a != nil {
			idA = a.ID
		}
		if b != nil {
			idB = b.ID
		}
		return bytes.Compare(idA[:], idB[:])
	})
}

func getMessagesCursor(
	actorLastReadID *uuid.UUID,
	fallbackMessageID *uuid.UUID,
) (*uuid.UUID, int, int) {
	if fallbackMessageID != nil {
		return fallbackMessageID, channel.MessageListBeforeLimit, channel.MessageListAfterLimit
	}

	if actorLastReadID != nil {
		return actorLastReadID, channel.MessageListBeforeLimit, channel.MessageListAfterLimit
	}

	return fallbackMessageID, channel.MessageListLimit, 0
}

func getMemberChannelIDs(members []*channel.Member) []uuid.UUID {
	channelIDs := make([]uuid.UUID, 0, len(members))

	for _, m := range members {
		if m == nil {
			continue
		}
		channelIDs = append(channelIDs, m.ChannelID)
	}

	return channelIDs
}
