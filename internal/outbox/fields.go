package outbox

import (
	"encoding/json"
	"unicode/utf8"

	"bonfire-api/internal/fields"
	"bonfire-api/internal/sanitize"
)

// -----------------------------------------------------------------------------
// Type
// -----------------------------------------------------------------------------

const typeMaxLength = 100

type Type struct {
	fields.Text
}

func NewType(v string) Type {
	return Type{Text: fields.NewText(v)}
}

func ParseType(raw string) (Type, error) {
	cleaned := sanitize.Text(raw)
	if cleaned == "" {
		return Type{}, nil
	}

	if utf8.RuneCountInString(cleaned) > typeMaxLength {
		return Type{}, ErrTypeTooLong()
	}

	return NewType(cleaned), nil
}

func ParseRequiredType(raw string) (Type, error) {
	eventType, err := ParseType(raw)
	if err != nil {
		return Type{}, err
	}
	if eventType.IsZero() {
		return Type{}, ErrTypeRequired()
	}
	return eventType, nil
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

// func ParsePayload(raw json.RawMessage) (Payload, error) {
// 	if len(raw) == 0 {
// 		return Payload{}, nil
// 	}

// 	if len(raw) >= maxPayloadByteSize {
// 		return Payload{}, ErrPayloadTooLarge()
// 	}

// 	if !json.Valid(raw) {
// 		return Payload{}, ErrPayloadInvalidJSON()
// 	}

// 	compacted := bytes.TrimSpace(raw)
// 	if bytes.Equal(compacted, []byte("{}")) || bytes.Equal(compacted, []byte("[]")) {
// 		return Payload{}, ErrPayloadEmpty()
// 	}

// 	return NewPayload(raw), nil
// }

// func ParseRequiredPayload(raw json.RawMessage) (Payload, error) {
// 	payload, err := ParsePayload(raw)
// 	if err != nil {
// 		return Payload{}, err
// 	}
// 	if !payload.IsValid() {
// 		return Payload{}, ErrPayloadRequired()
// 	}
// 	return payload, nil
// }

func (p Payload) Raw() json.RawMessage {
	return p.raw
}

func (p Payload) IsValid() bool {
	return len(p.raw) > 0
}

func (p Payload) IsZero() bool {
	return len(p.raw) == 0
}
