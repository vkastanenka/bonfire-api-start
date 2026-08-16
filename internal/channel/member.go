package channel

import (
	"bonfire-api/internal/fields"
	"bonfire-api/internal/presence"
	"bonfire-api/internal/user"
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

type MemberView struct {
	id          fields.ID
	displayName user.DisplayName
	avatarURL   fields.URL
	presence    presence.Presence
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
