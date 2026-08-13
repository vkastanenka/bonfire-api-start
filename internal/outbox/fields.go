package outbox

import (
	"bonfire-api/internal/fields"
	"bonfire-api/internal/sanitize"
)

// Default worker execution bounds
const (
	DefaultMaxAttempts = 5
	MaxEventTypeLength = 128
	MaxErrorLength     = 4000
)

// ============================================================================
// Value Objects
// ============================================================================

type EventType struct {
	fields.Text
}

func ParseEventType(field, raw string) (EventType, error) {
	s := sanitize.Text(raw)
	if err := fields.Validate(field, s, fields.ValidateCfg{
		MaxLen:   MaxEventTypeLength,
		Required: true,
	}); err != nil {
		return EventType{}, err
	}
	return EventType{Text: fields.NewText(s)}, nil
}

type LastError struct {
	fields.Text
}

func ParseLastError(field, raw string) (LastError, error) {
	s := sanitize.Text(raw)
	if err := fields.Validate(field, s, fields.ValidateCfg{
		MaxLen: MaxErrorLength,
	}); err != nil {
		return LastError{}, err
	}
	return LastError{Text: fields.NewText(s)}, nil
}
