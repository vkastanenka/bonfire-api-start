package channel

import (
	"bonfire-api/internal/fields"
	"bonfire-api/internal/user"
	"cmp"
	"slices"
)

type Member struct {
	channelID         fields.ID
	userID            fields.ID
	lastReadMessageID fields.ID
	lastReadMessageAt fields.Timestamp
	pinnedAt          fields.Timestamp
	mutedUntil        fields.Timestamp
	mentionCount      int
	isVisible         bool
	createdAt         fields.Timestamp
	updatedAt         fields.Timestamp
}

func ReconstituteMember(
	channelID fields.ID,
	userID fields.ID,
	lastReadMessageID fields.ID,
	lastReadMessageAt fields.Timestamp,
	pinnedAt fields.Timestamp,
	mutedUntil fields.Timestamp,
	mentionCount int,
	isVisible bool,
	createdAt fields.Timestamp,
	updatedAt fields.Timestamp,
) *Member {
	return &Member{
		channelID:         channelID,
		userID:            userID,
		lastReadMessageID: lastReadMessageID,
		lastReadMessageAt: lastReadMessageAt,
		pinnedAt:          pinnedAt,
		mutedUntil:        mutedUntil,
		mentionCount:      mentionCount,
		isVisible:         isVisible,
		createdAt:         createdAt,
		updatedAt:         updatedAt,
	}
}

func NewMember(
	channelID fields.ID,
	userID fields.ID,
	mentionCount int,
	now fields.Timestamp,
) *Member {
	return ReconstituteMember(
		channelID,
		userID,
		fields.ID{},
		fields.Timestamp{},
		fields.Timestamp{},
		fields.Timestamp{},
		mentionCount,
		true,
		now,
		now,
	)
}

func NewCreator(
	channelID fields.ID,
	userID fields.ID,
	now fields.Timestamp,
) *Member {
	return NewMember(channelID, userID, 0, now)
}

func NewPeer(
	channelID fields.ID,
	userID fields.ID,
	now fields.Timestamp,
) *Member {
	return NewMember(channelID, userID, 1, now)
}

func NewPeers(
	channelID fields.ID,
	userIDs []fields.ID,
	now fields.Timestamp,
) []*Member {
	peers := make([]*Member, 0, len(userIDs))
	for _, userID := range userIDs {
		peers = append(peers, NewPeer(channelID, userID, now))
	}
	return peers
}

func NewMembers(
	channelID fields.ID,
	creatorID fields.ID,
	peerIDs []fields.ID,
	now fields.Timestamp,
) []*Member {
	members := make([]*Member, 0, len(peerIDs)+1)
	members = append(members, NewCreator(channelID, creatorID, now))

	for _, peerID := range peerIDs {
		members = append(members, NewPeer(channelID, peerID, now))
	}

	return members
}

func (m *Member) ChannelID() fields.ID                { return m.channelID }
func (m *Member) UserID() fields.ID                   { return m.userID }
func (m *Member) LastReadMessageID() fields.ID        { return m.lastReadMessageID }
func (m *Member) LastReadMessageAt() fields.Timestamp { return m.lastReadMessageAt }
func (m *Member) PinnedAt() fields.Timestamp          { return m.pinnedAt }
func (m *Member) MutedUntil() fields.Timestamp        { return m.mutedUntil }
func (m *Member) MentionCount() int                   { return m.mentionCount }
func (m *Member) IsVisible() bool                     { return m.isVisible }
func (m *Member) CreatedAt() fields.Timestamp         { return m.createdAt }
func (m *Member) UpdatedAt() fields.Timestamp         { return m.updatedAt }

func getChannels(channelMap map[fields.ID]*Channel) []*Channel {
	channels := make([]*Channel, 0, len(channelMap))
	for _, ch := range channelMap {
		if ch != nil {
			channels = append(channels, ch)
		}
	}
	return channels
}

func filterPeerIDs(actorID fields.ID, parsedPeerIDs []fields.ID) []fields.ID {
	return fields.RemoveID(fields.DedupeIDs(parsedPeerIDs), actorID)
}

func filterRequiredPeerIDs(actorID fields.ID, parsedPeerIDs []fields.ID) ([]fields.ID, error) {
	peerIDs := filterPeerIDs(actorID, parsedPeerIDs)
	if len(peerIDs) == 0 {
		return nil, ErrNoNewMembers()
	}
	return peerIDs, nil
}

func getMemberIDs(members []*Member) []fields.ID {
	ids := make([]fields.ID, 0, len(members))
	for _, m := range members {
		if m != nil {
			ids = append(ids, m.UserID())
		}
	}
	return fields.DedupeIDs(ids)
}

func indexMemberships(members []*Member) ([]fields.ID, map[fields.ID]*Member) {
	channelIDs := make([]fields.ID, 0, len(members))
	membershipMap := make(map[fields.ID]*Member, len(members))

	for _, m := range members {
		if m == nil {
			continue
		}
		chID := m.ChannelID()
		channelIDs = append(channelIDs, chID)
		membershipMap[chID] = m
	}

	return channelIDs, membershipMap
}

func sortMembers(members []*Member, userMap map[fields.ID]*user.User) {
	displayName := func(m *Member) string {
		if m == nil {
			return ""
		}
		if u := userMap[m.UserID()]; u != nil {
			return u.DisplayName().String()
		}
		return ""
	}

	slices.SortFunc(members, func(a, b *Member) int {
		if cmp := cmp.Compare(displayName(a), displayName(b)); cmp != 0 {
			return cmp
		}
		return a.UserID().Compare(b.UserID())
	})
}

func validateMembership(userID fields.ID, members []*Member) (*Member, error) {
	for _, m := range members {
		if m != nil && m.UserID().Equals(userID) {
			return m, nil
		}
	}
	return nil, ErrNotChannelMember()
}
