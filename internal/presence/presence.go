package presence

import (
	"bytes"
	"strings"
	"unsafe"

	"bonfire-api/internal/errs"
)

func ErrPresenceInvalid() *errs.Error {
	return errs.InvalidArgument("Invalid presence status.").
		Reason("PRESENCE_INVALID").
		FieldViolation("presence", "Must be one of: online, offline, idle, busy, dnd, invisible.", "INVALID_ENUM_VALUE")
}

type Presence uint8

const (
	PresenceUnknown Presence = iota
	PresenceOnline
	PresenceOffline
	PresenceIdle
	PresenceBusy
	PresenceDND
	PresenceInvisible
	presenceMax
)

var presenceNames = [...]string{
	PresenceUnknown:   "unknown",
	PresenceOnline:    "online",
	PresenceOffline:   "offline",
	PresenceIdle:      "idle",
	PresenceBusy:      "busy",
	PresenceDND:       "dnd",
	PresenceInvisible: "invisible",
}

var presenceBytes = [...][]byte{
	PresenceUnknown:   []byte("unknown"),
	PresenceOnline:    []byte("online"),
	PresenceOffline:   []byte("offline"),
	PresenceIdle:      []byte("idle"),
	PresenceBusy:      []byte("busy"),
	PresenceDND:       []byte("dnd"),
	PresenceInvisible: []byte("invisible"),
}

func Parse(raw string) (Presence, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return PresenceUnknown, ErrPresenceInvalid()
	}

	switch s {
	case "online":
		return PresenceOnline, nil
	case "offline":
		return PresenceOffline, nil
	case "idle":
		return PresenceIdle, nil
	case "busy":
		return PresenceBusy, nil
	case "dnd":
		return PresenceDND, nil
	case "invisible":
		return PresenceInvisible, nil
	}

	switch strings.ToLower(s) {
	case "online":
		return PresenceOnline, nil
	case "offline":
		return PresenceOffline, nil
	case "idle":
		return PresenceIdle, nil
	case "busy":
		return PresenceBusy, nil
	case "dnd":
		return PresenceDND, nil
	case "invisible":
		return PresenceInvisible, nil
	default:
		return PresenceUnknown, ErrPresenceInvalid()
	}
}

func ParseBytes(b []byte) (Presence, error) {
	b = bytes.TrimSpace(b)
	if len(b) == 0 {
		return PresenceUnknown, ErrPresenceInvalid()
	}

	s := unsafe.String(unsafe.SliceData(b), len(b))
	return Parse(s)
}

func (p Presence) IsValid() bool {
	return p > PresenceUnknown && p < presenceMax
}

func (p Presence) String() string {
	if p.IsValid() {
		return presenceNames[p]
	}
	return presenceNames[PresenceUnknown]
}

func (p Presence) Int16() int16 {
	return int16(p)
}

func (p Presence) Int16Ptr() *int16 {
	if !p.IsValid() {
		return nil
	}
	v := p.Int16()
	return &v
}

func FromInt16(v int16) (Presence, error) {
	if v < 0 || v >= int16(presenceMax) {
		return PresenceUnknown, ErrPresenceInvalid()
	}
	p := Presence(v)
	if p == PresenceUnknown {
		return PresenceUnknown, ErrPresenceInvalid()
	}
	return p, nil
}

func (p Presence) MarshalText() ([]byte, error) {
	if p.IsValid() {
		return presenceBytes[p], nil
	}
	return presenceBytes[PresenceUnknown], nil
}

func (p *Presence) UnmarshalText(text []byte) error {
	if len(text) == 0 {
		*p = PresenceUnknown
		return nil
	}
	parsed, err := ParseBytes(text)
	if err != nil {
		return err
	}
	*p = parsed
	return nil
}
