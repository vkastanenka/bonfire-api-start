package presence

import (
	"bytes"
	"errors"
	"strings"
	"unsafe"
)

var ErrInvalidPresence = errors.New("invalid presence status")

const (
	EventUpdated = "presence.updated"
)

type PresenceUpdatedPayload struct {
	UserID   string `json:"user_id"`
	Presence string `json:"presence"`
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

func New(raw string) (Presence, error) {
	if strings.TrimSpace(raw) == "" {
		return PresenceUnknown, ErrInvalidPresence
	}
	b := unsafe.Slice(unsafe.StringData(raw), len(raw))
	return ParseBytes(b)
}

func ParseBytes(b []byte) (Presence, error) {
	b = bytes.TrimSpace(b)
	if len(b) == 0 {
		return PresenceUnknown, ErrInvalidPresence
	}

	for i := 1; i < int(presenceMax); i++ {
		if bytes.EqualFold(presenceBytes[i], b) {
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
