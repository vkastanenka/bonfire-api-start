package channel

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type ChannelView struct {
	ID            uuid.UUID   `json:"id"`
	Type          ChannelType `json:"type"`
	Name          *string     `json:"name,omitempty"`
	IconURL       *string     `json:"iconUrl,omitempty"`
	LastMessageID *uuid.UUID  `json:"lastMessageId,omitempty"`
	LastMessageAt time.Time   `json:"lastMessageAt"`
	CreatedAt     time.Time   `json:"createdAt"`
	UpdatedAt     time.Time   `json:"updatedAt"`
}

func ParseChannelView(c *Channel) ChannelView {
	return ChannelView{
		ID:            c.ID,
		Type:          c.Type,
		Name:          c.Name,
		IconURL:       c.IconURL,
		LastMessageID: c.LastMessageID,
		LastMessageAt: c.LastMessageAt,
		CreatedAt:     c.CreatedAt,
		UpdatedAt:     c.UpdatedAt,
	}
}

type MemberView struct {
	ChannelID         uuid.UUID  `json:"channelId"`
	UserID            uuid.UUID  `json:"userId"`
	LastReadMessageID *uuid.UUID `json:"lastReadMessageId,omitempty"`
	LastReadMessageAt time.Time  `json:"lastReadMessageAt"`
	MutedUntil        *time.Time `json:"mutedUntil,omitempty"`
	PinnedAt          *time.Time `json:"pinnedAt,omitempty"`
	MentionCount      int        `json:"mentionCount"`
	IsVisible         bool       `json:"isVisible"`
	CreatedAt         time.Time  `json:"createdAt"`
	UpdatedAt         time.Time  `json:"updatedAt"`
}

func ParseMemberView(m *Member) MemberView {
	return MemberView{
		ChannelID:         m.ChannelID,
		UserID:            m.UserID,
		LastReadMessageID: m.LastReadMessageID,
		LastReadMessageAt: m.LastReadMessageAt,
		MutedUntil:        m.MutedUntil,
		PinnedAt:          m.PinnedAt,
		MentionCount:      m.MentionCount,
		IsVisible:         m.IsVisible,
		CreatedAt:         m.CreatedAt,
		UpdatedAt:         m.UpdatedAt,
	}
}

func ParseMemberViewsMap(members []*Member) map[uuid.UUID]MemberView {
	if len(members) == 0 {
		return nil
	}

	views := make(map[uuid.UUID]MemberView, len(members))
	for _, m := range members {
		if m != nil {
			views[m.UserID] = ParseMemberView(m)
		}
	}

	return views
}

type MessageView struct {
	ID               uuid.UUID       `json:"id"`
	ChannelID        uuid.UUID       `json:"channelId"`
	AuthorID         *uuid.UUID      `json:"authorId,omitempty"`
	Type             MessageType     `json:"type"`
	Content          *string         `json:"content,omitempty"`
	Metadata         json.RawMessage `json:"metadata,omitempty"`
	ReplyToMessageID *uuid.UUID      `json:"replyToMessageId,omitempty"`
	ForwardMessageID *uuid.UUID      `json:"forwardMessageId,omitempty"`
	ForwardChannelID *uuid.UUID      `json:"forwardChannelId,omitempty"`
	PinnedAt         *time.Time      `json:"pinnedAt,omitempty"`
	CreatedAt        time.Time       `json:"createdAt"`
	UpdatedAt        time.Time       `json:"updatedAt"`
	EditedAt         *time.Time      `json:"editedAt,omitempty"`
}

func ParseMessageView(m *Message) MessageView {
	return MessageView{
		ID:               m.ID,
		ChannelID:        m.ChannelID,
		AuthorID:         m.AuthorID,
		Type:             m.Type,
		Content:          m.Content,
		Metadata:         m.Metadata,
		ReplyToMessageID: m.ReplyToMessageID,
		ForwardMessageID: m.ForwardMessageID,
		ForwardChannelID: m.ForwardChannelID,
		PinnedAt:         m.PinnedAt,
		CreatedAt:        m.CreatedAt,
		UpdatedAt:        m.UpdatedAt,
		EditedAt:         m.EditedAt,
	}
}

func ParseMessageViews(messages []*Message) []MessageView {
	if len(messages) == 0 {
		return nil
	}

	views := make([]MessageView, len(messages))
	for i, m := range messages {
		views[i] = ParseMessageView(m)
	}

	return views
}
