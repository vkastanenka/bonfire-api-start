package presence

import "bonfire-api/internal/fields"

type Value int

const (
	PresenceUnknown Value = iota
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
	fields.Enum[Value]
}

func New(val Value) Presence {
	return Presence{Enum: fields.NewEnum(val, presenceSpec)}
}

func NewOnline() Presence    { return New(PresenceOnline) }
func NewOffline() Presence   { return New(PresenceOffline) }
func NewIdle() Presence      { return New(PresenceIdle) }
func NewBusy() Presence      { return New(PresenceBusy) }
func NewDND() Presence       { return New(PresenceDND) }
func NewInvisible() Presence { return New(PresenceInvisible) }

func Parse[T fields.IntegerType](raw T) (Presence, error) {
	val := Value(raw)
	if val <= PresenceUnknown || int(val) >= presenceSpec.Max {
		return Presence{}, ErrPresenceInvalid()
	}
	return New(val), nil
}

func ParseString(s string) (Presence, error) {
	val, ok := fields.ParseEnumString[Value](s, presenceSpec)
	if !ok || val <= PresenceUnknown {
		return Presence{}, ErrPresenceInvalid()
	}
	return New(val), nil
}

func (p Presence) IsOnline() bool    { return p.Is(PresenceOnline) }
func (p Presence) IsOffline() bool   { return p.Is(PresenceOffline) }
func (p Presence) IsIdle() bool      { return p.Is(PresenceIdle) }
func (p Presence) IsBusy() bool      { return p.Is(PresenceBusy) }
func (p Presence) IsDND() bool       { return p.Is(PresenceDND) }
func (p Presence) IsInvisible() bool { return p.Is(PresenceInvisible) }

func IsPreferred(p Presence) bool {
	return p.IsIdle() || p.IsBusy() || p.IsDND()
}
