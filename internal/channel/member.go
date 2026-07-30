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
	ErrMentionCountNegative    = errors.New("mention count cannot be negative")
)

// Member represents a user's membership and read state within a channel.
type Member struct {
	channelID         ID
	userID            UserID
	lastReadMessageID *MessageID
	mentionCount      int32
	lastReadAt        time.Time
	createdAt         time.Time
	updatedAt         time.Time
}

// -----------------------------------------------------------------------------
// Getters
// -----------------------------------------------------------------------------

func (m *Member) ChannelID() ID                 { return m.channelID }
func (m *Member) UserID() UserID                { return m.userID }
func (m *Member) LastReadMessageID() *MessageID { return m.lastReadMessageID }
func (m *Member) MentionCount() int32           { return m.mentionCount }
func (m *Member) LastReadAt() time.Time         { return m.lastReadAt }
func (m *Member) CreatedAt() time.Time          { return m.createdAt }
func (m *Member) UpdatedAt() time.Time          { return m.updatedAt }

// -----------------------------------------------------------------------------
// Constructors / Factory Methods
// -----------------------------------------------------------------------------

// NewMember creates a fresh Member domain entity.
func NewMember(rawChannelID, rawUserID uuid.UUID) (*Member, error) {
	chID, err := NewID(rawChannelID)
	if err != nil {
		return nil, ErrMemberChannelIDRequired
	}

	uID, err := NewUserID(rawUserID)
	if err != nil {
		return nil, ErrMemberUserIDRequired
	}

	now := time.Now().UTC()

	return &Member{
		channelID:    chID,
		userID:       uID,
		mentionCount: 0,
		lastReadAt:   now,
		createdAt:    now,
		updatedAt:    now,
	}, nil
}

// ReconstituteMember restores an existing Member entity from persistence.
func ReconstituteMember(
	rawChannelID, rawUserID uuid.UUID,
	rawLastReadMessageID *uuid.UUID,
	mentionCount int32,
	lastReadAt, createdAt, updatedAt time.Time,
) (*Member, error) {
	chID, err := NewID(rawChannelID)
	if err != nil {
		return nil, ErrMemberChannelIDRequired
	}

	uID, err := NewUserID(rawUserID)
	if err != nil {
		return nil, ErrMemberUserIDRequired
	}

	msgID, err := NewMessageIDPtr(rawLastReadMessageID)
	if err != nil {
		return nil, err
	}

	if mentionCount < 0 {
		return nil, ErrMentionCountNegative
	}

	return &Member{
		channelID:         chID,
		userID:            uID,
		lastReadMessageID: msgID,
		mentionCount:      mentionCount,
		lastReadAt:        lastReadAt.UTC(),
		createdAt:         createdAt.UTC(),
		updatedAt:         updatedAt.UTC(),
	}, nil
}

// -----------------------------------------------------------------------------
// Domain Mutations
// -----------------------------------------------------------------------------

// SetLastRead updates the last read message pointer, updates lastReadAt, and clears mentions.
func (m *Member) SetLastRead(messageID MessageID) error {
	if !messageID.IsValid() {
		return ErrIDNil
	}

	// No-op check: already at this message and mentions cleared
	if m.lastReadMessageID != nil && m.lastReadMessageID.Equals(messageID) {
		return nil
	}

	now := time.Now().UTC()
	msgID := messageID

	m.lastReadMessageID = &msgID
	m.lastReadAt = now
	m.touchWith(now)

	return nil
}

// IncrementMention increments the unread mention counter.
func (m *Member) IncrementMention() {
	m.mentionCount++
	m.touch()
}

// ResetMentions resets the unread mention counter without advancing the read message pointer.
func (m *Member) ResetMentions() {
	if m.mentionCount == 0 {
		return
	}
	m.mentionCount = 0
	m.touch()
}

func (m *Member) touch() {
	m.updatedAt = time.Now().UTC()
}

func (m *Member) touchWith(t time.Time) {
	m.updatedAt = t
}

// -----------------------------------------------------------------------------
// MemberListItem
// -----------------------------------------------------------------------------

// MemberListItem represents a read-optimized projection of a channel member
// joined with their aggregate user profile data.
type MemberListItem struct {
	channelID     ID
	userID        UserID
	username      string
	displayName   string
	avatarURL     *string
	userCreatedAt time.Time
}

// -----------------------------------------------------------------------------
// Getters
// -----------------------------------------------------------------------------

func (m *MemberListItem) ChannelID() ID            { return m.channelID }
func (m *MemberListItem) UserID() UserID           { return m.userID }
func (m *MemberListItem) Username() string         { return m.username }
func (m *MemberListItem) DisplayName() string      { return m.displayName }
func (m *MemberListItem) AvatarURL() *string       { return m.avatarURL }
func (m *MemberListItem) UserCreatedAt() time.Time { return m.userCreatedAt }

// -----------------------------------------------------------------------------
// Factory Methods
// -----------------------------------------------------------------------------

// ReconstituteMemberListItem restores a MemberListItem projection from persistence.
func ReconstituteMemberListItem(
	rawChannelID uuid.UUID,
	rawUserID uuid.UUID,
	username string,
	displayName string,
	rawAvatarURL *string,
	userCreatedAt time.Time,
) (*MemberListItem, error) {
	channelID, err := NewID(rawChannelID)
	if err != nil {
		return nil, err
	}

	userID, err := NewUserID(rawUserID)
	if err != nil {
		return nil, err
	}

	return &MemberListItem{
		channelID:     channelID,
		userID:        userID,
		username:      username,
		displayName:   displayName,
		avatarURL:     rawAvatarURL,
		userCreatedAt: userCreatedAt,
	}, nil
}
