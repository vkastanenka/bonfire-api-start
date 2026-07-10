package presence

import (
	"bytes"
	"fmt"
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

func (p Presence) Valid() bool {
	return p > PresenceUnknown && p < presenceMax
}

func (p Presence) String() string {
	switch p {
	case PresenceOnline:
		return "online"
	case PresenceIdle:
		return "idle"
	case PresenceBusy:
		return "busy"
	case PresenceDND:
		return "dnd"
	default:
		return "offline"
	}
}

func ParsePresence(s string) Presence {
	switch s {
	case "online":
		return PresenceOnline
	case "idle":
		return PresenceIdle
	case "busy":
		return PresenceBusy
	case "dnd":
		return PresenceDND
	default:
		return PresenceOffline
	}
}

func (p Presence) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("%q", p.String())), nil
}

func (p *Presence) UnmarshalJSON(data []byte) error {
	data = bytes.Trim(data, "\"")

	*p = ParsePresence(string(data))
	return nil
}
