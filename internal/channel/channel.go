package channel

import (
	"bonfire-api/internal/fields"
	"bonfire-api/internal/presence"
	"bonfire-api/internal/user"
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

type ChannelSidebarPeerView struct {
	id          fields.ID
	displayName user.DisplayName
	avatarURL   fields.URL
	presence    presence.Presence
}

type ChannelSidebarView struct {
	id                fields.ID
	chType            ChannelType
	name              ChannelName
	iconURL           fields.URL
	lastMessageID     fields.ID
	lastMessageAt     fields.Timestamp
	lastReadMessageID fields.ID
	pinnedAt          fields.Timestamp
	mutedUntil        fields.Timestamp
	mentionCount      int32
	peers             []ChannelSidebarPeerView
	memberTotal       int16
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
	return c.chType.Raw() == ChannelTypeDirect
}

func (c *Channel) IsGroup() bool {
	return c.chType.Raw() == ChannelTypeGroup
}

func ParseChannel(
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

func (c *Channel) SetName(name ChannelName, now fields.Timestamp) {
	c.name = name
	c.touch(now)
}

func (c *Channel) SetIcon(icon fields.URL, now fields.Timestamp) {
	c.iconURL = icon
	c.touch(now)
}

func (c *Channel) SetLastMessage(id fields.ID, at fields.Timestamp, now fields.Timestamp) {
	c.lastMessageID = id
	c.lastMessageAt = at
	c.touch(now)
}

func (u *Channel) touch(at fields.Timestamp) {
	u.updatedAt = at
}
