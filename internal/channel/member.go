package channel

import (
	"bonfire-api/internal/fields"
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
	ErrInvalidDMVisibility     = errors.New("invalid dm visibility value")
)

// -----------------------------------------------------------------------------
// Domain Entities
// -----------------------------------------------------------------------------

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

// -----------------------------------------------------------------------------
// Getters
// -----------------------------------------------------------------------------

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

// -----------------------------------------------------------------------------
// Constructors / Factory Methods
// -----------------------------------------------------------------------------

// NewMember creates a fresh Member domain entity defaults to DMVisibilityVisible.
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
		pinnedAt:     nil,
		dmVisibility: DMVisibilityVisible,
		createdAt:    now,
		updatedAt:    now,
	}, nil
}

// ReconstituteMember restores an existing Member entity from persistence.
func ReconstituteMember(
	rawChannelID, rawUserID uuid.UUID,
	rawLastReadMessageID *uuid.UUID,
	mentionCount int32,
	lastReadAt time.Time,
	rawPinnedAt *time.Time,
	rawDMVisibility int16,
	createdAt, updatedAt time.Time,
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

	visibility, err := NewDMVisibility(rawDMVisibility)
	if err != nil {
		return nil, err
	}

	var pinnedAt *time.Time
	if rawPinnedAt != nil {
		t := rawPinnedAt.UTC()
		pinnedAt = &t
	}

	return &Member{
		channelID:         chID,
		userID:            uID,
		lastReadMessageID: msgID,
		mentionCount:      mentionCount,
		lastReadAt:        lastReadAt.UTC(),
		pinnedAt:          pinnedAt,
		dmVisibility:      visibility,
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

// TogglePinned flips the member's pinned status for sidebar/channel ordering.
func (m *Member) TogglePinned() {
	now := time.Now().UTC()
	if m.pinnedAt == nil {
		m.pinnedAt = &now
	} else {
		m.pinnedAt = nil
	}
	m.touchWith(now)
}

// CloseDM sets visibility state to DMVisibilityHidden.
func (m *Member) CloseDM() {
	if m.dmVisibility == DMVisibilityHidden {
		return
	}
	m.dmVisibility = DMVisibilityHidden
	m.touch()
}

// OpenDM sets visibility state to DMVisibilityVisible.
func (m *Member) OpenDM() {
	if m.dmVisibility == DMVisibilityVisible {
		return
	}
	m.dmVisibility = DMVisibilityVisible
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
	memberSince   time.Time
	lastReadAt    time.Time
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
func (m *MemberListItem) MemberSince() time.Time   { return m.memberSince }
func (m *MemberListItem) LastReadAt() time.Time    { return m.lastReadAt }
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
	memberSince time.Time,
	lastReadAt time.Time,
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
		memberSince:   memberSince.UTC(),
		lastReadAt:    lastReadAt.UTC(),
		username:      username,
		displayName:   displayName,
		avatarURL:     rawAvatarURL,
		userCreatedAt: userCreatedAt.UTC(),
	}, nil
}
