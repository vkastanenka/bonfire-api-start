package presence

import (
	"bytes"
	"fmt"
	"strconv"

	"bonfire-api/internal/sanitize"
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

func Parse(raw int) (Presence, error) {
	p := Presence(raw)
	if !p.IsValid() {
		return PresenceUnknown, fmt.Errorf("invalid presence value: %d", raw)
	}
	return p, nil
}

func ParseString(s string) (Presence, error) {
	str := sanitize.EnumValue(s)
	if str == "" {
		return PresenceUnknown, nil
	}
	for i, name := range presenceNames {
		if name == str {
			return Presence(i), nil
		}
	}
	return PresenceUnknown, fmt.Errorf("invalid presence string: %q", s)
}

func ParseBytes(raw []byte) (Presence, error) {
	cleaned := sanitize.Bytes(raw)
	if len(cleaned) == 0 {
		return PresenceUnknown, nil
	}
	return ParseString(string(cleaned))
}

func (p Presence) IsValid() bool {
	return p > PresenceUnknown && p < presenceMax
}

func (p Presence) String() string {
	if uint(p) < uint(len(presenceNames)) {
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

	parsed, err := ParseBytes(data)
	if err != nil {
		return err
	}

	*p = parsed
	return nil
}
