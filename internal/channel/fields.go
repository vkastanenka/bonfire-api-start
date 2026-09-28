package channel

import (
	"bytes"
	"strconv"
	"time"

	"bonfire-api/internal/pkg/errs"
	"bonfire-api/internal/pkg/sanitize"
)

type ChannelType int

const (
	ChannelTypeUnknown ChannelType = iota
	ChannelTypeDirect
	ChannelTypeGroup
	channelTypeMax
)

var channelTypeNames = [...]string{
	ChannelTypeUnknown: "UNKNOWN",
	ChannelTypeDirect:  "DIRECT",
	ChannelTypeGroup:   "GROUP",
}

func ParseChTypeInt(raw int) (ChannelType, error) {
	return sanitize.ParseEnumInt(raw, channelTypeMax, "channel type")
}

func ParseChTypeIntBytes(b []byte) (ChannelType, error) {
	return sanitize.ParseEnumIntBytes(b, channelTypeMax, "channel type")
}

func ParseChTypeString(s string) (ChannelType, error) {
	return sanitize.ParseEnumString[ChannelType](s, channelTypeNames[:], "channel type")
}

func ParseChTypeStringBytes(b []byte) (ChannelType, error) {
	return sanitize.ParseEnumStringBytes[ChannelType](b, channelTypeNames[:], "channel type")
}

func (ct ChannelType) IsValid() bool {
	return ct > ChannelTypeUnknown && ct < channelTypeMax
}

func (ct ChannelType) Int() int {
	if ct.IsValid() {
		return int(ct)
	}
	return int(ChannelTypeUnknown)
}

func (ct ChannelType) String() string {
	if ct.IsValid() {
		return channelTypeNames[ct]
	}
	return channelTypeNames[ChannelTypeUnknown]
}

func (ct ChannelType) IsDirect() bool { return ct == ChannelTypeDirect }
func (ct ChannelType) IsGroup() bool  { return ct == ChannelTypeGroup }

func (ct ChannelType) MarshalText() ([]byte, error) {
	return []byte(ct.String()), nil
}

func (ct *ChannelType) UnmarshalText(text []byte) error {
	parsed, err := ParseChTypeString(string(text))
	if err != nil {
		return err
	}
	*ct = parsed
	return nil
}

func (ct ChannelType) MarshalJSON() ([]byte, error) {
	return strconv.AppendQuote(nil, ct.String()), nil
}

func (ct *ChannelType) UnmarshalJSON(data []byte) error {
	if len(data) == 0 || bytes.Equal(data, []byte("null")) {
		*ct = ChannelTypeUnknown
		return nil
	}

	parsed, err := ParseChTypeStringBytes(data)
	if err != nil {
		return err
	}

	*ct = parsed
	return nil
}

type MessageType int

const (
	MessageTypeUnknown MessageType = iota
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

var messageTypeNames = [...]string{
	MessageTypeUnknown:      "UNKNOWN",
	MessageTypeDefault:      "DEFAULT",
	MessageTypeReply:        "REPLY",
	MessageTypeForward:      "FORWARD",
	MessageTypeMemberAdd:    "MEMBER_ADD",
	MessageTypeMemberRemove: "MEMBER_REMOVE",
	MessageTypeNameChange:   "NAME_CHANGE",
	MessageTypeIconChange:   "ICON_CHANGE",
	MessageTypePin:          "PIN",
}

func ParseMsgTypeInt(raw int) (MessageType, error) {
	return sanitize.ParseEnumInt(raw, messageTypeMax, "message type")
}

func ParseMsgTypeIntBytes(b []byte) (MessageType, error) {
	return sanitize.ParseEnumIntBytes(b, messageTypeMax, "message type")
}

func ParseMsgTypeString(s string) (MessageType, error) {
	return sanitize.ParseEnumString[MessageType](s, messageTypeNames[:], "message type")
}

func ParseMsgTypeStringBytes(b []byte) (MessageType, error) {
	return sanitize.ParseEnumStringBytes[MessageType](b, messageTypeNames[:], "message type")
}

func (mt MessageType) IsValid() bool {
	return mt > MessageTypeUnknown && mt < messageTypeMax
}

func (mt MessageType) Int() int {
	if mt.IsValid() {
		return int(mt)
	}
	return int(MessageTypeUnknown)
}

func (mt MessageType) String() string {
	if mt.IsValid() {
		return messageTypeNames[mt]
	}
	return messageTypeNames[MessageTypeUnknown]
}

func (mt MessageType) IsDefault() bool      { return mt == MessageTypeDefault }
func (mt MessageType) IsReply() bool        { return mt == MessageTypeReply }
func (mt MessageType) IsForward() bool      { return mt == MessageTypeForward }
func (mt MessageType) IsMemberAdd() bool    { return mt == MessageTypeMemberAdd }
func (mt MessageType) IsMemberRemove() bool { return mt == MessageTypeMemberRemove }
func (mt MessageType) IsNameChange() bool   { return mt == MessageTypeNameChange }
func (mt MessageType) IsIconChange() bool   { return mt == MessageTypeIconChange }
func (mt MessageType) IsPin() bool          { return mt == MessageTypePin }

func (mt MessageType) MarshalText() ([]byte, error) {
	return []byte(mt.String()), nil
}

func (mt *MessageType) UnmarshalText(text []byte) error {
	parsed, err := ParseMsgTypeString(string(text))
	if err != nil {
		return err
	}
	*mt = parsed
	return nil
}

func (mt MessageType) MarshalJSON() ([]byte, error) {
	return strconv.AppendQuote(nil, mt.String()), nil
}

func (mt *MessageType) UnmarshalJSON(data []byte) error {
	if len(data) == 0 || bytes.Equal(data, []byte("null")) {
		*mt = MessageTypeUnknown
		return nil
	}

	parsed, err := ParseMsgTypeStringBytes(data)
	if err != nil {
		return err
	}

	*mt = parsed
	return nil
}

type MuteDuration int

const (
	MuteDurationUnknown MuteDuration = iota
	MuteDuration15Min
	MuteDuration1Hour
	MuteDuration8Hours
	MuteDuration24Hours
	MuteDuration3Days
	MuteDurationForever
	muteDurationMax
)

var muteDurationNames = [...]string{
	MuteDurationUnknown: "UNKNOWN",
	MuteDuration15Min:   "15_MIN",
	MuteDuration1Hour:   "1_HOUR",
	MuteDuration8Hours:  "8_HOURS",
	MuteDuration24Hours: "24_HOURS",
	MuteDuration3Days:   "3_DAYS",
	MuteDurationForever: "FOREVER",
}

func ParseMuteDurationInt(raw int) (MuteDuration, error) {
	return sanitize.ParseEnumInt(raw, muteDurationMax, "mute duration")
}

func ParseMuteDurationIntBytes(b []byte) (MuteDuration, error) {
	return sanitize.ParseEnumIntBytes(b, muteDurationMax, "mute duration")
}

func ParseMuteDurationString(s string) (MuteDuration, error) {
	return sanitize.ParseEnumString[MuteDuration](s, muteDurationNames[:], "mute duration")
}

func ParseMuteDurationStringBytes(b []byte) (MuteDuration, error) {
	return sanitize.ParseEnumStringBytes[MuteDuration](b, muteDurationNames[:], "mute duration")
}

func (m MuteDuration) IsValid() bool {
	return m > MuteDurationUnknown && m < muteDurationMax
}

func (m MuteDuration) Int() int {
	if m.IsValid() {
		return int(m)
	}
	return int(MuteDurationUnknown)
}

func (m MuteDuration) String() string {
	if m.IsValid() {
		return muteDurationNames[m]
	}
	return muteDurationNames[MuteDurationUnknown]
}

func (m MuteDuration) Is15Min() bool   { return m == MuteDuration15Min }
func (m MuteDuration) Is1Hour() bool   { return m == MuteDuration1Hour }
func (m MuteDuration) Is8Hours() bool  { return m == MuteDuration8Hours }
func (m MuteDuration) Is24Hours() bool { return m == MuteDuration24Hours }
func (m MuteDuration) Is3Days() bool   { return m == MuteDuration3Days }
func (m MuteDuration) IsForever() bool { return m == MuteDurationForever }

func (m MuteDuration) ToDuration() (time.Duration, bool) {
	switch m {
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

func (m MuteDuration) CalculateUntil(now time.Time) (*time.Time, error) {
	if !m.IsValid() {
		return nil, errs.Internal("Cannot calculate expiry for invalid duration.").
			Meta("enum", "channel mute duration")
	}
	if m.IsForever() {
		return nil, nil
	}

	dur, _ := m.ToDuration()
	until := now.Add(dur)
	return &until, nil
}

func (m MuteDuration) MarshalText() ([]byte, error) {
	return []byte(m.String()), nil
}

func (m *MuteDuration) UnmarshalText(text []byte) error {
	parsed, err := ParseMuteDurationString(string(text))
	if err != nil {
		return err
	}
	*m = parsed
	return nil
}

func (m MuteDuration) MarshalJSON() ([]byte, error) {
	return strconv.AppendQuote(nil, m.String()), nil
}

func (m *MuteDuration) UnmarshalJSON(data []byte) error {
	if len(data) == 0 || bytes.Equal(data, []byte("null")) {
		*m = MuteDurationUnknown
		return nil
	}

	parsed, err := ParseMuteDurationStringBytes(data)
	if err != nil {
		return err
	}

	*m = parsed
	return nil
}
