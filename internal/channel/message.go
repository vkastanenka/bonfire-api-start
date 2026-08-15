package channel

import (
	"bonfire-api/internal/fields"
)

type Message struct {
	id                 fields.ID
	channelID          fields.ID
	authorID           fields.ID
	msgType            MessageType
	content            MessageContent
	systemMetadata     fields.JSON
	replyToMessageID   fields.ID
	forwardedMessageID fields.ID
	forwardedChannelID fields.ID
	pinnedAt           fields.Timestamp
	createdAt          fields.Timestamp
	updatedAt          fields.Timestamp
	editedAt           fields.Timestamp
}

func (m *Message) ID() fields.ID                 { return m.id }
func (m *Message) ChannelID() fields.ID          { return m.channelID }
func (m *Message) AuthorID() fields.ID           { return m.authorID }
func (m *Message) Type() MessageType             { return m.msgType }
func (m *Message) Content() MessageContent       { return m.content }
func (m *Message) SystemMetadata() fields.JSON   { return m.systemMetadata }
func (m *Message) ReplyToMessageID() fields.ID   { return m.replyToMessageID }
func (m *Message) ForwardedMessageID() fields.ID { return m.forwardedMessageID }
func (m *Message) ForwardedChannelID() fields.ID { return m.forwardedChannelID }
func (m *Message) PinnedAt() fields.Timestamp    { return m.pinnedAt }
func (m *Message) CreatedAt() fields.Timestamp   { return m.createdAt }
func (m *Message) UpdatedAt() fields.Timestamp   { return m.updatedAt }
func (m *Message) EditedAt() fields.Timestamp    { return m.editedAt }

func ParseMessage(
	id fields.ID,
	channelID fields.ID,
	authorID fields.ID,
	msgType MessageType,
	content MessageContent,
	systemMetadata fields.JSON,
	replyToMessageID fields.ID,
	forwardedMessageID fields.ID,
	forwardedChannelID fields.ID,
	pinnedAt fields.Timestamp,
	createdAt fields.Timestamp,
	updatedAt fields.Timestamp,
	editedAt fields.Timestamp,
) *Message {
	return &Message{
		id:                 id,
		channelID:          channelID,
		authorID:           authorID,
		msgType:            msgType,
		content:            content,
		systemMetadata:     systemMetadata,
		replyToMessageID:   replyToMessageID,
		forwardedMessageID: forwardedMessageID,
		forwardedChannelID: forwardedChannelID,
		pinnedAt:           pinnedAt,
		createdAt:          createdAt,
		updatedAt:          updatedAt,
		editedAt:           editedAt,
	}
}

func (m *Message) SetAuthorID(id fields.ID, now fields.Timestamp) {
	m.authorID = id
	m.touch(now)
}

func (m *Message) SetReplyToMessageID(id fields.ID, now fields.Timestamp) {
	m.replyToMessageID = id
	m.touch(now)
}

func (m *Message) SetForwardedMessage(messageID, channelID fields.ID, now fields.Timestamp) {
	m.forwardedMessageID = messageID
	m.forwardedChannelID = channelID
	m.touch(now)
}

func (m *Message) SetPinnedAt(pinnedAt, now fields.Timestamp) {
	m.pinnedAt = pinnedAt
	m.touch(now)
}

func (m *Message) SetContent(content MessageContent, now fields.Timestamp) {
	m.content = content
	m.editedAt = now
	m.touch(now)
}

func (m *Message) SetSystemMetadata(systemMetadata fields.JSON, now fields.Timestamp) {
	m.systemMetadata = systemMetadata
	m.touch(now)
}

func (m *Message) touch(at fields.Timestamp) {
	m.updatedAt = at
}
