package user

import (
	"fmt"
	"strconv"
	"time"
)

// TODO: Move to config
const presenceTTL = 30 * time.Second

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

// Fast string-to-enum lookup map.
var presenceValues = map[string]Presence{
	"online":    PresenceOnline,
	"offline":   PresenceOffline,
	"idle":      PresenceIdle,
	"busy":      PresenceBusy,
	"dnd":       PresenceDND,
	"invisible": PresenceInvisible,
}

func (p Presence) Valid() bool {
	return p > PresenceUnknown && p < presenceMax
}

func (p Presence) String() string {
	if int(p) < len(presenceNames) {
		return presenceNames[p]
	}
	return "offline"
}

func ParsePresence(s string) Presence {
	if p, ok := presenceValues[s]; ok {
		return p
	}
	return PresenceUnknown
}

func (p Presence) MarshalJSON() ([]byte, error) {
	return []byte(strconv.Quote(p.String())), nil
}

func (p *Presence) UnmarshalJSON(data []byte) error {
	unquoted, err := strconv.Unquote(string(data))
	if err != nil {
		return fmt.Errorf("invalid presence string: %w", err)
	}

	parsed := ParsePresence(unquoted)
	if !parsed.Valid() {
		return fmt.Errorf("invalid presence status: %q", unquoted)
	}

	*p = parsed
	return nil
}
