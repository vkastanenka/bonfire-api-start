package user

import (
	"encoding/json"
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
	case PresenceOffline:
		return "offline"
	case PresenceIdle:
		return "idle"
	case PresenceBusy:
		return "busy"
	case PresenceDND:
		return "dnd"
	case PresenceInvisible:
		return "invisible"
	default:
		return "offline"
	}
}
func Parse(s string) Presence {
	switch s {
	case "online":
		return PresenceOnline
	case "offline":
		return PresenceOffline
	case "idle":
		return PresenceIdle
	case "busy":
		return PresenceBusy
	case "dnd":
		return PresenceDND
	case "invisible":
		return PresenceInvisible
	default:
		return PresenceUnknown
	}
}

func (p Presence) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("%q", p.String())), nil
}

func (p *Presence) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}

	parsed := Parse(s)
	if !parsed.Valid() {
		return fmt.Errorf("invalid presence status: %q", s)
	}

	*p = parsed
	return nil
}
