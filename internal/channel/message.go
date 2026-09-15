package channel

import (
	"encoding/json"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	MessageListLimit       int = 25
	MessageListBeforeLimit int = 5
	MessageListAfterLimit  int = 20
)

type Message struct {
	ID               uuid.UUID
	ChannelID        uuid.UUID
	AuthorID         *uuid.UUID
	Type             MessageType
	Content          *string
	Metadata         json.RawMessage
	ReplyToMessageID *uuid.UUID
	ForwardMessageID *uuid.UUID
	ForwardChannelID *uuid.UUID
	PinnedAt         *time.Time
	CreatedAt        time.Time
	UpdatedAt        time.Time
	EditedAt         *time.Time
}

func ReconstituteMessage(
	id uuid.UUID,
	channelID uuid.UUID,
	authorID *uuid.UUID,
	msgType MessageType,
	content *string,
	metadata json.RawMessage,
	replyToMessageID *uuid.UUID,
	forwardMessageID *uuid.UUID,
	forwardChannelID *uuid.UUID,
	pinnedAt *time.Time,
	editedAt *time.Time,
	createdAt time.Time,
	updatedAt time.Time,
) *Message {
	return &Message{
		ID:               id,
		ChannelID:        channelID,
		AuthorID:         authorID,
		Type:             msgType,
		Content:          content,
		Metadata:         metadata,
		ReplyToMessageID: replyToMessageID,
		ForwardMessageID: forwardMessageID,
		ForwardChannelID: forwardChannelID,
		PinnedAt:         pinnedAt,
		EditedAt:         editedAt,
		CreatedAt:        createdAt,
		UpdatedAt:        updatedAt,
	}
}

func NewRawMessage(
	channelID uuid.UUID,
	authorID *uuid.UUID,
	msgType MessageType,
	content *string,
	metadata json.RawMessage,
	replyToMessageID *uuid.UUID,
	forwardMessageID *uuid.UUID,
	forwardChannelID *uuid.UUID,
	now time.Time,
) (*Message, error) {
	id, err := uuid.NewV7()
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
		nil,
		nil,
		now,
		now,
	), nil
}

func NewMessage(
	channelID uuid.UUID,
	authorID uuid.UUID,
	content string,
	replyToMessageID *uuid.UUID,
	forwardMessageID *uuid.UUID,
	forwardChannelID *uuid.UUID,
	now time.Time,
) (*Message, error) {
	return NewRawMessage(
		channelID,
		&authorID,
		MessageTypeDefault,
		&content,
		nil,
		replyToMessageID,
		forwardMessageID,
		forwardChannelID,
		now,
	)
}

func NewSystemMessage(
	channelID uuid.UUID,
	authorID *uuid.UUID,
	msgType MessageType,
	metadata json.RawMessage,
	now time.Time,
) (*Message, error) {
	return NewRawMessage(
		channelID,
		authorID,
		msgType,
		nil,
		metadata,
		nil,
		nil,
		nil,
		now,
	)
}

func NewMessageMemberAdd(
	channelID uuid.UUID,
	authorID *uuid.UUID,
	memberID uuid.UUID,
	now time.Time,
) (*Message, error) {
	metadata, err := json.Marshal(map[string]any{"user_id": memberID.String()})
	if err != nil {
		return nil, err
	}

	return NewSystemMessage(
		channelID,
		authorID,
		MessageTypeMemberAdd,
		metadata,
		now,
	)
}

func NewMessageMemberLeave(
	channelID uuid.UUID,
	authorID *uuid.UUID,
	now time.Time,
) (*Message, error) {
	return NewSystemMessage(
		channelID,
		authorID,
		MessageTypeMemberRemove,
		nil,
		now,
	)
}

func NewMessageNameChange(
	channelID uuid.UUID,
	authorID *uuid.UUID,
	newName *string,
	now time.Time,
) (*Message, error) {
	metadata, err := json.Marshal(map[string]any{"name": newName})
	if err != nil {
		return nil, err
	}

	return NewSystemMessage(
		channelID,
		authorID,
		MessageTypeNameChange,
		metadata,
		now,
	)
}

func NewMessageIconChange(
	channelID uuid.UUID,
	authorID *uuid.UUID,
	now time.Time,
) (*Message, error) {
	return NewSystemMessage(
		channelID,
		authorID,
		MessageTypeIconChange,
		nil,
		now,
	)
}

func NewMessagePin(
	channelID uuid.UUID,
	authorID *uuid.UUID,
	pinnedMessageID uuid.UUID,
	now time.Time,
) (*Message, error) {
	metadata, err := json.Marshal(map[string]any{"message_id": pinnedMessageID.String()})
	if err != nil {
		return nil, err
	}

	return NewSystemMessage(
		channelID,
		authorID,
		MessageTypePin,
		metadata,
		now,
	)
}

func getMessageIDs(messages []*Message) ([]uuid.UUID, []uuid.UUID) {
	msgIDs := make([]uuid.UUID, 0, len(messages))
	authorIDs := make([]uuid.UUID, 0, len(messages))
	seenAuthors := make(map[uuid.UUID]struct{}, len(messages))

	for _, m := range messages {
		if m == nil {
			continue
		}
		msgIDs = append(msgIDs, m.ID)
		if m.AuthorID != nil && *m.AuthorID != (uuid.UUID{}) {
			if _, exists := seenAuthors[*m.AuthorID]; !exists {
				seenAuthors[*m.AuthorID] = struct{}{}
				authorIDs = append(authorIDs, *m.AuthorID)
			}
		}
	}
	return msgIDs, authorIDs
}

func sortMessages(messages []*Message) {
	slices.SortFunc(messages, func(a, b *Message) int {
		if a == nil && b == nil {
			return 0
		}
		if a == nil {
			return -1
		}
		if b == nil {
			return 1
		}
		return strings.Compare(a.ID.String(), b.ID.String())
	})
}

func sortPinnedMessages(messages []*Message) {
	slices.SortFunc(messages, func(a, b *Message) int {
		if a == nil && b == nil {
			return 0
		}
		if a == nil {
			return 1
		}
		if b == nil {
			return -1
		}
		return strings.Compare(b.ID.String(), a.ID.String())
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
