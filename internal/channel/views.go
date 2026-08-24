package channel

import (
	"bonfire-api/internal/fields"
	"bonfire-api/internal/presence"
	"bonfire-api/internal/user"
)

type MemberView struct {
	ID          fields.ID         `json:"id"`
	DisplayName user.DisplayName  `json:"display_name"`
	AvatarURL   fields.URL        `json:"avatar_url"`
	Presence    presence.Presence `json:"presence"`
}

type MessageView struct {
	Reactions          []EmojiCount     `json:"reactions"`
	ID                 fields.ID        `json:"id"`
	AuthorID           fields.ID        `json:"author_id"`
	DisplayName        user.DisplayName `json:"display_name"`
	AvatarURL          fields.URL       `json:"avatar_url"`
	MsgType            MessageType      `json:"msg_type"`
	Content            MessageContent   `json:"content"`
	SystemMetadata     fields.JSON      `json:"system_metadata"`
	ReplyToMessageID   fields.ID        `json:"reply_to_message_id"`
	ForwardedMessageID fields.ID        `json:"forwarded_message_id"`
	ForwardedChannelID fields.ID        `json:"forwarded_channel_id"`
	CreatedAt          fields.Timestamp `json:"created_at"`
	EditedAt           fields.Timestamp `json:"edited_at"`
}

type MessagePinnedView struct {
	ID          fields.ID        `json:"id"`
	AvatarURL   fields.URL       `json:"avatar_url"`
	DisplayName user.DisplayName `json:"display_name"`
	Content     MessageContent   `json:"content"`
	PinnedAt    fields.Timestamp `json:"pinned_at"`
	CreatedAt   fields.Timestamp `json:"created_at"`
}

type SidebarView struct {
	Peers             []MemberView     `json:"peers"`
	ID                fields.ID        `json:"id"`
	ChType            ChannelType      `json:"ch_type"`
	Name              ChannelName      `json:"name"`
	IconURL           fields.URL       `json:"icon_url"`
	LastMessageID     fields.ID        `json:"last_message_id"`
	LastMessageAt     fields.Timestamp `json:"last_message_at"`
	LastReadMessageID fields.ID        `json:"last_read_message_id"`
	PinnedAt          fields.Timestamp `json:"pinned_at"`
	MutedUntil        fields.Timestamp `json:"muted_until"`
	MemberTotal       int              `json:"member_total"`
	MentionCount      int            `json:"mention_count"`
}

func HydrateMemberView(
	memberID fields.ID,
	u *user.User,
	p presence.Presence,
) (MemberView, bool) {
	if u == nil {
		return MemberView{}, false
	}

	return MemberView{
		ID:          memberID,
		DisplayName: u.DisplayName(),
		AvatarURL:   u.AvatarURL(),
		Presence:    p,
	}, true
}

func HydrateMemberViews(
	members []*Member,
	userMap map[fields.ID]*user.User,
	presenceMap map[fields.ID]presence.Presence,
) []MemberView {
	views := make([]MemberView, 0, len(members))

	for _, m := range members {
		if m == nil {
			continue
		}

		u := userMap[m.UserID()]
		if u == nil {
			continue
		}

		p, ok := presenceMap[m.UserID()]
		if !ok {
			p = presence.New(presence.PresenceOffline)
		}

		if view, ok := HydrateMemberView(m.UserID(), u, p); ok {
			views = append(views, view)
		}
	}

	return views
}

func HydrateMessageView(
	msg *Message,
	author *user.User,
	reactions []EmojiCount,
) (MessageView, bool) {
	if msg == nil || author == nil {
		return MessageView{}, false
	}

	return MessageView{
		ID:                 msg.ID(),
		AuthorID:           msg.AuthorID(),
		DisplayName:        author.DisplayName(),
		AvatarURL:          author.AvatarURL(),
		MsgType:            msg.Type(),
		Content:            msg.Content(),
		SystemMetadata:     msg.Metadata(),
		ReplyToMessageID:   msg.ReplyToMessageID(),
		ForwardedMessageID: msg.ForwardMessageID(),
		ForwardedChannelID: msg.ForwardChannelID(),
		CreatedAt:          msg.CreatedAt(),
		EditedAt:           msg.EditedAt(),
		Reactions:          reactions,
	}, true
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
		if msg == nil {
			continue
		}

		author := userMap[msg.AuthorID()]
		if author == nil {
			continue
		}

		var reactions []EmojiCount
		if summary := reactionMap[msg.ID()]; summary != nil {
			reactions = summary.Counts
		}

		if view, ok := HydrateMessageView(msg, author, reactions); ok {
			views = append(views, view)
		}
	}
	return views
}

func HydrateMessagePinnedView(
	msg *Message,
	author *user.User,
) (MessagePinnedView, bool) {
	if msg == nil || author == nil {
		return MessagePinnedView{}, false
	}

	return MessagePinnedView{
		ID:          msg.ID(),
		AvatarURL:   author.AvatarURL(),
		DisplayName: author.DisplayName(),
		Content:     msg.Content(),
		PinnedAt:    msg.PinnedAt(),
		CreatedAt:   msg.CreatedAt(),
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
		if msg == nil {
			continue
		}

		author := userMap[msg.AuthorID()]
		if author == nil {
			continue
		}

		if view, ok := HydrateMessagePinnedView(msg, author); ok {
			views = append(views, view)
		}
	}
	return views
}

func HydrateSidebarPeer(
	currentUserID fields.ID,
	pMem *Member,
	u *user.User,
	p presence.Presence,
) (MemberView, bool) {
	if pMem == nil || pMem.UserID() == currentUserID {
		return MemberView{}, false
	}

	return HydrateMemberView(pMem.UserID(), u, p)
}

func HydrateSidebarPeers(
	currentUserID fields.ID,
	rawPeers []*Member,
	userMap map[fields.ID]*user.User,
	presenceMap map[fields.ID]presence.Presence,
) []MemberView {
	views := make([]MemberView, 0, len(rawPeers))
	for _, pMem := range rawPeers {
		if pMem == nil {
			continue
		}

		u := userMap[pMem.UserID()]
		if u == nil {
			continue
		}

		p, ok := presenceMap[pMem.UserID()]
		if !ok {
			p = presence.New(presence.PresenceOffline)
		}

		if view, ok := HydrateSidebarPeer(currentUserID, pMem, u, p); ok {
			views = append(views, view)
		}
	}
	return views
}

func HydrateSidebarView(
	ch *Channel,
	mem *Member,
	peersView []MemberView,
	memberTotal int,
) (SidebarView, bool) {
	if ch == nil || mem == nil {
		return SidebarView{}, false
	}

	return SidebarView{
		ID:                ch.ID(),
		ChType:            ch.Type(),
		Name:              ch.Name(),
		IconURL:           ch.IconURL(),
		LastMessageID:     ch.LastMessageID(),
		LastMessageAt:     ch.LastMessageAt(),
		LastReadMessageID: mem.LastReadMessageID(),
		PinnedAt:          mem.PinnedAt(),
		MutedUntil:        mem.MutedUntil(),
		MentionCount:      mem.MentionCount(),
		Peers:             peersView,
		MemberTotal:       memberTotal,
	}, true
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
		if ch == nil {
			continue
		}

		mem := userMembersMap[ch.ID()]
		if mem == nil {
			continue
		}

		rawPeers := peerMembersMap[ch.ID()]
		peersView := HydrateSidebarPeers(currentUserID, rawPeers, userMap, presenceMap)

		if view, ok := HydrateSidebarView(ch, mem, peersView, len(rawPeers)); ok {
			views = append(views, view)
		}
	}

	return views
}
