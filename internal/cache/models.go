package cache

import (
	"bonfire-api/internal/channel"
	"bonfire-api/internal/fields"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type Channel struct {
	ID            uuid.UUID `json:"id"`
	Type          int16     `json:"type"`
	Name          string    `json:"name"`
	IconURL       string    `json:"icon_url"`
	LastMessageID uuid.UUID `json:"last_message_id"`
	LastMessageAt time.Time `json:"last_message_at"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func (c Channel) ToDomain() (*channel.Channel, error) {
	id, err := fields.ParseRequiredID("id", c.ID)
	if err != nil {
		return nil, err
	}

	chType, err := channel.ParseChannelType(c.Type)
	if err != nil {
		return nil, err
	}

	name, err := channel.ParseChannelName(c.Name)
	if err != nil {
		return nil, err
	}

	iconURL, err := fields.ParseURL("icon_url", c.IconURL)
	if err != nil {
		return nil, err
	}

	lastMessageID, err := fields.ParseID("last_message_id", c.LastMessageID)
	if err != nil {
		return nil, err
	}

	return channel.ParseChannel(
		id,
		chType,
		name,
		iconURL,
		lastMessageID,
		fields.NewTimestamp(c.LastMessageAt),
		fields.NewTimestamp(c.CreatedAt),
		fields.NewTimestamp(c.UpdatedAt),
	), nil
}

func ParseChannel(ch *channel.Channel) Channel {
	return Channel{
		ID:            ch.ID().UUID(),
		Type:          ch.Type().Int16(),
		Name:          ch.Name().String(),
		IconURL:       ch.IconURL().String(),
		LastMessageID: ch.LastMessageID().UUID(),
		LastMessageAt: ch.LastMessageAt().Time(),
		CreatedAt:     ch.CreatedAt().Time(),
		UpdatedAt:     ch.UpdatedAt().Time(),
	}
}

type Member struct {
	ChannelID         uuid.UUID `json:"channel_id"`
	UserID            uuid.UUID `json:"user_id"`
	LastReadMessageID uuid.UUID `json:"last_read_message_id"`
	LastReadMessageAt time.Time `json:"last_read_message_at"`
	PinnedAt          time.Time `json:"pinned_at"`
	MutedUntil        time.Time `json:"muted_until"`
	MentionCount      int32     `json:"mention_count"`
	IsVisible         bool      `json:"is_visible"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

func (m Member) ToDomain() (*channel.Member, error) {
	channelID, err := fields.ParseRequiredID("channel_id", m.ChannelID)
	if err != nil {
		return nil, err
	}

	userID, err := fields.ParseRequiredID("user_id", m.UserID)
	if err != nil {
		return nil, err
	}

	lastReadMessageID, err := fields.ParseID("last_read_message_id", m.LastReadMessageID)
	if err != nil {
		return nil, err
	}

	return channel.ParseMember(
		channelID,
		userID,
		lastReadMessageID,
		fields.NewTimestamp(m.LastReadMessageAt),
		fields.NewTimestamp(m.PinnedAt),
		fields.NewTimestamp(m.MutedUntil),
		m.MentionCount,
		m.IsVisible,
		fields.NewTimestamp(m.CreatedAt),
		fields.NewTimestamp(m.UpdatedAt),
	), nil
}

func ParseMember(m *channel.Member) Member {
	return Member{
		ChannelID:         m.ChannelID().UUID(),
		UserID:            m.UserID().UUID(),
		LastReadMessageID: m.LastReadMessageID().UUID(),
		LastReadMessageAt: m.LastReadMessageAt().Time(),
		PinnedAt:          m.PinnedAt().Time(),
		MutedUntil:        m.MutedUntil().Time(),
		MentionCount:      m.MentionCount(),
		IsVisible:         m.IsVisible(),
		CreatedAt:         m.CreatedAt().Time(),
		UpdatedAt:         m.UpdatedAt().Time(),
	}
}

type Message struct {
	ID                 uuid.UUID       `json:"id"`
	ChannelID          uuid.UUID       `json:"channel_id"`
	AuthorID           uuid.UUID       `json:"author_id"`
	MsgType            int16           `json:"msg_type"`
	Content            json.RawMessage `json:"content"`
	SystemMetadata     json.RawMessage `json:"system_metadata"`
	ReplyToMessageID   uuid.UUID       `json:"reply_to_message_id"`
	ForwardedMessageID uuid.UUID       `json:"forwarded_message_id"`
	ForwardedChannelID uuid.UUID       `json:"forwarded_channel_id"`
	PinnedAt           time.Time       `json:"pinned_at"`
	CreatedAt          time.Time       `json:"created_at"`
	UpdatedAt          time.Time       `json:"updated_at"`
	EditedAt           time.Time       `json:"edited_at"`
}

func (m Message) ToDomain() (*channel.Message, error) {
	id, err := fields.ParseRequiredID("id", m.ID)
	if err != nil {
		return nil, err
	}

	channelID, err := fields.ParseRequiredID("channel_id", m.ChannelID)
	if err != nil {
		return nil, err
	}

	authorID, err := fields.ParseRequiredID("author_id", m.AuthorID)
	if err != nil {
		return nil, err
	}

	replyToMessageID, err := fields.ParseID("reply_to_message_id", m.ReplyToMessageID)
	if err != nil {
		return nil, err
	}

	forwardedMessageID, err := fields.ParseID("forwarded_message_id", m.ForwardedMessageID)
	if err != nil {
		return nil, err
	}

	forwardedChannelID, err := fields.ParseID("forwarded_channel_id", m.ForwardedChannelID)
	if err != nil {
		return nil, err
	}

	var content channel.MessageContent
	if len(m.Content) > 0 {
		if err := json.Unmarshal(m.Content, &content); err != nil {
			return nil, fmt.Errorf("failed to unmarshal message content: %w", err)
		}
	}

	var sysMeta fields.JSON
	if len(m.SystemMetadata) > 0 {
		var parseErr error
		sysMeta, parseErr = fields.ParseJSON("system_metadata", m.SystemMetadata)
		if parseErr != nil {
			return nil, fmt.Errorf("failed to parse system metadata: %w", parseErr)
		}
	}

	msgType, err := channel.ParseMessageType(m.MsgType)
	if err != nil {
		return nil, err
	}

	return channel.ParseMessage(
		id,
		channelID,
		authorID,
		msgType,
		content,
		sysMeta,
		replyToMessageID,
		forwardedMessageID,
		forwardedChannelID,
		fields.NewTimestamp(m.PinnedAt),
		fields.NewTimestamp(m.CreatedAt),
		fields.NewTimestamp(m.UpdatedAt),
		fields.NewTimestamp(m.EditedAt),
	), nil
}

func ParseMessage(m *channel.Message) (Message, error) {
	contentBytes, err := json.Marshal(m.Content())
	if err != nil {
		return Message{}, fmt.Errorf("failed to marshal message content: %w", err)
	}

	return Message{
		ID:                 m.ID().UUID(),
		ChannelID:          m.ChannelID().UUID(),
		AuthorID:           m.AuthorID().UUID(),
		MsgType:            m.Type().Int16(),
		Content:            contentBytes,
		SystemMetadata:     m.SystemMetadata().Bytes(),
		ReplyToMessageID:   m.ReplyToMessageID().UUID(),
		ForwardedMessageID: m.ForwardedMessageID().UUID(),
		ForwardedChannelID: m.ForwardedChannelID().UUID(),
		PinnedAt:           m.PinnedAt().Time(),
		CreatedAt:          m.CreatedAt().Time(),
		UpdatedAt:          m.UpdatedAt().Time(),
		EditedAt:           m.EditedAt().Time(),
	}, nil
}
