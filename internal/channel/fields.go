package channel

import (
	"unicode/utf8"

	"bonfire-api/internal/errs"
	"bonfire-api/internal/fields"
	"bonfire-api/internal/sanitize"
)

// -----------------------------------------------------------------------------
// Channel Name
// -----------------------------------------------------------------------------

const channelNameMaxLength = 100

type ChannelName struct {
	fields.Text
}

func NewChannelName(v string) ChannelName {
	return ChannelName{Text: fields.NewText(v)}
}

func ParseChannelName(raw string) (ChannelName, error) {
	cleaned := sanitize.Text(raw)
	if cleaned == "" {
		return ChannelName{}, nil
	}

	if utf8.RuneCountInString(cleaned) > channelNameMaxLength {
		return ChannelName{}, errs.InvalidArgument("Name too long.").
			Reason("NAME_TOO_LONG").
			FieldViolation("name", "Name must be 100 characters or fewer.", "MAX_LENGTH_EXCEEDED").
			Meta("domain", "channels")
	}

	return NewChannelName(cleaned), nil
}

func ParseRequiredChannelName(raw string) (ChannelName, error) {
	name, err := ParseChannelName(raw)
	if err != nil {
		return ChannelName{}, err
	}
	if name.IsZero() {
		return ChannelName{}, errs.InvalidArgument("Channel name is required.").
			Reason("NAME_REQUIRED").
			FieldViolation("name", "Field is required.", "REQUIRED").
			Meta("domain", "channels")
	}
	return name, nil
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

func NewChannelType(val ChannelTypeValue) ChannelType {
	return ChannelType{Enum: fields.NewEnum(val, channelTypeSpec)}
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
	return NewChannelType(ChannelTypeValue(raw)), nil
}

func ParseChannelTypeString(s string) (ChannelType, error) {
	kind, ok := fields.ParseEnumString[ChannelTypeValue](s, channelTypeSpec)
	if !ok || kind >= channelTypeMax {
		return ChannelType{}, ErrChannelInvalidType()
	}
	return NewChannelType(kind), nil
}

// -----------------------------------------------------------------------------
// Message Content
// -----------------------------------------------------------------------------

const messageContentMaxLength = 4000

type MessageContent struct {
	fields.Text
}

func NewMessageContent(v string) MessageContent {
	return MessageContent{Text: fields.NewText(v)}
}

func ParseMessageContent(raw string) (MessageContent, error) {
	cleaned := sanitize.Text(raw)
	if cleaned == "" {
		return MessageContent{}, nil
	}

	if utf8.RuneCountInString(cleaned) > messageContentMaxLength {
		return MessageContent{}, errs.InvalidArgument("Content too long.").
			Reason("CONTENT_TOO_LONG").
			FieldViolation("content", "Content must be 4000 characters or fewer.", "MAX_LENGTH_EXCEEDED").
			Meta("domain", "messages")
	}

	return NewMessageContent(cleaned), nil
}

func ParseRequiredMessageContent(raw string) (MessageContent, error) {
	content, err := ParseMessageContent(raw)
	if err != nil {
		return MessageContent{}, err
	}
	if content.IsZero() {
		return MessageContent{}, errs.InvalidArgument("Message content is required.").
			Reason("CONTENT_REQUIRED").
			FieldViolation("content", "Field is required.", "REQUIRED").
			Meta("domain", "messages")
	}
	return content, nil
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

func NewMessageType(val MessageTypeValue) MessageType {
	return MessageType{Enum: fields.NewEnum(val, messageTypeSpec)}
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
	return NewMessageType(MessageTypeValue(raw)), nil
}

func ParseMessageTypeString(s string) (MessageType, error) {
	kind, ok := fields.ParseEnumString[MessageTypeValue](s, messageTypeSpec)
	if !ok || kind >= messageTypeMax {
		return MessageType{}, ErrMessageInvalidType()
	}
	return NewMessageType(kind), nil
}

// -----------------------------------------------------------------------------
// Reaction Emoji
// -----------------------------------------------------------------------------

const reactionEmojiMaxLength = 64

type ReactionEmoji struct {
	fields.Text
}

func NewReactionEmoji(v string) ReactionEmoji {
	return ReactionEmoji{Text: fields.NewText(v)}
}

func ParseReactionEmoji(raw string) (ReactionEmoji, error) {
	cleaned := sanitize.Text(raw)
	if cleaned == "" {
		return ReactionEmoji{}, nil
	}

	if utf8.RuneCountInString(cleaned) > reactionEmojiMaxLength {
		return ReactionEmoji{}, errs.InvalidArgument("Emoji too long.").
			Reason("EMOJI_TOO_LONG").
			FieldViolation("emoji", "Emoji must be 64 characters or fewer.", "MAX_LENGTH_EXCEEDED").
			Meta("domain", "reactions")
	}

	return NewReactionEmoji(cleaned), nil
}

func ParseRequiredReactionEmoji(raw string) (ReactionEmoji, error) {
	emoji, err := ParseReactionEmoji(raw)
	if err != nil {
		return ReactionEmoji{}, err
	}
	if emoji.IsZero() {
		return ReactionEmoji{}, errs.InvalidArgument("Emoji is required.").
			Reason("EMOJI_REQUIRED").
			FieldViolation("emoji", "Emoji cannot be empty.", "REQUIRED").
			Meta("domain", "reactions")
	}
	return emoji, nil
}
