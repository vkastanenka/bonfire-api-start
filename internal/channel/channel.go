package channel

import (
	"bonfire-api/internal/fields"
	"slices"

	"github.com/google/uuid"
)

const (
	ChannelMinMembers      = 1
	ChannelMaxMembers      = 10
	ChannelMaxPeers        = 9
	ChannelMaxSidebarItems = 100
)

type Channel struct {
	id            fields.ID
	chType        ChannelType
	name          ChannelName
	iconURL       fields.URL
	lastMessageID fields.ID
	lastMessageAt fields.Timestamp
	createdAt     fields.Timestamp
	updatedAt     fields.Timestamp
}

func ReconstituteChannel(
	id fields.ID,
	chType ChannelType,
	name ChannelName,
	iconURL fields.URL,
	lastMessageID fields.ID,
	lastMessageAt fields.Timestamp,
	createdAt fields.Timestamp,
	updatedAt fields.Timestamp,
) *Channel {
	return &Channel{
		id:            id,
		chType:        chType,
		name:          name,
		iconURL:       iconURL,
		lastMessageID: lastMessageID,
		lastMessageAt: lastMessageAt,
		createdAt:     createdAt,
		updatedAt:     updatedAt,
	}
}

func NewChannel(chType ChannelType, now fields.Timestamp) (*Channel, error) {
	id, err := fields.NewID()
	if err != nil {
		return nil, err
	}

	return ReconstituteChannel(
		id,
		chType,
		ChannelName{},
		fields.URL{},
		fields.ID{},
		fields.Timestamp{},
		now,
		now,
	), nil
}

func NewDirectChannel(now fields.Timestamp) (*Channel, error) {
	return NewChannel(NewChannelTypeDirect(), now)
}

func NewGroupChannel(now fields.Timestamp) (*Channel, error) {
	return NewChannel(NewChannelTypeGroup(), now)
}

func (c *Channel) ID() fields.ID                   { return c.id }
func (c *Channel) Type() ChannelType               { return c.chType }
func (c *Channel) Name() ChannelName               { return c.name }
func (c *Channel) IconURL() fields.URL             { return c.iconURL }
func (c *Channel) LastMessageID() fields.ID        { return c.lastMessageID }
func (c *Channel) LastMessageAt() fields.Timestamp { return c.lastMessageAt }
func (c *Channel) CreatedAt() fields.Timestamp     { return c.createdAt }
func (c *Channel) UpdatedAt() fields.Timestamp     { return c.updatedAt }

func (c *Channel) IsDirect() bool {
	return c.chType.IsDirect()
}

func (c *Channel) IsGroup() bool {
	return c.chType.IsGroup()
}

func sortSidebar(channels []*Channel, userMembersMap map[fields.ID]*Member) {
	slices.SortFunc(channels, func(a, b *Channel) int {
		mA := userMembersMap[a.ID()]
		mB := userMembersMap[b.ID()]

		aPinned := mA != nil && mA.PinnedAt().IsValid()
		bPinned := mB != nil && mB.PinnedAt().IsValid()
		if aPinned != bPinned {
			if aPinned {
				return -1
			}
			return 1
		}
		if aPinned {
			if mA.PinnedAt().After(mB.PinnedAt()) {
				return -1
			}
			if mB.PinnedAt().After(mA.PinnedAt()) {
				return 1
			}
		}

		aLast := a.LastMessageAt()
		bLast := b.LastMessageAt()
		if !aLast.Equals(bLast) {
			if aLast.After(bLast) {
				return -1
			}
			return 1
		}

		if a.CreatedAt().After(b.CreatedAt()) {
			return -1
		}
		if b.CreatedAt().After(a.CreatedAt()) {
			return 1
		}

		return a.ID().Compare(b.ID())
	})
}

func validateMaxPeers(rawPeerIDs []uuid.UUID) error {
	if len(rawPeerIDs) > ChannelMaxPeers {
		return ErrMaxPeersExceeded(ChannelMaxPeers)
	}
	return nil
}

func validateMinMembers(rawMemberIDs []uuid.UUID) error {
	if len(rawMemberIDs) < ChannelMinMembers {
		return ErrMinMembersInvalid(ChannelMinMembers)
	}
	return nil
}
