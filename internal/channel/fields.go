package channel

import (
	"bytes"
	"fmt"
	"strconv"
	"time"

	"bonfire-api/internal/sanitize"
)

// -----------------------------------------------------------------------------
// Channel Type
// -----------------------------------------------------------------------------

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

func ParseChannelType(raw int) (ChannelType, error) {
	ct := ChannelType(raw)
	if !ct.IsValid() {
		return ChannelTypeUnknown, fmt.Errorf("invalid channel type value: %d", raw)
	}
	return ct, nil
}

func ParseChannelTypeString(s string) (ChannelType, error) {
	for i, name := range channelTypeNames {
		if name == s {
			return ChannelType(i), nil
		}
	}
	return ChannelTypeUnknown, fmt.Errorf("invalid channel type string: %q", s)
}

func ParseChannelTypeBytes(raw []byte) (ChannelType, error) {
	cleaned := sanitize.Bytes(raw)
	if len(cleaned) == 0 {
		return ChannelTypeUnknown, nil
	}
	return ParseChannelTypeString(string(cleaned))
}

func (ct ChannelType) IsValid() bool {
	return ct > ChannelTypeUnknown && ct < channelTypeMax
}

func (ct ChannelType) String() string {
	if uint(ct) < uint(len(channelTypeNames)) {
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
	parsed, err := ParseChannelTypeString(string(text))
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

	parsed, err := ParseChannelTypeBytes(data)
	if err != nil {
		return err
	}

	*ct = parsed
	return nil
}

// -----------------------------------------------------------------------------
// Message Type
// -----------------------------------------------------------------------------

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

func ParseMessageType(raw int) (MessageType, error) {
	mt := MessageType(raw)
	if !mt.IsValid() {
		return MessageTypeUnknown, fmt.Errorf("invalid message type value: %d", raw)
	}
	return mt, nil
}

func ParseMessageTypeString(s string) (MessageType, error) {
	for i, name := range messageTypeNames {
		if name == s {
			return MessageType(i), nil
		}
	}
	return MessageTypeUnknown, fmt.Errorf("invalid message type string: %q", s)
}

func ParseMessageTypeBytes(raw []byte) (MessageType, error) {
	cleaned := sanitize.Bytes(raw)
	if len(cleaned) == 0 {
		return MessageTypeUnknown, nil
	}
	return ParseMessageTypeString(string(cleaned))
}

func (mt MessageType) IsValid() bool {
	return mt > MessageTypeUnknown && mt < messageTypeMax
}

func (mt MessageType) String() string {
	if uint(mt) < uint(len(messageTypeNames)) {
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
	parsed, err := ParseMessageTypeString(string(text))
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

	parsed, err := ParseMessageTypeBytes(data)
	if err != nil {
		return err
	}

	*mt = parsed
	return nil
}

// -----------------------------------------------------------------------------
// Mute Duration
// -----------------------------------------------------------------------------

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

func ParseMuteDuration(raw int) (MuteDuration, error) {
	m := MuteDuration(raw)
	if !m.IsValid() {
		return MuteDurationUnknown, fmt.Errorf("invalid mute duration value: %d", raw)
	}
	return m, nil
}

func ParseMuteDurationString(s string) (MuteDuration, error) {
	for i, name := range muteDurationNames {
		if name == s {
			return MuteDuration(i), nil
		}
	}
	return MuteDurationUnknown, fmt.Errorf("invalid mute duration string: %q", s)
}

func ParseMuteDurationBytes(raw []byte) (MuteDuration, error) {
	cleaned := sanitize.Bytes(raw)
	if len(cleaned) == 0 {
		return MuteDurationUnknown, nil
	}
	return ParseMuteDurationString(string(cleaned))
}

func (m MuteDuration) IsValid() bool {
	return m > MuteDurationUnknown && m < muteDurationMax
}

func (m MuteDuration) String() string {
	if uint(m) < uint(len(muteDurationNames)) {
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
		return nil, fmt.Errorf("cannot calculate expiry for invalid duration: %s", m)
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

	parsed, err := ParseMuteDurationBytes(data)
	if err != nil {
		return err
	}

	*m = parsed
	return nil
}
