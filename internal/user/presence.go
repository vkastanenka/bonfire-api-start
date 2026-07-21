package user

import (
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
	s := p.String()
	b := make([]byte, 0, len(s)+2)
	b = append(b, '"')
	b = append(b, s...)
	b = append(b, '"')
	return b, nil
}

func (p *Presence) UnmarshalJSON(data []byte) error {
	if len(data) < 2 || data[0] != '"' || data[len(data)-1] != '"' {
		return fmt.Errorf("invalid presence status: %s", string(data))
	}

	unquoted := string(data[1 : len(data)-1])

	parsed := Parse(unquoted)
	if !parsed.Valid() {
		return fmt.Errorf("invalid presence status: %q", unquoted)
	}

	*p = parsed
	return nil
}
