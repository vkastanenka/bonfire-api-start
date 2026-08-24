package channel

import (
	"bonfire-api/internal/fields"
	"slices"

	"github.com/google/uuid"
)

const (
	MessageListLimit       int = 25
	MessageListBeforeLimit int = 5
	MessageListAfterLimit  int = 20
)

type Message struct {
	id               fields.ID
	channelID        fields.ID
	authorID         fields.ID
	msgType          MessageType
	content          MessageContent
	metadata         fields.JSON
	replyToMessageID fields.ID
	forwardMessageID fields.ID
	forwardChannelID fields.ID
	pinnedAt         fields.Timestamp
	createdAt        fields.Timestamp
	updatedAt        fields.Timestamp
	editedAt         fields.Timestamp
}

func ReconstituteMessage(
	id fields.ID,
	channelID fields.ID,
	authorID fields.ID,
	msgType MessageType,
	content MessageContent,
	metadata fields.JSON,
	replyToMessageID fields.ID,
	forwardMessageID fields.ID,
	forwardChannelID fields.ID,
	pinnedAt fields.Timestamp,
	createdAt fields.Timestamp,
	updatedAt fields.Timestamp,
	editedAt fields.Timestamp,
) *Message {
	return &Message{
		id:               id,
		channelID:        channelID,
		authorID:         authorID,
		msgType:          msgType,
		content:          content,
		metadata:         metadata,
		replyToMessageID: replyToMessageID,
		forwardMessageID: forwardMessageID,
		forwardChannelID: forwardChannelID,
		pinnedAt:         pinnedAt,
		createdAt:        createdAt,
		updatedAt:        updatedAt,
		editedAt:         editedAt,
	}
}

func NewRawMessage(
	channelID,
	authorID fields.ID,
	msgType MessageType,
	content MessageContent,
	metadata fields.JSON,
	replyToMessageID,
	forwardMessageID,
	forwardChannelID fields.ID,
	now fields.Timestamp,
) (*Message, error) {
	id, err := fields.NewID()
	if err != nil {
		return nil, err
	}

	return ReconstituteMessage(
		id,
		channelID,
		authorID,
		msgType,
		content,
		metadata,
		replyToMessageID,
		forwardMessageID,
		forwardChannelID,
		fields.Timestamp{},
		now,
		now,
		fields.Timestamp{},
	), nil
}

func NewMessage(
	channelID,
	authorID fields.ID,
	content MessageContent,
	replyToMessageID,
	forwardMessageID,
	forwardChannelID fields.ID,
	now fields.Timestamp,
) (*Message, error) {
	return NewRawMessage(
		channelID,
		authorID,
		NewMessageTypeDefault(),
		content,
		fields.JSON{},
		replyToMessageID,
		forwardMessageID,
		forwardChannelID,
		now,
	)
}

func NewSystemMessage(
	channelID,
	authorID fields.ID,
	msgType MessageType,
	metadata fields.JSON,
	now fields.Timestamp,
) (*Message, error) {
	return NewRawMessage(
		channelID,
		authorID,
		msgType,
		MessageContent{},
		metadata,
		fields.ID{},
		fields.ID{},
		fields.ID{},
		now,
	)
}

func NewMessageMemberAdd(
	channelID,
	authorID,
	memberID fields.ID,
	now fields.Timestamp,
) (*Message, error) {
	metadataJSON := fields.NewJSON(map[string]any{"user_id": memberID.String()})

	return NewSystemMessage(
		channelID,
		authorID,
		NewMessageTypeMemberAdd(),
		metadataJSON,
		now,
	)
}

func NewMessageMemberLeave(
	channelID,
	authorID fields.ID,
	now fields.Timestamp,
) (*Message, error) {
	return NewSystemMessage(
		channelID,
		authorID,
		NewMessageTypeMemberRemove(),
		fields.JSON{},
		now,
	)
}

func NewMessageNameChange(
	channelID,
	authorID fields.ID,
	newName ChannelName,
	now fields.Timestamp,
) (*Message, error) {
	metadataJSON := fields.NewJSON(map[string]any{"name": newName.String()})

	return NewSystemMessage(
		channelID,
		authorID,
		NewMessageTypeNameChange(),
		metadataJSON,
		now,
	)
}

func NewMessageIconChange(
	channelID,
	authorID fields.ID,
	now fields.Timestamp,
) (*Message, error) {
	return NewSystemMessage(
		channelID,
		authorID,
		NewMessageTypeIconChange(),
		fields.JSON{},
		now,
	)
}

func NewMessagePin(
	channelID,
	authorID,
	pinnedMessageID fields.ID,
	now fields.Timestamp,
) (*Message, error) {
	metadataJSON := fields.NewJSON(map[string]any{"message_id": pinnedMessageID.String()})

	return NewSystemMessage(
		channelID,
		authorID,
		NewMessageTypePin(),
		metadataJSON,
		now,
	)
}

func (m *Message) ID() fields.ID               { return m.id }
func (m *Message) ChannelID() fields.ID        { return m.channelID }
func (m *Message) AuthorID() fields.ID         { return m.authorID }
func (m *Message) Type() MessageType           { return m.msgType }
func (m *Message) Content() MessageContent     { return m.content }
func (m *Message) Metadata() fields.JSON       { return m.metadata }
func (m *Message) ReplyToMessageID() fields.ID { return m.replyToMessageID }
func (m *Message) ForwardMessageID() fields.ID { return m.forwardMessageID }
func (m *Message) ForwardChannelID() fields.ID { return m.forwardChannelID }
func (m *Message) PinnedAt() fields.Timestamp  { return m.pinnedAt }
func (m *Message) CreatedAt() fields.Timestamp { return m.createdAt }
func (m *Message) UpdatedAt() fields.Timestamp { return m.updatedAt }
func (m *Message) EditedAt() fields.Timestamp  { return m.editedAt }

func getMessageIDs(messages []*Message) ([]fields.ID, []fields.ID) {
	msgIDs := make([]fields.ID, 0, len(messages))
	authorIDs := make([]fields.ID, 0, len(messages))
	for _, m := range messages {
		msgIDs = append(msgIDs, m.ID())
		if m.authorID.IsValid() {
			authorIDs = append(authorIDs, m.AuthorID())
		}
	}
	return msgIDs, fields.DedupeIDs(authorIDs)
}

func sortMessages(messages []*Message) {
	slices.SortFunc(messages, func(a, b *Message) int {
		return a.ID().Compare(b.ID())
	})
}

func sortPinnedMessages(messages []*Message) {
	slices.SortFunc(messages, func(a, b *Message) int {
		return b.ID().Compare(a.ID())
	})
}

func validateReply(hasReply bool, hasFwdMsg, hasFwdChan bool) error {
	if hasReply && (hasFwdMsg || hasFwdChan) {
		return ErrMessageReplyConflict()
	}
	return nil
}

func validateForward(hasFwdMsg, hasFwdChan bool) error {
	if hasFwdMsg != hasFwdChan {
		return ErrMessageForwardIncomplete()
	}
	return nil
}

func validateMessageIDs(rawActorID, rawChannelID, rawMsgID uuid.UUID) (fields.ID, fields.ID, fields.ID, error) {
	actorID, channelID, err := validateIDs(rawActorID, rawChannelID)
	if err != nil {
		return fields.ID{}, fields.ID{}, fields.ID{}, err
	}

	msgID, err := fields.ParseRequiredID("message_id", rawMsgID)
	if err != nil {
		return fields.ID{}, fields.ID{}, fields.ID{}, err
	}

	return actorID, channelID, msgID, nil
}
