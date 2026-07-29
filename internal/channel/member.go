package channel

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// -----------------------------------------------------------------------------
// Domain Errors
// -----------------------------------------------------------------------------

var (
	ErrMemberChannelIDRequired = errors.New("channel id is required")
	ErrMemberUserIDRequired    = errors.New("user id is required")
)

// Member represents a user's membership and read state within a channel.
type Member struct {
	channelID         uuid.UUID
	userID            uuid.UUID
	joinedAt          time.Time
	lastReadMessageID *uuid.UUID
	mentionCount      int32
}

// -----------------------------------------------------------------------------
// Getters
// -----------------------------------------------------------------------------

func (m *Member) ChannelID() uuid.UUID          { return m.channelID }
func (m *Member) UserID() uuid.UUID             { return m.userID }
func (m *Member) JoinedAt() time.Time           { return m.joinedAt }
func (m *Member) LastReadMessageID() *uuid.UUID { return m.lastReadMessageID }
func (m *Member) MentionCount() int32           { return m.mentionCount }

// -----------------------------------------------------------------------------
// Constructors / Factory Methods
// -----------------------------------------------------------------------------

// NewMember creates a fresh Member domain entity.
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

// ReconstituteMember restores an existing Member entity from persistence.
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

// -----------------------------------------------------------------------------
// Domain Mutations
// -----------------------------------------------------------------------------

// MarkRead updates the last read message pointer and clears unread mentions.
func (m *Member) MarkRead(messageID uuid.UUID) {
	if messageID == uuid.Nil {
		return
	}
	if m.lastReadMessageID != nil && *m.lastReadMessageID == messageID && m.mentionCount == 0 {
		return
	}

	m.lastReadMessageID = &messageID
	m.mentionCount = 0
}

// IncrementMention increments the unread mention counter.
func (m *Member) IncrementMention() {
	m.mentionCount++
}

// ResetMentions resets the unread mention counter without advancing read state.
func (m *Member) ResetMentions() {
	m.mentionCount = 0
}
