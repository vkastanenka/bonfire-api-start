package user

import (
	"bonfire-api/internal/errs"
	"bonfire-api/internal/fields"
)

func ErrPresenceInvalid() *errs.Error {
	return errs.InvalidArgument("Invalid presence.").
		Reason("PRESENCE_INVALID").
		FieldViolation("presence", "Must be one of ONLINE, OFFLINE, IDLE, BUSY, DND, or INVISIBLE.", "INVALID_ENUM_VALUE").
		Meta("domain", "presence")
}

type PresenceValue int

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
	Max:    int(presenceMax),
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

func NewPresence(val PresenceValue) Presence {
	return Presence{Enum: fields.NewEnum(val, presenceSpec)}
}

func NewPresenceOnline() Presence    { return NewPresence(PresenceOnline) }
func NewPresenceOffline() Presence   { return NewPresence(PresenceOffline) }
func NewPresenceIdle() Presence      { return NewPresence(PresenceIdle) }
func NewPresenceBusy() Presence      { return NewPresence(PresenceBusy) }
func NewPresenceDND() Presence       { return NewPresence(PresenceDND) }
func NewPresenceInvisible() Presence { return NewPresence(PresenceInvisible) }

func ParsePresence[T fields.IntegerType](raw T) (Presence, error) {
	val := PresenceValue(raw)
	if val <= PresenceUnknown || int(val) >= presenceSpec.Max {
		return Presence{}, ErrPresenceInvalid()
	}
	return NewPresence(val), nil
}

func ParsePresenceString(s string) (Presence, error) {
	val, ok := fields.ParseEnumString[PresenceValue](s, presenceSpec)
	if !ok || val <= PresenceUnknown {
		return Presence{}, ErrPresenceInvalid()
	}
	return NewPresence(val), nil
}

func (p Presence) IsOnline() bool    { return p.Is(PresenceOnline) }
func (p Presence) IsOffline() bool   { return p.Is(PresenceOffline) }
func (p Presence) IsIdle() bool      { return p.Is(PresenceIdle) }
func (p Presence) IsBusy() bool      { return p.Is(PresenceBusy) }
func (p Presence) IsDND() bool       { return p.Is(PresenceDND) }
func (p Presence) IsInvisible() bool { return p.Is(PresenceInvisible) }
