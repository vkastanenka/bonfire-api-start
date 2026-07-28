package channel

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrMemberChannelIDRequired = errors.New("channel id is required")
	ErrMemberUserIDRequired    = errors.New("user id is required")
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
	if channelID == uuid.Nil {
		return nil, ErrMemberChannelIDRequired
	}
	if userID == uuid.Nil {
		return nil, ErrMemberUserIDRequired
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
) (*Member, error) {
	if channelID == uuid.Nil {
		return nil, ErrMemberChannelIDRequired
	}
	if userID == uuid.Nil {
		return nil, ErrMemberUserIDRequired
	}
	if mentionCount < 0 {
		mentionCount = 0
	}

	return &Member{
		channelID:         channelID,
		userID:            userID,
		joinedAt:          joinedAt,
		lastReadMessageID: lastReadMessageID,
		mentionCount:      mentionCount,
	}, nil
}

func (m *Member) MarkRead(messageID uuid.UUID) {
	m.lastReadMessageID = &messageID
	m.mentionCount = 0
}

func (m *Member) IncrementMention() {
	m.mentionCount++
}

func (m *Member) ResetMentions() {
	m.mentionCount = 0
}

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
	createdAt, updatedAt time.Time,
) (*UserSidebarItem, error) {
	ch, err := Reconstitute(
		channelID,
		chType,
		ownerID,
		name,
		iconURL,
		lastMessageID,
		createdAt,
		updatedAt,
	)
	if err != nil {
		return nil, err
	}

	mem, err := ReconstituteMember(
		channelID,
		userID,
		joinedAt,
		lastReadMessageID,
		mentionCount,
	)
	if err != nil {
		return nil, err
	}

	return &UserSidebarItem{
		Member:  *mem,
		Channel: *ch,
	}, nil
}
