package fields

import (
	"bytes"
	"net/url"
	"regexp"
	"strings"
	"time"

	"bonfire-api/internal/pkg/ptr"
	"bonfire-api/internal/sanitize"

	"github.com/google/uuid"
)

// ============================================================================
// Core Interfaces & Helpers
// ============================================================================

type Validatable interface {
	IsValid() bool
}

// ensureRequired centralizes mandatory value checks across all domain primitives.
func ensureRequired[T Validatable](field string, val T, err error) (T, error) {
	if err != nil {
		var zero T
		return zero, err
	}
	if !val.IsValid() {
		var zero T
		return zero, ErrRequired(field)
	}
	return val, nil
}

// UnmarshalText standardizes custom unmarshaling logic for text-based fields.
func UnmarshalText[T any](text []byte, fieldName string, parser func(field, raw string) (T, error)) (T, error) {
	if len(text) == 0 {
		var zero T
		return zero, nil
	}
	return parser(fieldName, string(text))
}

// ============================================================================
// Bytes
// ============================================================================

type Bytes struct {
	value []byte
}

func NewBytes(v []byte) Bytes {
	if len(v) == 0 {
		return Bytes{}
	}
	buf := make([]byte, len(v))
	copy(buf, v)
	return Bytes{value: buf}
}

func (b Bytes) Bytes() []byte {
	if len(b.value) == 0 {
		return nil
	}
	buf := make([]byte, len(b.value))
	copy(buf, b.value)
	return buf
}

func (b Bytes) IsZero() bool            { return len(b.value) == 0 }
func (b Bytes) IsValid() bool           { return !b.IsZero() }
func (b Bytes) Equals(other Bytes) bool { return bytes.Equal(b.value, other.value) }

func UnmarshalTextBytes[T any](text []byte, fieldName string, parser func(field string, raw []byte) (T, error)) (T, error) {
	if len(text) == 0 {
		var zero T
		return zero, nil
	}
	return parser(fieldName, text)
}

// ============================================================================
// Text
// ============================================================================

type Text struct {
	value string
}

func NewText(v string) Text           { return Text{value: v} }
func (t Text) String() string         { return t.value }
func (t Text) IsZero() bool           { return t.value == "" }
func (t Text) IsValid() bool          { return !t.IsZero() }
func (t Text) Equals(other Text) bool { return t.value == other.value }

func (t Text) StringPtr() *string {
	if t.IsZero() {
		return nil
	}
	return ptr.To(t.value)
}

func (t Text) MarshalText() ([]byte, error) {
	return []byte(t.value), nil
}

// ============================================================================
// HexColor
// ============================================================================

var rgxHexColor = regexp.MustCompile(`(?i)^#[0-9a-f]{6}$`)

type HexColor struct{ Text }

func ParseHexColor(field, raw string) (HexColor, error) {
	s := sanitize.Text(raw)
	if s == "" {
		return HexColor{}, nil
	}

	if err := Validate(field, s, ValidateCfg{Regex: rgxHexColor}); err != nil {
		return HexColor{}, err
	}

	return HexColor{Text: NewText(strings.ToUpper(s))}, nil
}

func ParseRequiredHexColor(field, raw string) (HexColor, error) {
	v, err := ParseHexColor(field, raw)
	return ensureRequired(field, v, err)
}

func (hc HexColor) Equals(other HexColor) bool { return hc.Text.Equals(other.Text) }

func (hc *HexColor) UnmarshalText(text []byte) error {
	var err error
	*hc, err = UnmarshalText(text, "hex_color", ParseHexColor)
	return err
}

// ============================================================================
// ID
// ============================================================================

type ID uuid.UUID

func ParseID(field string, raw uuid.UUID) (ID, error) {
	id := ID(raw)
	if !id.IsValid() {
		return ID{}, nil
	}
	return id, nil
}

func ParseRequiredID(field string, raw uuid.UUID) (ID, error) {
	return ensureRequired(field, ID(raw), nil)
}

func ParseIDFromString(field, raw string) (ID, error) {
	s := sanitize.Text(raw)
	if s == "" {
		return ID{}, nil
	}

	parsed, err := uuid.Parse(s)
	if err != nil {
		return ID{}, ErrInvalidFormat(field, "Must be a valid UUID")
	}

	return ID(parsed), nil
}

func ParseRequiredIDFromString(field, raw string) (ID, error) {
	v, err := ParseIDFromString(field, raw)
	return ensureRequired(field, v, err)
}

func (id ID) UUID() uuid.UUID      { return uuid.UUID(id) }
func (id ID) String() string       { return uuid.UUID(id).String() }
func (id ID) IsValid() bool        { return uuid.UUID(id) != uuid.Nil }
func (id ID) Equals(other ID) bool { return id == other }

func (id ID) StringPtr() *string {
	if !id.IsValid() {
		return nil
	}
	return ptr.To(id.String())
}

func (id *ID) UnmarshalText(text []byte) error {
	var err error
	*id, err = UnmarshalText(text, "id", ParseIDFromString)
	return err
}

// ============================================================================
// Timestamp
// ============================================================================

type Timestamp struct {
	value time.Time
}

func ParseTimestamp(field, raw string) (Timestamp, error) {
	s := sanitize.Text(raw)
	if s == "" {
		return Timestamp{}, nil
	}

	parsed, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		return Timestamp{}, ErrInvalidFormat(field, "Timestamp must be a valid RFC 3339 date-time format")
	}

	return Timestamp{value: parsed.UTC()}, nil
}

func NewTimestampFromTime(t time.Time) Timestamp {
	if t.IsZero() {
		return Timestamp{}
	}
	return Timestamp{value: t.UTC()}
}

func NewTimestampFromUnix(sec int64) Timestamp {
	if sec <= 0 {
		return Timestamp{}
	}
	return NewTimestampFromTime(time.Unix(sec, 0))
}

func (t Timestamp) Time() time.Time         { return t.value }
func (t Timestamp) IsValid() bool           { return !t.value.IsZero() }
func (t Timestamp) Equals(o Timestamp) bool { return t.value.Equal(o.value) }

func (t Timestamp) TimePtr() *time.Time {
	if !t.IsValid() {
		return nil
	}
	return ptr.To(t.value)
}

func (t Timestamp) String() string {
	if !t.IsValid() {
		return ""
	}
	return t.value.Format(time.RFC3339)
}

func (t Timestamp) StringPtr() *string {
	if !t.IsValid() {
		return nil
	}
	return ptr.To(t.String())
}

func (t Timestamp) Unix() int64 {
	if !t.IsValid() {
		return 0
	}
	return t.value.Unix()
}

func (t Timestamp) UnixPtr() *int64 {
	if !t.IsValid() {
		return nil
	}
	return ptr.To(t.Unix())
}

func (t Timestamp) HasPassed(now time.Time) bool {
	if !t.IsValid() || now.IsZero() {
		return false
	}
	return now.After(t.value)
}

func (t *Timestamp) UnmarshalText(text []byte) error {
	var err error
	*t, err = UnmarshalText(text, "timestamp", ParseTimestamp)
	return err
}

// ============================================================================
// URL
// ============================================================================

const MaxURLLength = 2048

type URL struct{ Text }

func ParseURL(field, raw string) (URL, error) {
	s := sanitize.URL(raw)
	if s == "" {
		return URL{}, nil
	}

	if err := Validate(field, s, ValidateCfg{MaxLen: MaxURLLength}); err != nil {
		return URL{}, err
	}

	parsed, err := url.ParseRequestURI(s)
	if err != nil || parsed.Host == "" || (!strings.HasPrefix(s, "http://") && !strings.HasPrefix(s, "https://")) {
		return URL{}, ErrInvalidFormat(field, "URL must be a valid HTTP or HTTPS address")
	}

	return URL{Text: NewText(parsed.String())}, nil
}

func ParseRequiredURL(field, raw string) (URL, error) {
	v, err := ParseURL(field, raw)
	return ensureRequired(field, v, err)
}

func (u URL) Equals(other URL) bool { return u.Text.Equals(other.Text) }

func (u *URL) UnmarshalText(text []byte) error {
	var err error
	*u, err = UnmarshalText(text, "url", ParseURL)
	return err
}

// ============================================================================
// VerificationCode
// ============================================================================

var rgxVerificationCode = regexp.MustCompile(`^[2-9A-HJ-NP-Z]{6}$`)

type VerificationCode struct{ Text }

func ParseVerificationCode(field, raw string) (VerificationCode, error) {
	s := strings.ToUpper(sanitize.Text(raw))
	if s == "" {
		return VerificationCode{}, nil
	}

	if err := Validate(field, s, ValidateCfg{Regex: rgxVerificationCode}); err != nil {
		return VerificationCode{}, ErrInvalidFormat(field, "Verification code must be 6 alphanumeric characters")
	}

	return VerificationCode{Text: NewText(s)}, nil
}

func ParseRequiredVerificationCode(field, raw string) (VerificationCode, error) {
	v, err := ParseVerificationCode(field, raw)
	return ensureRequired(field, v, err)
}

func (c VerificationCode) Equals(other VerificationCode) bool { return c.Text.Equals(other.Text) }

func (c *VerificationCode) UnmarshalText(text []byte) error {
	var err error
	*c, err = UnmarshalText(text, "verification_code", ParseVerificationCode)
	return err
}
