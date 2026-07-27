package channel

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

type Member struct {
	channelID         uuid.UUID
	userID            uuid.UUID
	joinedAt          time.Time
	lastReadMessageID *uuid.UUID
	mentionCount      int32
}

func (m *Member) ChannelID() uuid.UUID          { return m.channelID }
func (m *Member) UserID() uuid.UUID             { return m.userID }
func (m *Member) JoinedAt() time.Time           { return m.joinedAt }
func (m *Member) LastReadMessageID() *uuid.UUID { return m.lastReadMessageID }
func (m *Member) MentionCount() int32           { return m.mentionCount }

func NewMember(channelID, userID uuid.UUID) (*Member, error) {
	if channelID == uuid.Nil || userID == uuid.Nil {
		return nil, errors.New("channelID and userID are required")
	}

	return &Member{
		channelID:    channelID,
		userID:       userID,
		joinedAt:     time.Now().UTC(),
		mentionCount: 0,
	}, nil
}

func ReconstituteMember(
	channelID, userID uuid.UUID,
	joinedAt time.Time,
	lastReadMessageID *uuid.UUID,
	mentionCount int32,
) *Member {
	return &Member{
		channelID:         channelID,
		userID:            userID,
		joinedAt:          joinedAt,
		lastReadMessageID: lastReadMessageID,
		mentionCount:      mentionCount,
	}
}

func (m *Member) MarkRead(messageID uuid.UUID) {
	m.lastReadMessageID = &messageID
	m.mentionCount = 0
}

func (m *Member) IncrementMention() {
	m.mentionCount++
}

// UserSidebarItem combines membership details with channel metadata for UI display.
type UserSidebarItem struct {
	Member  Member
	Channel Channel
}

func ReconstituteSidebarItem(
	channelID, userID uuid.UUID,
	joinedAt time.Time,
	lastReadMessageID *uuid.UUID,
	mentionCount int32,
	chType Type,
	ownerID *uuid.UUID,
	name, iconURL *string,
	lastMessageID *uuid.UUID,
	updatedAt time.Time,
) *UserSidebarItem {
	return &UserSidebarItem{
		Member: Member{
			channelID:         channelID,
			userID:            userID,
			joinedAt:          joinedAt,
			lastReadMessageID: lastReadMessageID,
			mentionCount:      mentionCount,
		},
		Channel: Channel{
			id:            channelID,
			channelType:   chType,
			ownerID:       ownerID,
			name:          name,
			iconURL:       iconURL,
			lastMessageID: lastMessageID,
			updatedAt:     updatedAt,
		},
	}
}
