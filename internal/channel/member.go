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
	mentionCount      int32
	isVisible         bool
	createdAt         fields.Timestamp
	updatedAt         fields.Timestamp
}

func ParseMember(
	channelID fields.ID,
	userID fields.ID,
	lastReadMessageID fields.ID,
	lastReadMessageAt fields.Timestamp,
	pinnedAt fields.Timestamp,
	mutedUntil fields.Timestamp,
	mentionCount int32,
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

// TOODO: Deprecate
func ParseMembers(
	channelID fields.ID,
	userIDs []fields.ID,
	lastReadMessageID fields.ID,
	lastReadMessageAt fields.Timestamp,
	pinnedAt fields.Timestamp,
	mutedUntil fields.Timestamp,
	mentionCount int32,
	isVisible bool,
	createdAt fields.Timestamp,
	updatedAt fields.Timestamp,
) []*Member {
	if len(userIDs) == 0 {
		return nil
	}

	members := make([]*Member, 0, len(userIDs))
	for _, userID := range userIDs {
		members = append(members, ParseMember(
			channelID,
			userID,
			lastReadMessageID,
			lastReadMessageAt,
			pinnedAt,
			mutedUntil,
			mentionCount,
			isVisible,
			createdAt,
			updatedAt,
		))
	}

	return members
}

func NewCreator(
	channelID fields.ID,
	userID fields.ID,
	now fields.Timestamp,
) *Member {
	return &Member{
		channelID:         channelID,
		userID:            userID,
		lastReadMessageID: fields.ID{},
		lastReadMessageAt: fields.Timestamp{},
		pinnedAt:          fields.Timestamp{},
		mutedUntil:        fields.Timestamp{},
		mentionCount:      0,
		isVisible:         true,
		createdAt:         now,
		updatedAt:         now,
	}
}

func NewPeer(
	channelID fields.ID,
	userID fields.ID,
	now fields.Timestamp,
) *Member {
	return &Member{
		channelID:         channelID,
		userID:            userID,
		lastReadMessageID: fields.ID{},
		lastReadMessageAt: fields.Timestamp{},
		pinnedAt:          fields.Timestamp{},
		mutedUntil:        fields.Timestamp{},
		mentionCount:      1,
		isVisible:         true,
		createdAt:         now,
		updatedAt:         now,
	}
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

func FilterPeerIDs(actorID fields.ID, parsedPeerIDs []fields.ID) []fields.ID {
	return fields.RemoveID(fields.DedupeIDs(parsedPeerIDs), actorID)
}

func (m *Member) ChannelID() fields.ID                { return m.channelID }
func (m *Member) UserID() fields.ID                   { return m.userID }
func (m *Member) LastReadMessageID() fields.ID        { return m.lastReadMessageID }
func (m *Member) LastReadMessageAt() fields.Timestamp { return m.lastReadMessageAt }
func (m *Member) PinnedAt() fields.Timestamp          { return m.pinnedAt }
func (m *Member) MutedUntil() fields.Timestamp        { return m.mutedUntil }
func (m *Member) MentionCount() int32                 { return m.mentionCount }
func (m *Member) IsVisible() bool                     { return m.isVisible }
func (m *Member) CreatedAt() fields.Timestamp         { return m.createdAt }
func (m *Member) UpdatedAt() fields.Timestamp         { return m.updatedAt }

func (m *Member) SetLastReadMessage(id fields.ID, at fields.Timestamp, now fields.Timestamp) {
	m.lastReadMessageID = id
	m.lastReadMessageAt = at
	m.touch(now)
}

func (m *Member) SetPinnedAt(now fields.Timestamp) {
	m.pinnedAt = now
	m.touch(now)
}

func (m *Member) SetMutedUntil(now fields.Timestamp) {
	m.mutedUntil = now
	m.touch(now)
}

func (m *Member) IncrementMention(now fields.Timestamp) {
	m.mentionCount++
	m.touch(now)
}

func (m *Member) ResetMentions(now fields.Timestamp) {
	m.mentionCount = 0
	m.touch(now)
}

func (m *Member) SetIsVisible(isVisible bool, now fields.Timestamp) {
	m.isVisible = isVisible
	m.touch(now)
}

func (m *Member) touch(at fields.Timestamp) {
	m.updatedAt = at
}
