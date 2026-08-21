package channel

import (
	"time"
	"unicode/utf8"

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
		return ChannelName{}, ErrChannelNameTooLong()
	}

	return NewChannelName(cleaned), nil
}

func ParseRequiredChannelName(raw string) (ChannelName, error) {
	name, err := ParseChannelName(raw)
	if err != nil {
		return ChannelName{}, err
	}
	if name.IsZero() {
		return ChannelName{}, ErrChannelNameRequired()
	}
	return name, nil
}

// -----------------------------------------------------------------------------
// Channel Type
// -----------------------------------------------------------------------------

type ChannelTypeValue int16

const (
	ChannelTypeUnknown ChannelTypeValue = iota
	ChannelTypeDirect
	ChannelTypeGroup
	channelTypeMax
)

var channelTypeSpec = &fields.EnumSpec{
	Domain: "CHANNEL_TYPE",
	Max:    int(channelTypeMax),
	Names:  []string{"UNKNOWN", "DIRECT", "GROUP"},
	Bytes:  [][]byte{[]byte("UNKNOWN"), []byte("DIRECT"), []byte("GROUP")},
}

type ChannelType struct {
	fields.Enum[ChannelTypeValue]
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

func ParseChannelType[T fields.IntegerType](raw T) (ChannelType, error) {
	val := ChannelTypeValue(raw)
	if val <= ChannelTypeUnknown || int(val) >= channelTypeSpec.Max {
		return ChannelType{}, ErrChannelTypeInvalid()
	}
	return NewChannelType(val), nil
}

func ParseChannelTypeString(s string) (ChannelType, error) {
	val, ok := fields.ParseEnumString[ChannelTypeValue](s, channelTypeSpec)
	if !ok || val <= ChannelTypeUnknown {
		return ChannelType{}, ErrChannelTypeInvalid()
	}
	return NewChannelType(val), nil
}

func (t ChannelType) IsDirect() bool { return t.Is(ChannelTypeDirect) }
func (t ChannelType) IsGroup() bool  { return t.Is(ChannelTypeGroup) }

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
		return MessageContent{}, ErrMessageContentTooLong()
	}

	return NewMessageContent(cleaned), nil
}

func ParseRequiredMessageContent(raw string) (MessageContent, error) {
	content, err := ParseMessageContent(raw)
	if err != nil {
		return MessageContent{}, err
	}
	if content.IsZero() {
		return MessageContent{}, ErrMessageContentRequired()
	}
	return content, nil
}

// -----------------------------------------------------------------------------
// Message Type
// -----------------------------------------------------------------------------

type MessageTypeValue int

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
	Max:    int(messageTypeMax),
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

func NewMessageTypeDefault() MessageType      { return NewMessageType(MessageTypeDefault) }
func NewMessageTypeReply() MessageType        { return NewMessageType(MessageTypeReply) }
func NewMessageTypeForward() MessageType      { return NewMessageType(MessageTypeForward) }
func NewMessageTypeMemberAdd() MessageType    { return NewMessageType(MessageTypeMemberAdd) }
func NewMessageTypeMemberRemove() MessageType { return NewMessageType(MessageTypeMemberRemove) }
func NewMessageTypeNameChange() MessageType   { return NewMessageType(MessageTypeNameChange) }
func NewMessageTypeIconChange() MessageType   { return NewMessageType(MessageTypeIconChange) }
func NewMessageTypePin() MessageType          { return NewMessageType(MessageTypePin) }

func (t MessageType) IsDefault() bool      { return t.Is(MessageTypeDefault) }
func (t MessageType) IsReply() bool        { return t.Is(MessageTypeReply) }
func (t MessageType) IsForward() bool      { return t.Is(MessageTypeForward) }
func (t MessageType) IsMemberAdd() bool    { return t.Is(MessageTypeMemberAdd) }
func (t MessageType) IsMemberRemove() bool { return t.Is(MessageTypeMemberRemove) }
func (t MessageType) IsNameChange() bool   { return t.Is(MessageTypeNameChange) }
func (t MessageType) IsIconChange() bool   { return t.Is(MessageTypeIconChange) }
func (t MessageType) IsPin() bool          { return t.Is(MessageTypePin) }

func ParseMessageType[T fields.IntegerType](raw T) (MessageType, error) {
	val := MessageTypeValue(raw)
	if val <= MessageTypeUnknown || int(val) >= messageTypeSpec.Max {
		return MessageType{}, ErrMessageTypeInvalid()
	}
	return NewMessageType(val), nil
}

// String Parser
func ParseMessageTypeString(s string) (MessageType, error) {
	val, ok := fields.ParseEnumString[MessageTypeValue](s, messageTypeSpec)
	if !ok || val <= MessageTypeUnknown {
		return MessageType{}, ErrMessageTypeInvalid()
	}
	return NewMessageType(val), nil
}

// -----------------------------------------------------------------------------
// Mute Duration
// -----------------------------------------------------------------------------

type MuteDurationValue int

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
	Max:    int(muteDurationMax),
	Names:  []string{"UNKNOWN", "15_MIN", "1_HOUR", "8_HOURS", "24_HOURS", "3_DAYS", "FOREVER"},
	Bytes:  [][]byte{[]byte("UNKNOWN"), []byte("15_MIN"), []byte("1_HOUR"), []byte("8_HOURS"), []byte("24_HOURS"), []byte("3_DAYS"), []byte("FOREVER")},
}

type MuteDuration struct {
	fields.Enum[MuteDurationValue]
}

func NewMuteDuration(val MuteDurationValue) MuteDuration {
	return MuteDuration{Enum: fields.NewEnum(val, muteDurationSpec)}
}

func NewMuteDuration15Min() MuteDuration   { return NewMuteDuration(MuteDuration15Min) }
func NewMuteDuration1Hour() MuteDuration   { return NewMuteDuration(MuteDuration1Hour) }
func NewMuteDuration8Hours() MuteDuration  { return NewMuteDuration(MuteDuration8Hours) }
func NewMuteDuration24Hours() MuteDuration { return NewMuteDuration(MuteDuration24Hours) }
func NewMuteDuration3Days() MuteDuration   { return NewMuteDuration(MuteDuration3Days) }
func NewMuteDurationForever() MuteDuration { return NewMuteDuration(MuteDurationForever) }

func (m MuteDuration) Is15Min() bool   { return m.Is(MuteDuration15Min) }
func (m MuteDuration) Is1Hour() bool   { return m.Is(MuteDuration1Hour) }
func (m MuteDuration) Is8Hours() bool  { return m.Is(MuteDuration8Hours) }
func (m MuteDuration) Is24Hours() bool { return m.Is(MuteDuration24Hours) }
func (m MuteDuration) Is3Days() bool   { return m.Is(MuteDuration3Days) }
func (m MuteDuration) IsForever() bool { return m.Is(MuteDurationForever) }

func ParseMuteDuration[T fields.IntegerType](raw T) (MuteDuration, error) {
	val := MuteDurationValue(raw)
	if val <= MuteDurationUnknown || int(val) >= muteDurationSpec.Max {
		return MuteDuration{}, ErrMuteDurationInvalid()
	}
	return NewMuteDuration(val), nil
}

func ParseMuteDurationString(s string) (MuteDuration, error) {
	val, ok := fields.ParseEnumString[MuteDurationValue](s, muteDurationSpec)
	if !ok || val <= MuteDurationUnknown {
		return MuteDuration{}, ErrMuteDurationInvalid()
	}
	return NewMuteDuration(val), nil
}

func (m MuteDuration) ToDuration() (time.Duration, bool) {
	switch m.Value() {
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

func (m MuteDuration) CalculateUntil(now fields.Timestamp) (fields.Timestamp, error) {
	if m.IsForever() {
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
		return ReactionEmoji{}, ErrReactionEmojiTooLong()
	}

	return NewReactionEmoji(cleaned), nil
}

func ParseRequiredReactionEmoji(raw string) (ReactionEmoji, error) {
	emoji, err := ParseReactionEmoji(raw)
	if err != nil {
		return ReactionEmoji{}, err
	}
	if emoji.IsZero() {
		return ReactionEmoji{}, ErrReactionEmojiRequired()
	}
	return emoji, nil
}
