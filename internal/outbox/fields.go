package outbox

import (
	"encoding/json"
	"fmt"
	"unicode/utf8"

	"bonfire-api/internal/fields"
	"bonfire-api/internal/sanitize"
)

// ============================================================================
// EventType
// ============================================================================

const (
	MinEventTypeLength = 1
	MaxEventTypeLength = 100
)

type EventType struct {
	fields.Text
}

func ParseEventType(field, raw string) (EventType, error) {
	s := sanitize.Text(raw)
	if err := fields.Validate(field, s, fields.ValidateCfg{
		MinLen:   MinEventTypeLength,
		MaxLen:   MaxEventTypeLength,
		Required: true,
	}); err != nil {
		return EventType{}, err
	}

	return EventType{Text: fields.NewText(s)}, nil
}

func (t EventType) Equals(other EventType) bool {
	return t.Text.Equals(other.Text)
}

func (t *EventType) UnmarshalText(text []byte) error {
	var err error
	*t, err = fields.UnmarshalText(text, "event_type", ParseEventType)
	return err
}

// ============================================================================
// AggregateType (Optional)
// ============================================================================

const (
	MinAggregateTypeLength = 1
	MaxAggregateTypeLength = 100
)

type AggregateType struct {
	fields.Text
}

func ParseAggregateType(field, raw string) (AggregateType, error) {
	s := sanitize.Text(raw)
	if s == "" {
		return AggregateType{}, nil
	}

	if err := fields.Validate(field, s, fields.ValidateCfg{
		MinLen: MinAggregateTypeLength,
		MaxLen: MaxAggregateTypeLength,
	}); err != nil {
		return AggregateType{}, err
	}

	return AggregateType{Text: fields.NewText(s)}, nil
}

func (a AggregateType) Equals(other AggregateType) bool {
	return a.Text.Equals(other.Text)
}

func (a *AggregateType) UnmarshalText(text []byte) error {
	var err error
	*a, err = fields.UnmarshalText(text, "aggregate_type", ParseAggregateType)
	return err
}

// ============================================================================
// TraceID (Optional)
// ============================================================================

const (
	MinTraceIDLength = 1
	MaxTraceIDLength = 256
)

type TraceID struct {
	fields.Text
}

func ParseTraceID(field, raw string) (TraceID, error) {
	s := sanitize.Text(raw)
	if s == "" {
		return TraceID{}, nil
	}

	if err := fields.Validate(field, s, fields.ValidateCfg{
		MinLen: MinTraceIDLength,
		MaxLen: MaxTraceIDLength,
	}); err != nil {
		return TraceID{}, err
	}

	return TraceID{Text: fields.NewText(s)}, nil
}

func (t TraceID) Equals(other TraceID) bool {
	return t.Text.Equals(other.Text)
}

func (t *TraceID) UnmarshalText(text []byte) error {
	var err error
	*t, err = fields.UnmarshalText(text, "trace_id", ParseTraceID)
	return err
}

// ============================================================================
// Payload
// ============================================================================

const MaxPayloadByteSize = 102400 // 100 KB limit from CHECK constraint

type Payload struct {
	raw json.RawMessage
}

func ParsePayload(field string, raw json.RawMessage) (Payload, error) {
	if len(raw) == 0 {
		return Payload{}, fields.ErrRequired(field)
	}

	if len(raw) >= MaxPayloadByteSize {
		return Payload{}, fields.ErrInvalidFormat(
			field,
			fmt.Sprintf("Payload size exceeds maximum allowed size of %d bytes", MaxPayloadByteSize),
		)
	}

	// Validate valid JSON format
	if !json.Valid(raw) {
		return Payload{}, fields.ErrInvalidFormat(field, "Must be valid JSON")
	}

	// Enforce DB constraint: payload_populated CHECK (payload != '{}' AND payload != '[]')
	compacted := string(raw)
	if compacted == "{}" || compacted == "[]" {
		return Payload{}, fields.ErrInvalidFormat(field, "Payload cannot be an empty JSON object or array")
	}

	return Payload{raw: raw}, nil
}

func (p Payload) Raw() json.RawMessage {
	return p.raw
}

func (p Payload) IsValid() bool {
	return len(p.raw) > 0
}

// ============================================================================
// LastError (Optional)
// ============================================================================

const (
	MinLastErrorLength = 1
	MaxLastErrorLength = 2000
)

type LastError struct {
	fields.Text
}

func ParseLastError(field, raw string) (LastError, error) {
	s := sanitize.Text(raw)
	if s == "" {
		return LastError{}, nil
	}

	// Truncate if exceeds max length to prevent panic/DB drop on long error backtraces
	if utf8.RuneCountInString(s) > MaxLastErrorLength {
		runes := []rune(s)
		s = string(runes[:MaxLastErrorLength])
	}

	return LastError{Text: fields.NewText(s)}, nil
}

func (l LastError) Equals(other LastError) bool {
	return l.Text.Equals(other.Text)
}

func (l *LastError) UnmarshalText(text []byte) error {
	var err error
	*l, err = fields.UnmarshalText(text, "last_error", ParseLastError)
	return err
}
