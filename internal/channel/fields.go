package channel

import (
	"time"
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

func (t ChannelType) IsDirect() bool {
	return t.Value == uint8(ChannelTypeDirect)
}

func (t ChannelType) IsGroup() bool {
	return t.Value == uint8(ChannelTypeGroup)
}

func NewChannelType(val ChannelTypeValue) ChannelType {
	return ChannelType{Enum: fields.NewEnum(val, channelTypeSpec)}
}

func NewChannelTypeDirect() ChannelType {
	return NewChannelType(ChannelTypeDirect)
}

func NewChannelTypeGroup() ChannelType {
	return NewChannelType(ChannelTypeGroup)
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

func (t MessageType) IsDefault() bool {
	return t.Value == uint8(MessageTypeDefault)
}

func (t MessageType) IsReply() bool {
	return t.Value == uint8(MessageTypeReply)
}

func (t MessageType) IsForward() bool {
	return t.Value == uint8(MessageTypeForward)
}

func (t MessageType) IsMemberAdd() bool {
	return t.Value == uint8(MessageTypeMemberAdd)
}

func (t MessageType) IsMemberRemove() bool {
	return t.Value == uint8(MessageTypeMemberRemove)
}

func (t MessageType) IsNameChange() bool {
	return t.Value == uint8(MessageTypeNameChange)
}

func (t MessageType) IsIconChange() bool {
	return t.Value == uint8(MessageTypeIconChange)
}

func (t MessageType) IsPin() bool {
	return t.Value == uint8(MessageTypePin)
}

func NewMessageType(val MessageTypeValue) MessageType {
	return MessageType{Enum: fields.NewEnum(val, messageTypeSpec)}
}

func NewMessageTypeDefault() MessageType {
	return NewMessageType(MessageTypeDefault)
}

func NewMessageTypeReply() MessageType {
	return NewMessageType(MessageTypeReply)
}

func NewMessageTypeForward() MessageType {
	return NewMessageType(MessageTypeForward)
}

func NewMessageTypeMemberAdd() MessageType {
	return NewMessageType(MessageTypeMemberAdd)
}

func NewMessageTypeMemberRemove() MessageType {
	return NewMessageType(MessageTypeMemberRemove)
}

func NewMessageTypeNameChange() MessageType {
	return NewMessageType(MessageTypeNameChange)
}

func NewMessageTypeIconChange() MessageType {
	return NewMessageType(MessageTypeIconChange)
}

func NewMessageTypePin() MessageType {
	return NewMessageType(MessageTypePin)
}

func ErrMessageTypeInvalid() *errs.Error {
	return errs.InvalidArgument("Invalid message type.").
		Reason("MESSAGE_TYPE_INVALID").
		FieldViolation("type", "Must be a valid message type.", "INVALID_ENUM_VALUE").
		Meta("domain", "messages")
}

func ParseMessageType(raw int16) (MessageType, error) {
	if raw <= 0 || raw >= int16(messageTypeMax) {
		return MessageType{}, ErrMessageTypeInvalid()
	}
	return NewMessageType(MessageTypeValue(raw)), nil
}

func ParseMessageTypeString(s string) (MessageType, error) {
	kind, ok := fields.ParseEnumString[MessageTypeValue](s, messageTypeSpec)
	if !ok || kind >= messageTypeMax {
		return MessageType{}, ErrMessageTypeInvalid()
	}
	return NewMessageType(kind), nil
}

// -----------------------------------------------------------------------------
// Mute Duration
// -----------------------------------------------------------------------------

type MuteDurationValue uint8

const (
	MuteDurationUnknown MuteDurationValue = iota
	MuteDuration15Min
	MuteDuration1Hour
	MuteDuration8Hours
	MuteDuration24Hours
	MuteDuration3Days
	MuteDurationForever
	muteDurationMax
)

var muteDurationSpec = &fields.EnumSpec{
	Domain: "MUTE_DURATION",
	Max:    uint8(muteDurationMax),
	Names:  []string{"UNKNOWN", "15_MIN", "1_HOUR", "8_HOURS", "24_HOURS", "3_DAYS", "FOREVER"},
	Bytes:  [][]byte{[]byte("UNKNOWN"), []byte("15_MIN"), []byte("1_HOUR"), []byte("8_HOURS"), []byte("24_HOURS"), []byte("3_DAYS"), []byte("FOREVER")},
}

type MuteDuration struct {
	fields.Enum[MuteDurationValue]
}

func NewMuteDuration(val MuteDurationValue) MuteDuration {
	return MuteDuration{Enum: fields.NewEnum(val, muteDurationSpec)}
}

func ErrMuteDurationInvalid() *errs.Error {
	return errs.InvalidArgument("Invalid mute duration.").
		Reason("MUTE_DURATION_INVALID").
		FieldViolation("mute_duration", "Must be one of: 15_MIN, 1_HOUR, 8_HOURS, 24_HOURS, 3_DAYS, FOREVER.", "INVALID_ENUM_VALUE").
		Meta("domain", "members")
}

func ParseMuteDuration(raw uint8) (MuteDuration, error) {
	if raw <= 0 || raw >= uint8(muteDurationMax) {
		return MuteDuration{}, ErrMuteDurationInvalid()
	}
	return NewMuteDuration(MuteDurationValue(raw)), nil
}

func ParseMuteDurationString(s string) (MuteDuration, error) {
	kind, ok := fields.ParseEnumString[MuteDurationValue](s, muteDurationSpec)
	if !ok || kind >= muteDurationMax {
		return MuteDuration{}, ErrMuteDurationInvalid()
	}
	return NewMuteDuration(kind), nil
}

// ToDuration converts the enum value into a actual time.Duration or handles forever.
func (m MuteDuration) ToDuration() (time.Duration, bool) {
	switch MuteDurationValue(m.Value) {
	case MuteDuration15Min:
		return 15 * time.Minute, true
	case MuteDuration1Hour:
		return time.Hour, true
	case MuteDuration8Hours:
		return 8 * time.Hour, true
	case MuteDuration24Hours:
		return 24 * time.Hour, true
	case MuteDuration3Days:
		return 72 * time.Hour, true
	case MuteDurationForever:
		return 0, true
	default:
		return 0, false
	}
}

// CalculateUntil computes the absolute timestamp based on "now".
func (m MuteDuration) CalculateUntil(now fields.Timestamp) (fields.Timestamp, error) {
	if MuteDurationValue(m.Value) == MuteDurationForever {
		// Far-future convention for "forever"
		farFuture := time.Date(9999, 12, 31, 23, 59, 59, 0, time.UTC)
		return fields.NewTimestamp(farFuture), nil
	}

	d, ok := m.ToDuration()
	if !ok {
		return fields.Timestamp{}, ErrMuteDurationInvalid()
	}

	return fields.NewTimestamp(now.Time().Add(d)), nil
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
