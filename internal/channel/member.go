package channel

import (
	"bonfire-api/internal/fields"
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
