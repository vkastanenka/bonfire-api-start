package user

import (
	"errors"
	"strings"
	"time"
)

// TODO: Move to config
const presenceTTL = 30 * time.Second

var ErrInvalidPresence = errors.New("invalid presence status")

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

func NewPresence(raw string) (Presence, error) {
	s := strings.ToLower(strings.TrimSpace(raw))
	for i := 1; i < int(presenceMax); i++ {
		if presenceNames[i] == s {
			return Presence(i), nil
		}
	}
	return PresenceUnknown, ErrInvalidPresence
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

func (p Presence) MarshalText() ([]byte, error) {
	return []byte(p.String()), nil
}

func (p *Presence) UnmarshalText(text []byte) error {
	if len(text) == 0 {
		*p = PresenceUnknown
		return nil
	}
	parsed, err := NewPresence(string(text))
	if err != nil {
		return err
	}
	*p = parsed
	return nil
}
