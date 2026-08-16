package presence

import (
	"bonfire-api/internal/errs"
	"bonfire-api/internal/fields"
)

func ErrInvalid() *errs.Error {
	return errs.InvalidArgument("Invalid presence.").
		Reason("PRESENCE_INVALID").
		FieldViolation("presence", "Must be one of UNKNOWN, ONLINE, OFFLINE, IDLE, BUSY, DND, or INVISIBLE.", "INVALID_ENUM_VALUE").
		Meta("domain", "presence")
}

type PresenceValue uint8

const (
	PresenceUnknown PresenceValue = iota
	PresenceOnline
	PresenceOffline
	PresenceIdle
	PresenceBusy
	PresenceDND
	PresenceInvisible
	presenceMax
)

var presenceSpec = &fields.EnumSpec{
	Domain: "PRESENCE",
	Max:    uint8(presenceMax),
	Names: []string{
		"UNKNOWN",
		"ONLINE",
		"OFFLINE",
		"IDLE",
		"BUSY",
		"DND",
		"INVISIBLE",
	},
	Bytes: [][]byte{
		[]byte("UNKNOWN"),
		[]byte("ONLINE"),
		[]byte("OFFLINE"),
		[]byte("IDLE"),
		[]byte("BUSY"),
		[]byte("DND"),
		[]byte("INVISIBLE"),
	},
}

type Presence struct {
	fields.Enum[PresenceValue]
}

func New(val PresenceValue) Presence {
	return Presence{Enum: fields.NewEnum(val, presenceSpec)}
}

func Parse(raw int16) (Presence, error) {
	if raw <= 0 || raw >= int16(presenceMax) {
		return Presence{}, ErrInvalid()
	}
	return New(PresenceValue(raw)), nil
}

func ParseString(s string) (Presence, error) {
	kind, ok := fields.ParseEnumString[PresenceValue](s, presenceSpec)
	if !ok || kind >= presenceMax {
		return Presence{}, ErrInvalid()
	}
	return New(kind), nil
}
