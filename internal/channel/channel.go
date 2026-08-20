package channel

import (
	"bonfire-api/internal/errs"
	"bonfire-api/internal/fields"
	"bonfire-api/internal/presence"
	"bonfire-api/internal/user"
	"fmt"
	"slices"
	"strings"

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

type SidebarPeer struct {
	id          fields.ID
	displayName user.DisplayName
	avatarURL   fields.URL
	presence    presence.Presence
}

type SidebarView struct {
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
	peers             []SidebarPeer
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

func FilterNewMemberIDs(userID fields.ID, existingMembers []*Member, newPeerIDs []fields.ID) ([]fields.ID, error) {
	existingMemberSet := make(map[fields.ID]struct{}, len(existingMembers))
	for _, m := range existingMembers {
		existingMemberSet[m.UserID()] = struct{}{}
	}

	cleanNewPeerIDs := fields.RemoveID(newPeerIDs, userID)
	toAddIDs := make([]fields.ID, 0, len(cleanNewPeerIDs))

	for _, id := range cleanNewPeerIDs {
		if _, exists := existingMemberSet[id]; !exists {
			toAddIDs = append(toAddIDs, id)
		}
	}

	if len(toAddIDs) == 0 {
		return nil, errs.InvalidArgument("All specified users are already members of this channel.").
			Reason("ALREADY_MEMBERS")
	}

	if len(existingMembers)+len(toAddIDs) > ChannelMaxPeers+1 {
		return nil, errs.InvalidArgument(fmt.Sprintf("Adding these members exceeds the maximum limit of %d members.", ChannelMaxPeers+1)).
			Reason("MAX_CAPACITY_EXCEEDED")
	}

	return toAddIDs, nil
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
	author *user.User,
	reactionMap map[fields.ID]*ReactionSummary,
) MessageView {
	var reactions []EmojiCount
	if summary, ok := reactionMap[msg.ID()]; ok && summary != nil {
		reactions = summary.Counts
	}

	return MessageView{
		id:                 msg.ID(),
		authorID:           msg.AuthorID(),
		displayName:        author.DisplayName(),
		avatarURL:          author.AvatarURL(),
		msgType:            msg.Type(),
		content:            msg.Content(),
		systemMetadata:     msg.SystemMetadata(),
		replyToMessageID:   msg.ReplyToMessageID(),
		forwardedMessageID: msg.ForwardedMessageID(),
		forwardedChannelID: msg.ForwardedChannelID(),
		createdAt:          msg.CreatedAt(),
		editedAt:           msg.EditedAt(),
		reactions:          reactions,
	}
}

func HydrateMessageViews(
	messages []*Message,
	userMap map[fields.ID]*user.User,
	reactionMap map[fields.ID]*ReactionSummary,
) []MessageView {
	if len(messages) == 0 {
		return nil
	}

	views := make([]MessageView, 0, len(messages))
	for _, msg := range messages {
		author, ok := userMap[msg.AuthorID()]
		if !ok || author == nil {
			continue
		}

		views = append(views, HydrateMessageView(msg, author, reactionMap))
	}
	return views
}

func HydrateMessagePinnedView(
	msg *Message,
	author *user.User,
) (MessagePinnedView, bool) {
	if msg == nil {
		return MessagePinnedView{}, false
	}

	if author == nil {
		author = &user.User{}
	}

	return MessagePinnedView{
		id:          msg.ID(),
		avatarURL:   author.AvatarURL(),
		displayName: author.DisplayName(),
		content:     msg.Content(),
		createdAt:   msg.CreatedAt(),
	}, true
}

func HydrateMessagePinnedViews(
	messages []*Message,
	userMap map[fields.ID]*user.User,
) []MessagePinnedView {
	if len(messages) == 0 {
		return nil
	}

	views := make([]MessagePinnedView, 0, len(messages))
	for _, msg := range messages {
		author, _ := userMap[msg.AuthorID()]
		if view, ok := HydrateMessagePinnedView(msg, author); ok {
			views = append(views, view)
		}
	}
	return views
}

func HydrateSidebarPeer(
	currentUserID fields.ID,
	pMem *Member,
	userMap map[fields.ID]*user.User,
	presenceMap map[fields.ID]presence.Presence,
) (SidebarPeer, bool) {
	if pMem == nil || pMem.UserID() == currentUserID {
		return SidebarPeer{}, false
	}

	u, ok := userMap[pMem.UserID()]
	if !ok || u == nil {
		return SidebarPeer{}, false
	}

	p, ok := presenceMap[pMem.UserID()]
	if !ok {
		p = presence.New(presence.PresenceOffline)
	}

	return SidebarPeer{
		id:          u.ID(),
		displayName: u.DisplayName(),
		avatarURL:   u.AvatarURL(),
		presence:    p,
	}, true
}

func HydrateSidebarPeers(
	currentUserID fields.ID,
	rawPeers []*Member,
	userMap map[fields.ID]*user.User,
	presenceMap map[fields.ID]presence.Presence,
) []SidebarPeer {
	views := make([]SidebarPeer, 0, len(rawPeers))
	for _, pMem := range rawPeers {
		if view, ok := HydrateSidebarPeer(currentUserID, pMem, userMap, presenceMap); ok {
			views = append(views, view)
		}
	}
	return views
}

func HydrateSidebarViews(
	currentUserID fields.ID,
	channels []*Channel,
	userMembersMap map[fields.ID]*Member,
	peerMembersMap map[fields.ID][]*Member,
	userMap map[fields.ID]*user.User,
	presenceMap map[fields.ID]presence.Presence,
) []SidebarView {
	views := make([]SidebarView, 0, len(channels))

	for _, ch := range channels {
		mem := userMembersMap[ch.ID()]
		if mem == nil {
			continue
		}

		rawPeers := peerMembersMap[ch.ID()]
		peersView := HydrateSidebarPeers(currentUserID, rawPeers, userMap, presenceMap)

		views = append(views, SidebarView{
			id:                ch.ID(),
			chType:            ch.Type(),
			name:              ch.Name(),
			iconURL:           ch.IconURL(),
			lastMessageID:     ch.LastMessageID(),
			lastMessageAt:     ch.LastMessageAt(),
			lastReadMessageID: mem.LastReadMessageID(),
			pinnedAt:          mem.PinnedAt(),
			mutedUntil:        mem.MutedUntil(),
			mentionCount:      mem.MentionCount(),
			peers:             peersView,
			memberTotal:       int16(len(rawPeers)),
		})
	}

	return views
}

func IndexMemberships(members []*Member) ([]fields.ID, map[fields.ID]*Member) {
	channelIDs := make([]fields.ID, len(members))
	membershipMap := make(map[fields.ID]*Member, len(members))
	for i, m := range members {
		chID := m.ChannelID()
		channelIDs[i] = chID
		membershipMap[chID] = m
	}
	return channelIDs, membershipMap
}

func SortMembers(members []*Member, userMap map[fields.ID]*user.User) {
	slices.SortFunc(members, func(a, b *Member) int {
		uA, okA := userMap[a.UserID()]
		uB, okB := userMap[b.UserID()]

		nameA := ""
		if okA && uA != nil {
			nameA = uA.DisplayName().String()
		}
		nameB := ""
		if okB && uB != nil {
			nameB = uB.DisplayName().String()
		}

		return strings.Compare(nameA, nameB)
	})
}

func SortMessages(messages []*Message) {
	slices.SortFunc(messages, func(a, b *Message) int {
		return a.ID().Compare(b.ID())
	})
}

func SortPinnedMessages(messages []*Message) {
	slices.SortFunc(messages, func(a, b *Message) int {
		return b.ID().Compare(a.ID())
	})
}

func SortSidebar(channels []*Channel, userMembersMap map[fields.ID]*Member) {
	slices.SortFunc(channels, func(a, b *Channel) int {
		mA := userMembersMap[a.ID()]
		mB := userMembersMap[b.ID()]

		// 1. Pinned priority
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

		// 2. Activity (lastMessageAt)
		aLast := a.LastMessageAt()
		bLast := b.LastMessageAt()
		if !aLast.Equals(bLast) {
			if aLast.After(bLast) {
				return -1
			}
			return 1
		}

		// 3. Creation date
		if a.CreatedAt().After(b.CreatedAt()) {
			return -1
		}
		if b.CreatedAt().After(a.CreatedAt()) {
			return 1
		}

		// 4. Guaranteed deterministic ID tie-breaker
		return a.ID().Compare(b.ID())
	})
}

func ValidateMaxPeers(rawPeerIDs []uuid.UUID) error {
	count := (len(rawPeerIDs))
	if count > ChannelMaxPeers {
		return errs.InvalidArgument(fmt.Sprintf("Peer list cannot exceed %d items.", ChannelMaxPeers)).
			Reason("MAX_PEERS_EXCEEDED")
	}
	return nil
}

func ValidateMinMembers(rawMemberIDs []uuid.UUID) error {
	count := (len(rawMemberIDs))
	if count < ChannelMinMembers {
		return errs.InvalidArgument(fmt.Sprintf("Member list must be at least %d items.", ChannelMinMembers)).
			Reason("MIN_MEMBERS_INVALID")
	}
	return nil
}

func ValidateMembership(userID fields.ID, members []*Member) (*Member, error) {
	for _, m := range members {
		if m.UserID() == userID {
			return m, nil
		}
	}
	return nil, errs.PermissionDenied("You are not a member of this channel.")
}
