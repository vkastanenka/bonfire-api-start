package channel

import (
	"time"

	"github.com/google/uuid"
)

const (
	ChannelMinMembers      = 1
	ChannelMaxMembers      = 10
	ChannelMaxPeers        = 9
	ChannelMaxSidebarItems = 100
)

type Channel struct {
	ID            uuid.UUID
	Type          ChannelType
	Name          *string
	IconURL       *string
	LastMessageID *uuid.UUID
	LastMessageAt *time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func ReconstituteChannel(
	id uuid.UUID,
	chType ChannelType,
	name *string,
	iconURL *string,
	lastMessageID *uuid.UUID,
	lastMessageAt *time.Time,
	createdAt time.Time,
	updatedAt time.Time,
) *Channel {
	return &Channel{
		ID:            id,
		Type:          chType,
		Name:          name,
		IconURL:       iconURL,
		LastMessageID: lastMessageID,
		LastMessageAt: lastMessageAt,
		CreatedAt:     createdAt,
		UpdatedAt:     updatedAt,
	}
}

func NewChannel(chType ChannelType, now time.Time) (*Channel, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return nil, err
	}

	return ReconstituteChannel(
		id,
		chType,
		nil,
		nil,
		nil,
		nil,
		now,
		now,
	), nil
}

func NewDirectChannel(now time.Time) (*Channel, error) {
	return NewChannel(ChannelTypeDirect, now)
}

func NewGroupChannel(now time.Time) (*Channel, error) {
	return NewChannel(ChannelTypeGroup, now)
}

func (c *Channel) IsDirect() bool {
	return c.Type.IsDirect()
}

func (c *Channel) IsGroup() bool {
	return c.Type.IsGroup()
}

func getChannelUserIDs(memberIDs []uuid.UUID, messages []*Message) []uuid.UUID {
	seen := make(map[uuid.UUID]struct{}, len(memberIDs)+len(messages))
	result := make([]uuid.UUID, 0, len(memberIDs)+len(messages))

	for _, id := range memberIDs {
		if id == (uuid.UUID{}) {
			continue
		}
		if _, exists := seen[id]; !exists {
			seen[id] = struct{}{}
			result = append(result, id)
		}
	}

	for _, msg := range messages {
		if msg == nil || msg.AuthorID == nil {
			continue
		}

		authorID := *msg.AuthorID
		if authorID == (uuid.UUID{}) {
			continue
		}

		if _, exists := seen[authorID]; !exists {
			seen[authorID] = struct{}{}
			result = append(result, authorID)
		}
	}

	return result
}

func indexChannels(channels []*Channel) []uuid.UUID {
	channelIDs := make([]uuid.UUID, 0, len(channels))

	for _, ch := range channels {
		if ch == nil {
			continue
		}
		channelIDs = append(channelIDs, ch.ID)
	}

	return channelIDs
}

func validateMaxPeers(rawPeerIDs []uuid.UUID) error {
	if len(rawPeerIDs) > ChannelMaxPeers {
		return ErrMaxPeersExceeded()
	}
	return nil
}

func validateMinMembers(rawMemberIDs []uuid.UUID) error {
	if len(rawMemberIDs) < ChannelMinMembers {
		return ErrMinMembersInvalid()
	}
	return nil
}
