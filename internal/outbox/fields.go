package outbox

import (
	"bytes"
	"encoding/json"
	"unicode/utf8"

	"bonfire-api/internal/fields"
	"bonfire-api/internal/sanitize"
)

// -----------------------------------------------------------------------------
// Event Type
// -----------------------------------------------------------------------------

const eventTypeMaxLength = 100

type EventType struct {
	fields.Text
}

func NewEventType(v string) EventType {
	return EventType{Text: fields.NewText(v)}
}

func ParseEventType(raw string) (EventType, error) {
	cleaned := sanitize.Text(raw)
	if cleaned == "" {
		return EventType{}, nil
	}

	if utf8.RuneCountInString(cleaned) > eventTypeMaxLength {
		return EventType{}, ErrEventTypeTooLong()
	}

	return NewEventType(cleaned), nil
}

func ParseRequiredEventType(raw string) (EventType, error) {
	eventType, err := ParseEventType(raw)
	if err != nil {
		return EventType{}, err
	}
	if eventType.IsZero() {
		return EventType{}, ErrEventTypeRequired()
	}
	return eventType, nil
}

// -----------------------------------------------------------------------------
// Aggregate Type
// -----------------------------------------------------------------------------

const aggregateTypeMaxLength = 100

type AggregateType struct {
	fields.Text
}

func NewAggregateType(v string) AggregateType {
	return AggregateType{Text: fields.NewText(v)}
}

func ParseAggregateType(raw string) (AggregateType, error) {
	cleaned := sanitize.Text(raw)
	if cleaned == "" {
		return AggregateType{}, nil
	}

	if utf8.RuneCountInString(cleaned) > aggregateTypeMaxLength {
		return AggregateType{}, ErrAggregateTypeTooLong()
	}

	return NewAggregateType(cleaned), nil
}

func ParseRequiredAggregateType(raw string) (AggregateType, error) {
	aggregateType, err := ParseAggregateType(raw)
	if err != nil {
		return AggregateType{}, err
	}
	if aggregateType.IsZero() {
		return AggregateType{}, ErrAggregateTypeRequired()
	}
	return aggregateType, nil
}

// -----------------------------------------------------------------------------
// Payload
// -----------------------------------------------------------------------------

const maxPayloadByteSize = 102400 // 100 KB limit from CHECK constraint

type Payload struct {
	raw json.RawMessage
}

func NewPayload(raw json.RawMessage) Payload {
	return Payload{raw: raw}
}

func ParsePayload(raw json.RawMessage) (Payload, error) {
	if len(raw) == 0 {
		return Payload{}, nil
	}

	if len(raw) >= maxPayloadByteSize {
		return Payload{}, ErrPayloadTooLarge()
	}

	if !json.Valid(raw) {
		return Payload{}, ErrPayloadInvalidJSON()
	}

	compacted := bytes.TrimSpace(raw)
	if bytes.Equal(compacted, []byte("{}")) || bytes.Equal(compacted, []byte("[]")) {
		return Payload{}, ErrPayloadEmpty()
	}

	return NewPayload(raw), nil
}

func ParseRequiredPayload(raw json.RawMessage) (Payload, error) {
	payload, err := ParsePayload(raw)
	if err != nil {
		return Payload{}, err
	}
	if !payload.IsValid() {
		return Payload{}, ErrPayloadRequired()
	}
	return payload, nil
}

func (p Payload) Raw() json.RawMessage {
	return p.raw
}

func (p Payload) IsValid() bool {
	return len(p.raw) > 0
}

func (p Payload) IsZero() bool {
	return len(p.raw) == 0
}

// -----------------------------------------------------------------------------
// Last Error
// -----------------------------------------------------------------------------

const lastErrorMaxLength = 2000

type LastError struct {
	fields.Text
}

func NewLastError(v string) LastError {
	return LastError{Text: fields.NewText(v)}
}

func ParseLastError(raw string) (LastError, error) {
	cleaned := sanitize.Text(raw)
	if cleaned == "" {
		return LastError{}, nil
	}

	if utf8.RuneCountInString(cleaned) > lastErrorMaxLength {
		runes := []rune(cleaned)
		cleaned = string(runes[:lastErrorMaxLength])
	}

	return NewLastError(cleaned), nil
}
