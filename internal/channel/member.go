package channel

import (
	"cmp"
	"slices"
	"strings"
	"time"

	"bonfire-api/internal/user"

	"github.com/google/uuid"
)

type Member struct {
	ChannelID         uuid.UUID
	UserID            uuid.UUID
	LastReadMessageID *uuid.UUID
	LastReadMessageAt *time.Time
	PinnedAt          *time.Time
	MutedUntil        *time.Time
	MentionCount      int
	IsVisible         bool
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

func ReconstituteMember(
	channelID uuid.UUID,
	userID uuid.UUID,
	lastReadMessageID *uuid.UUID,
	lastReadMessageAt *time.Time,
	pinnedAt *time.Time,
	mutedUntil *time.Time,
	mentionCount int,
	isVisible bool,
	createdAt time.Time,
	updatedAt time.Time,
) *Member {
	return &Member{
		ChannelID:         channelID,
		UserID:            userID,
		LastReadMessageID: lastReadMessageID,
		LastReadMessageAt: lastReadMessageAt,
		PinnedAt:          pinnedAt,
		MutedUntil:        mutedUntil,
		MentionCount:      mentionCount,
		IsVisible:         isVisible,
		CreatedAt:         createdAt,
		UpdatedAt:         updatedAt,
	}
}

func NewMember(
	channelID uuid.UUID,
	userID uuid.UUID,
	mentionCount int,
	now time.Time,
) *Member {
	return ReconstituteMember(
		channelID,
		userID,
		nil,
		nil,
		nil,
		nil,
		mentionCount,
		true,
		now,
		now,
	)
}

func NewCreator(
	channelID uuid.UUID,
	userID uuid.UUID,
	now time.Time,
) *Member {
	return NewMember(channelID, userID, 0, now)
}

func NewPeer(
	channelID uuid.UUID,
	userID uuid.UUID,
	now time.Time,
) *Member {
	return NewMember(channelID, userID, 1, now)
}

func NewPeers(
	channelID uuid.UUID,
	userIDs []uuid.UUID,
	now time.Time,
) []*Member {
	peers := make([]*Member, 0, len(userIDs))
	for _, userID := range userIDs {
		peers = append(peers, NewPeer(channelID, userID, now))
	}
	return peers
}

func NewMembers(
	channelID uuid.UUID,
	creatorID uuid.UUID,
	peerIDs []uuid.UUID,
	now time.Time,
) []*Member {
	members := make([]*Member, 0, len(peerIDs)+1)
	members = append(members, NewCreator(channelID, creatorID, now))

	for _, peerID := range peerIDs {
		members = append(members, NewPeer(channelID, peerID, now))
	}

	return members
}

func getChannels(channelMap map[uuid.UUID]*Channel) []*Channel {
	channels := make([]*Channel, 0, len(channelMap))
	for _, ch := range channelMap {
		if ch != nil {
			channels = append(channels, ch)
		}
	}
	return channels
}

func filterMembership(actorID uuid.UUID, membs []*Member) *Member {
	for _, m := range membs {
		if m != nil && m.UserID == actorID {
			return m
		}
	}
	return nil
}

func filterPeerIDs(actorID uuid.UUID, parsedPeerIDs []uuid.UUID) []uuid.UUID {
	peerIDs := make([]uuid.UUID, 0, len(parsedPeerIDs))
	seen := make(map[uuid.UUID]struct{}, len(parsedPeerIDs))

	for _, id := range parsedPeerIDs {
		if id == actorID || id == (uuid.UUID{}) {
			continue
		}
		if _, exists := seen[id]; !exists {
			seen[id] = struct{}{}
			peerIDs = append(peerIDs, id)
		}
	}

	return peerIDs
}

func filterRequiredPeerIDs(actorID uuid.UUID, parsedPeerIDs []uuid.UUID) ([]uuid.UUID, error) {
	peerIDs := filterPeerIDs(actorID, parsedPeerIDs)
	if len(peerIDs) == 0 {
		return nil, ErrNoNewMembers()
	}
	return peerIDs, nil
}

func getMemberIDs(members []*Member) []uuid.UUID {
	ids := make([]uuid.UUID, 0, len(members))
	seen := make(map[uuid.UUID]struct{}, len(members))

	for _, m := range members {
		if m != nil && m.UserID != (uuid.UUID{}) {
			if _, exists := seen[m.UserID]; !exists {
				seen[m.UserID] = struct{}{}
				ids = append(ids, m.UserID)
			}
		}
	}

	return ids
}

func indexMemberships(members []*Member) ([]uuid.UUID, map[uuid.UUID]*Member) {
	channelIDs := make([]uuid.UUID, 0, len(members))
	membershipMap := make(map[uuid.UUID]*Member, len(members))

	for _, m := range members {
		if m == nil {
			continue
		}
		chID := m.ChannelID
		channelIDs = append(channelIDs, chID)
		membershipMap[chID] = m
	}

	return channelIDs, membershipMap
}

func sortMembers(members []*Member, userMap map[uuid.UUID]*user.User) {
	displayName := func(m *Member) string {
		if m == nil {
			return ""
		}
		if u := userMap[m.UserID]; u != nil {
			return u.DisplayName
		}
		return ""
	}

	slices.SortFunc(members, func(a, b *Member) int {
		if cmpVal := cmp.Compare(displayName(a), displayName(b)); cmpVal != 0 {
			return cmpVal
		}

		var idA, idB string
		if a != nil {
			idA = a.UserID.String()
		}
		if b != nil {
			idB = b.UserID.String()
		}
		return strings.Compare(idA, idB)
	})
}

func sortMemberIDs(memberIDs []uuid.UUID, users map[uuid.UUID]*user.User) {
	slices.SortFunc(memberIDs, func(a, b uuid.UUID) int {
		uA, okA := users[a]
		uB, okB := users[b]

		if !okA && !okB {
			return 0
		}
		if !okA {
			return 1
		}
		if !okB {
			return -1
		}

		return strings.Compare(uA.DisplayName, uB.DisplayName)
	})
}

func validateMembership(userID uuid.UUID, members []*Member) (*Member, error) {
	for _, m := range members {
		if m != nil && m.UserID == userID {
			return m, nil
		}
	}
	return nil, ErrNotChannelMember()
}
