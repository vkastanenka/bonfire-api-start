package channel

import (
	"bonfire-api/internal/fields"

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

func getChannelUserIDs(memberIDs []fields.ID, messages []*Message) []fields.ID {
	seen := make(map[fields.ID]struct{}, len(memberIDs)+len(messages))
	result := make([]fields.ID, 0, len(memberIDs)+len(messages))

	for _, id := range memberIDs {
		if _, exists := seen[id]; !exists {
			seen[id] = struct{}{}
			result = append(result, id)
		}
	}

	for _, msg := range messages {
		authorID := msg.AuthorID()
		if authorID.IsZero() {
			continue
		}
		if _, exists := seen[authorID]; !exists {
			seen[authorID] = struct{}{}
			result = append(result, authorID)
		}
	}

	return result
}

func indexChannels(channels []*Channel) []fields.ID {
	channelIDs := make([]fields.ID, 0, len(channels))

	for _, ch := range channels {
		if ch == nil {
			continue
		}
		channelIDs = append(channelIDs, ch.ID())
	}

	return channelIDs
}

func validateIDs(rawActorID, rawSessionID, rawChannelID uuid.UUID) (actorID, sessionID, channelID fields.ID, err error) {
	if actorID, err = fields.ParseRequiredID("actor_id", rawActorID); err != nil {
		return fields.ID{}, fields.ID{}, fields.ID{}, err
	}
	sessionID, err = fields.ParseRequiredID("session_id", rawSessionID)
	if err != nil {
		return fields.ID{}, fields.ID{}, fields.ID{}, err
	}
	if channelID, err = fields.ParseRequiredID("channel_id", rawChannelID); err != nil {
		return fields.ID{}, fields.ID{}, fields.ID{}, err
	}
	return actorID, sessionID, channelID, nil
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
