package presence

import (
	"bytes"
	"strconv"

	"bonfire-api/internal/pkg/sanitize"
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

func ParseInt(raw int) (Presence, error) {
	return sanitize.ParseEnumInt(raw, presenceMax, "presence")
}

func ParseIntBytes(b []byte) (Presence, error) {
	return sanitize.ParseEnumIntBytes(b, presenceMax, "presence")
}

func ParseString(s string) (Presence, error) {
	return sanitize.ParseEnumString[Presence](s, presenceNames[:], "presence")
}

func ParseStringBytes(b []byte) (Presence, error) {
	return sanitize.ParseEnumStringBytes[Presence](b, presenceNames[:], "presence")
}

func (p Presence) IsValid() bool {
	return p > PresenceUnknown && p < presenceMax
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

func IsPreferred(p Presence) bool {
	return p.IsIdle() || p.IsBusy() || p.IsDND()
}

func (p Presence) MarshalText() ([]byte, error) {
	return []byte(p.String()), nil
}

func (p *Presence) UnmarshalText(text []byte) error {
	parsed, err := ParseString(string(text))
	if err != nil {
		return err
	}
	*p = parsed
	return nil
}

func (p Presence) MarshalJSON() ([]byte, error) {
	return strconv.AppendQuote(nil, p.String()), nil
}

func (p *Presence) UnmarshalJSON(data []byte) error {
	if len(data) == 0 || bytes.Equal(data, []byte("null")) {
		*p = PresenceUnknown
		return nil
	}

	parsed, err := ParseStringBytes(data)
	if err != nil {
		return err
	}

	*p = parsed
	return nil
}
