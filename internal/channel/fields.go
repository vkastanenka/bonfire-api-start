package channel

import (
	"unicode/utf8"

	"bonfire-api/internal/errs"
	"bonfire-api/internal/fields"
	"bonfire-api/internal/pkg/ptr"
	"bonfire-api/internal/sanitize"
)

// -----------------------------------------------------------------------------
// Channel Name
// -----------------------------------------------------------------------------

const channelNameMaxLength = 100

type ChannelName struct {
	value fields.Text
}

func ParseChannelName(raw *string) (*ChannelName, error) {
	if raw == nil {
		return nil, nil
	}

	cleaned := sanitize.Text(ptr.From(raw))
	if cleaned == "" {
		return nil, nil
	}

	if utf8.RuneCountInString(cleaned) > channelNameMaxLength {
		return nil, errs.InvalidArgument("Name too long.").
			Reason("NAME_TOO_LONG").
			FieldViolation("name", "Name must be 100 characters or fewer.", "MAX_LENGTH_EXCEEDED").
			Meta("domain", "channels")
	}

	return &ChannelName{value: fields.NewText(cleaned)}, nil
}

// -----------------------------------------------------------------------------
// Channel Type
// -----------------------------------------------------------------------------

type ChannelTypeValue uint8

const (
	ChannelTypeUnknown ChannelTypeValue = iota
	ChannelTypeDirect
	ChannelTypeGroup
	channelTypeMax
)

var channelTypeSpec = &fields.EnumSpec{
	Domain: "CHANNEL_TYPE",
	Max:    uint8(channelTypeMax),
	Names:  []string{"UNKNOWN", "DIRECT", "GROUP"},
	Bytes:  [][]byte{[]byte("UNKNOWN"), []byte("DIRECT"), []byte("GROUP")},
}

type ChannelType struct {
	fields.Enum[ChannelTypeValue]
}

func ErrChannelInvalidType() *errs.Error {
	return errs.InvalidArgument("Invalid channel type.").
		Reason("CHANNEL_TYPE_INVALID").
		FieldViolation("type", "Must be one of: DIRECT, GROUP.", "INVALID_ENUM_VALUE").
		Meta("domain", "channels")
}

func ParseChannelType(raw int16) (ChannelType, error) {
	if raw <= 0 || raw >= int16(channelTypeMax) {
		return ChannelType{}, ErrChannelInvalidType()
	}
	return ChannelType{Enum: fields.NewEnum(ChannelTypeValue(raw), channelTypeSpec)}, nil
}

func ParseChannelTypeString(s string) (ChannelType, error) {
	kind, ok := fields.ParseEnumString[ChannelTypeValue](s, channelTypeSpec)
	if !ok || kind >= channelTypeMax {
		return ChannelType{}, ErrChannelInvalidType()
	}
	return ChannelType{Enum: fields.NewEnum(kind, channelTypeSpec)}, nil
}

// -----------------------------------------------------------------------------
// Message Content
// -----------------------------------------------------------------------------

const messageContentMaxLength = 4000

type MessageContent struct {
	value fields.Text
}

func ParseMessageContent(raw *string) (*MessageContent, error) {
	if raw == nil {
		return nil, nil
	}

	cleaned := sanitize.Text(ptr.From(raw))
	if cleaned == "" {
		return nil, nil
	}

	if utf8.RuneCountInString(cleaned) > messageContentMaxLength {
		return nil, errs.InvalidArgument("Content too long.").
			Reason("CONTENT_TOO_LONG").
			FieldViolation("content", "Content must be 4000 characters or fewer.", "MAX_LENGTH_EXCEEDED").
			Meta("domain", "messages")
	}

	return &MessageContent{value: fields.NewText(cleaned)}, nil
}

// -----------------------------------------------------------------------------
// Message Type
// -----------------------------------------------------------------------------

type MessageTypeValue uint8

const (
	MessageTypeUnknown MessageTypeValue = iota
	MessageTypeDefault
	MessageTypeReply
	MessageTypeForward
	MessageTypeMemberAdd
	MessageTypeMemberRemove
	MessageTypeNameChange
	MessageTypeIconChange
	MessageTypePin
	messageTypeMax
)

var messageTypeSpec = &fields.EnumSpec{
	Domain: "MESSAGE_TYPE",
	Max:    uint8(messageTypeMax),
	Names: []string{
		"UNKNOWN",
		"DEFAULT",
		"REPLY",
		"FORWARD",
		"MEMBER_ADD",
		"MEMBER_REMOVE",
		"NAME_CHANGE",
		"ICON_CHANGE",
		"PIN",
	},
	Bytes: [][]byte{
		[]byte("UNKNOWN"),
		[]byte("DEFAULT"),
		[]byte("REPLY"),
		[]byte("FORWARD"),
		[]byte("MEMBER_ADD"),
		[]byte("MEMBER_REMOVE"),
		[]byte("NAME_CHANGE"),
		[]byte("ICON_CHANGE"),
		[]byte("PIN"),
	},
}

type MessageType struct {
	fields.Enum[MessageTypeValue]
}

func ErrMessageInvalidType() *errs.Error {
	return errs.InvalidArgument("Invalid message type.").
		Reason("MESSAGE_TYPE_INVALID").
		FieldViolation("type", "Must be a valid message type.", "INVALID_ENUM_VALUE").
		Meta("domain", "messages")
}

func ParseMessageType(raw int16) (MessageType, error) {
	if raw <= 0 || raw >= int16(messageTypeMax) {
		return MessageType{}, ErrMessageInvalidType()
	}
	return MessageType{Enum: fields.NewEnum(MessageTypeValue(raw), messageTypeSpec)}, nil
}

func ParseMessageTypeString(s string) (MessageType, error) {
	kind, ok := fields.ParseEnumString[MessageTypeValue](s, messageTypeSpec)
	if !ok || kind >= messageTypeMax {
		return MessageType{}, ErrMessageInvalidType()
	}
	return MessageType{Enum: fields.NewEnum(kind, messageTypeSpec)}, nil
}

// -----------------------------------------------------------------------------
// Reaction Emoji
// -----------------------------------------------------------------------------

const reactionEmojiMaxLength = 64

var errReactionEmojiRequired = errs.InvalidArgument("Invalid value.").
	Reason("EMOJI_REQUIRED").
	FieldViolation("emoji", "Emoji cannot be empty.", "REQUIRED").Meta("domain", "reactions")

type ReactionEmoji struct {
	value fields.Text
}

func ParseReactionEmoji(raw *string) (*ReactionEmoji, error) {
	if raw == nil {
		return nil, errReactionEmojiRequired
	}

	cleaned := sanitize.Text(ptr.From(raw))
	if cleaned == "" {
		return nil, errReactionEmojiRequired
	}

	if utf8.RuneCountInString(cleaned) > reactionEmojiMaxLength {
		return nil, errs.InvalidArgument("Emoji too long.").
			Reason("EMOJI_TOO_LONG").
			FieldViolation("emoji", "Emoji must be 64 characters or fewer.", "MAX_LENGTH_EXCEEDED").
			Meta("domain", "reactions")
	}

	return &ReactionEmoji{value: fields.NewText(cleaned)}, nil
}
