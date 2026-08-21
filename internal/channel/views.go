package channel

import (
	"bonfire-api/internal/fields"
	"bonfire-api/internal/presence"
	"bonfire-api/internal/user"
)

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
	memberTotal       int
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
			memberTotal:       len(rawPeers),
		})
	}

	return views
}
