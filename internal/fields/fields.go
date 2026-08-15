package fields

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"

	"bonfire-api/internal/errs"
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

func (id ID) UUIDPtr() *uuid.UUID {
	if !id.IsValid() {
		return nil
	}
	return ptr.To(id.UUID())
}

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

// -----------------------------------------------------------------------------
// SystemMetadata
// -----------------------------------------------------------------------------

type SystemMetadata struct {
	value map[string]any
}

const (
	MaxSystemMetadataBytes = 16 * 1024
	MaxMetadataDepth       = 5
)

// ParseSystemMetadata validates and constructs a SystemMetadata instance from raw JSON bytes.
func ParseSystemMetadata(domain string, raw []byte) (*SystemMetadata, error) {
	// 1. Fast-path nil or empty payload check
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		return nil, nil
	}

	// 2. Enforce byte length bounds before parsing to avoid CPU/Memory waste
	if len(trimmed) > MaxSystemMetadataBytes {
		return nil, errs.InvalidArgument("System metadata payload is too large.").
			Reason("METADATA_TOO_LARGE").
			FieldViolation("system_metadata", fmt.Sprintf("Metadata must be %d bytes or fewer.", MaxSystemMetadataBytes), "MAX_SIZE_EXCEEDED").
			Meta("domain", domain)
	}

	// 3. Strict object check: System metadata MUST be a JSON Object `{...}`, not an array `[...]` or primitive
	if !bytes.HasPrefix(trimmed, []byte("{")) || !bytes.HasSuffix(trimmed, []byte("}")) {
		return nil, errs.InvalidArgument("Invalid system metadata format.").
			Reason("INVALID_METADATA_FORMAT").
			FieldViolation("system_metadata", "System metadata must be a JSON object.", "INVALID_TYPE").
			Meta("domain", domain)
	}

	// 4. Single-pass unmarshal using a strictly configured decoder
	decoder := json.NewDecoder(bytes.NewReader(trimmed))

	var val map[string]any
	if err := decoder.Decode(&val); err != nil {
		return nil, errs.InvalidArgument("Invalid JSON payload.").
			Reason("MALFORMED_JSON").
			FieldViolation("system_metadata", "System metadata must be valid JSON.", "MALFORMED_JSON").
			Meta("domain", domain)
	}

	// 5. Check if the object had keys after stripping empty structures `{}`
	if len(val) == 0 {
		return nil, nil
	}

	// 6. Deep safety inspection (Nesting depth + Primitive sanitation)
	if err := validateDepthAndTypes(domain, val, 1); err != nil {
		return nil, err
	}

	return &SystemMetadata{value: val}, nil
}

// ParseSystemMetadataFromString wraps ParseSystemMetadata for raw string inputs.
func ParseSystemMetadataFromString(domain string, raw *string) (*SystemMetadata, error) {
	if raw == nil {
		return nil, nil
	}
	return ParseSystemMetadata(domain, []byte(*raw))
}

// ParseSystemMetadataFromMap validates and constructs a SystemMetadata instance from an already parsed map.
func ParseSystemMetadataFromMap(domain string, raw map[string]any) (*SystemMetadata, error) {
	if len(raw) == 0 {
		return nil, nil
	}

	b, err := json.Marshal(raw)
	if err != nil {
		return nil, errs.InvalidArgument("Invalid system metadata structure.").
			Reason("MALFORMED_JSON").
			FieldViolation("system_metadata", "Failed to process metadata map.", "INVALID_STRUCTURE").
			Meta("domain", domain)
	}

	return ParseSystemMetadata(domain, b)
}

// Value returns a copy of the underlying map structure.
func (m *SystemMetadata) Value() map[string]any {
	if m == nil || m.value == nil {
		return nil
	}
	return m.value
}

// EncodeJSON marshals the inner payload back to a standard JSON byte array for SQL insertion.
func (m *SystemMetadata) EncodeJSON() ([]byte, error) {
	if m == nil || len(m.value) == 0 {
		return nil, nil
	}
	return json.Marshal(m.value)
}

func validateDepthAndTypes(domain string, curr map[string]any, depth int) error {
	if depth > MaxMetadataDepth {
		return errs.InvalidArgument("System metadata nesting is too deep.").
			Reason("METADATA_NESTING_EXCEEDED").
			FieldViolation("system_metadata", fmt.Sprintf("Nesting level cannot exceed %d.", MaxMetadataDepth), "MAX_DEPTH_EXCEEDED").
			Meta("domain", "messages")
	}

	for _, v := range curr {
		if childMap, ok := v.(map[string]any); ok {
			if err := validateDepthAndTypes(domain, childMap, depth+1); err != nil {
				return err
			}
		}
	}

	return nil
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
