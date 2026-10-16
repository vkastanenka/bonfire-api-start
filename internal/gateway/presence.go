package gateway

import (
	"bonfire-api/internal/pkg/parse"
)

type Presence int

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
	PresenceUnknown:   "UNKNOWN",
	PresenceOnline:    "ONLINE",
	PresenceOffline:   "OFFLINE",
	PresenceIdle:      "IDLE",
	PresenceBusy:      "BUSY",
	PresenceDND:       "DND",
	PresenceInvisible: "INVISIBLE",
}

func ParsePresenceInt(i int) (Presence, error) {
	return parse.EnumInt(i, presenceMax, "presence")
}

func ParsePresenceIntBytes(b []byte) (Presence, error) {
	return parse.EnumIntBytes(b, presenceMax, "presence")
}

func ParsePresenceString(s string) (Presence, error) {
	return parse.EnumString[Presence](s, presenceNames[:], "presence")
}

func ParsePresenceStringBytes(b []byte) (Presence, error) {
	return parse.EnumStringBytes[Presence](b, presenceNames[:], "presence")
}

func (p Presence) IsValid() bool {
	return p > PresenceUnknown && p < presenceMax
}

func (p Presence) IsPreferred() bool {
	return p.IsIdle() || p.IsBusy() || p.IsDND()
}

func (p Presence) Int() int {
	if p.IsValid() {
		return int(p)
	}
	return int(PresenceUnknown)
}

func (p Presence) String() string {
	if p.IsValid() {
		return presenceNames[p]
	}
	return presenceNames[PresenceUnknown]
}

func (p Presence) IsOnline() bool    { return p == PresenceOnline }
func (p Presence) IsOffline() bool   { return p == PresenceOffline }
func (p Presence) IsIdle() bool      { return p == PresenceIdle }
func (p Presence) IsBusy() bool      { return p == PresenceBusy }
func (p Presence) IsDND() bool       { return p == PresenceDND }
func (p Presence) IsInvisible() bool { return p == PresenceInvisible }

func (p Presence) MarshalText() ([]byte, error) {
	return []byte(p.String()), nil
}

func (p *Presence) UnmarshalText(text []byte) error {
	parsed, err := ParsePresenceString(string(text))
	if err != nil {
		return err
	}
	*p = parsed
	return nil
}
