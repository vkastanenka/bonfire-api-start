package channel

import (
	"bonfire-api/internal/errs"
	"bonfire-api/internal/fields"
	"bonfire-api/internal/presence"
	"bonfire-api/internal/user"
	"fmt"

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

type MemberView struct {
	id          fields.ID
	displayName user.DisplayName
	avatarURL   fields.URL
	presence    presence.Presence
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

func HydrateMemberView(
	m *Member,
	userMap map[fields.ID]*user.User,
	presenceMap map[fields.ID]presence.Presence,
) (MemberView, bool) {
	if m == nil {
		return MemberView{}, false
	}

	u, ok := userMap[m.UserID()]
	if !ok || u == nil {
		return MemberView{}, false
	}

	p, ok := presenceMap[m.UserID()]
	if !ok {
		p = presence.New(presence.PresenceOffline)
	}

	return MemberView{
		id:          m.UserID(),
		displayName: u.DisplayName(),
		avatarURL:   u.AvatarURL(),
		presence:    p,
	}, true
}

func HydrateMemberViews(
	members []*Member,
	userMap map[fields.ID]*user.User,
	presenceMap map[fields.ID]presence.Presence,
) []MemberView {
	views := make([]MemberView, 0, len(members))
	for _, m := range members {
		if view, ok := HydrateMemberView(m, userMap, presenceMap); ok {
			views = append(views, view)
		}
	}
	return views
}

func HydrateMessageView(
	msg *Message,
	userMap map[fields.ID]*user.User,
	reactionMap map[fields.ID]*ReactionSummary,
) (MessageView, bool) {
	if msg == nil {
		return MessageView{}, false
	}

	u, ok := userMap[msg.AuthorID()]
	if !ok || u == nil {
		return MessageView{}, false
	}

	var reactions []EmojiCount
	if summary, ok := reactionMap[msg.ID()]; ok && summary != nil {
		reactions = summary.Counts
	}

	return MessageView{
		id:                 msg.ID(),
		authorID:           msg.AuthorID(),
		displayName:        u.DisplayName(),
		avatarURL:          u.AvatarURL(),
		msgType:            msg.Type(),
		content:            msg.Content(),
		systemMetadata:     msg.SystemMetadata(),
		replyToMessageID:   msg.ReplyToMessageID(),
		forwardedMessageID: msg.ForwardedMessageID(),
		forwardedChannelID: msg.ForwardedChannelID(),
		createdAt:          msg.CreatedAt(),
		editedAt:           msg.EditedAt(),
		reactions:          reactions,
	}, true
}

func HydrateMessageViews(
	messages []*Message,
	userMap map[fields.ID]*user.User,
	reactionMap map[fields.ID]*ReactionSummary,
) []MessageView {
	views := make([]MessageView, 0, len(messages))
	for _, msg := range messages {
		if view, ok := HydrateMessageView(msg, userMap, reactionMap); ok {
			views = append(views, view)
		}
	}
	return views
}

func ValidateMaxPeers(rawPeerIDs []uuid.UUID) error {
	count := (len(rawPeerIDs))
	if count > ChannelMaxPeers {
		return errs.InvalidArgument(fmt.Sprintf("Peer list cannot exceed %d items.", ChannelMaxPeers)).
			Reason("MAX_PEERS_EXCEEDED").
			Meta("domain", "channels")
	}
	return nil
}

func ValidateMembership(members []*Member, userID fields.ID) (*Member, error) {
	for _, m := range members {
		if m.UserID() == userID {
			return m, nil
		}
	}
	return nil, errs.PermissionDenied("You are not a member of this channel.")
}
