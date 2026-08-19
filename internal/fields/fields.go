package fields

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"bonfire-api/internal/errs"
	"bonfire-api/internal/pkg/ptr"
	"bonfire-api/internal/sanitize"

	"github.com/google/uuid"
)

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

func (b Bytes) IsZero() bool  { return len(b.value) == 0 }
func (b Bytes) IsValid() bool { return !b.IsZero() }
func (b Bytes) Len() int      { return len(b.value) }

func (b Bytes) Equals(other Bytes) bool {
	if b.IsZero() && other.IsZero() {
		return true
	}
	return bytes.Equal(b.value, other.value)
}

func (b Bytes) MarshalBinary() ([]byte, error) {
	return b.Bytes(), nil
}

func (b *Bytes) UnmarshalBinary(data []byte) error {
	*b = NewBytes(data)
	return nil
}

// ============================================================================
// Enum
// ============================================================================

type EnumSpec struct {
	Domain string
	Max    uint8
	Names  []string
	Bytes  [][]byte
}

type Enum[T ~uint8] struct {
	Value uint8
	Desc  *EnumSpec
}

func NewEnum[T ~uint8](val T, desc *EnumSpec) Enum[T] {
	return Enum[T]{Value: uint8(val), Desc: desc}
}

func (e Enum[T]) IsValid() bool {
	return e.Desc != nil && e.Value > 0 && e.Value < e.Desc.Max
}

func (e Enum[T]) Raw() T {
	return T(e.Value)
}

func (e Enum[T]) Int16() int16 {
	return int16(e.Value)
}

func (e Enum[T]) Int16Ptr() *int16 {
	if !e.IsValid() {
		return nil
	}
	return ptr.To(int16(e.Value))
}

func (e Enum[T]) String() string {
	if e.Desc == nil {
		return "UNKNOWN"
	}
	if e.IsValid() && int(e.Value) < len(e.Desc.Names) {
		return e.Desc.Names[e.Value]
	}
	if len(e.Desc.Names) > 0 {
		return e.Desc.Names[0]
	}
	return "UNKNOWN"
}

func (e Enum[T]) MarshalText() ([]byte, error) {
	if e.Desc == nil {
		return []byte("UNKNOWN"), nil
	}
	if e.IsValid() && int(e.Value) < len(e.Desc.Bytes) {
		return e.Desc.Bytes[e.Value], nil
	}
	if len(e.Desc.Bytes) > 0 {
		return e.Desc.Bytes[0], nil
	}
	return []byte("UNKNOWN"), nil
}

func ParseEnumString[T ~uint8](s string, desc *EnumSpec) (T, bool) {
	if desc == nil {
		return 0, false
	}
	str := strings.TrimSpace(s)
	if str == "" {
		return 0, false
	}
	for i := 1; i < len(desc.Names); i++ {
		if strings.EqualFold(desc.Names[i], str) {
			return T(i), true
		}
	}
	return 0, false
}

// ============================================================================
// HexColor
// ============================================================================

var rgxHexColor = regexp.MustCompile(`(?i)^#[0-9a-f]{6}$`)

type HexColor struct {
	Text
}

func NewHexColor(v string) HexColor {
	return HexColor{Text: NewText(v)}
}

func ParseHexColor(fieldName, raw string) (HexColor, error) {
	s := sanitize.Text(raw)
	if s == "" {
		return HexColor{}, nil
	}

	if !rgxHexColor.MatchString(s) {
		return HexColor{}, errs.InvalidArgument("Invalid hex color format.").
			Reason("INVALID_HEX_COLOR").
			FieldViolation(fieldName, "Must be a 6-character hex color (e.g. #FF0000).", "INVALID_FORMAT")
	}

	return NewHexColor(strings.ToUpper(s)), nil
}

func ParseRequiredHexColor(fieldName, raw string) (HexColor, error) {
	color, err := ParseHexColor(fieldName, raw)
	if err != nil {
		return HexColor{}, err
	}
	if color.IsZero() {
		return HexColor{}, errs.InvalidArgument("Hex color is required.").
			Reason("HEX_COLOR_REQUIRED").
			FieldViolation(fieldName, "Field is required.", "REQUIRED")
	}
	return color, nil
}

// ============================================================================
// ID
// ============================================================================

type ID uuid.UUID

func NewID() (ID, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return ID{}, errs.Internal("Unable to create new ID.").Wrap(err)
	}
	return ID(id), nil
}

func ParseID(raw uuid.UUID) (ID, error) {
	id := ID(raw)
	if id.IsZero() {
		return ID{}, nil
	}
	return id, nil
}

func ParseRequiredID(fieldName string, raw uuid.UUID) (ID, error) {
	id := ID(raw)
	if id.IsZero() {
		return ID{}, errs.InvalidArgument(fieldName+" is required.").
			Reason("ID_REQUIRED").
			FieldViolation(fieldName, fieldName+" is required.", "REQUIRED")
	}
	return id, nil
}

func ParseIDs(fieldName string, raws []uuid.UUID) ([]ID, error) {
	if len(raws) == 0 {
		return []ID{}, nil
	}

	ids := make([]ID, 0, len(raws))
	for _, raw := range raws {
		id, err := ParseID(raw)
		if err != nil {
			return nil, err
		}
		if id.IsZero() {
			continue
		}
		ids = append(ids, id)
	}

	return ids, nil
}

func ParseIDFromString(fieldName, raw string) (ID, error) {
	s := sanitize.Text(raw)
	if s == "" {
		return ID{}, nil
	}

	parsed, err := uuid.Parse(s)
	if err != nil {
		return ID{}, errs.InvalidArgument("Invalid ID format.").
			Reason("INVALID_ID_FORMAT").
			FieldViolation(fieldName, "Must be a valid UUID.", "INVALID_FORMAT")
	}

	return ID(parsed), nil
}

func ParseRequiredIDFromString(fieldName, raw string) (ID, error) {
	id, err := ParseIDFromString(fieldName, raw)
	if err != nil {
		return ID{}, err
	}
	if id.IsZero() {
		return ID{}, errs.InvalidArgument(fieldName+" is required.").
			Reason("ID_REQUIRED").
			FieldViolation(fieldName, fieldName+" is required.", "REQUIRED")
	}
	return id, nil
}

func DedupeIDs(ids []ID) []ID {
	if len(ids) == 0 {
		return []ID{}
	}
	seen := make(map[ID]struct{}, len(ids))
	result := make([]ID, 0, len(ids))
	for _, id := range ids {
		if id.IsZero() {
			continue
		}
		if _, exists := seen[id]; !exists {
			seen[id] = struct{}{}
			result = append(result, id)
		}
	}
	return result
}

func RemoveID(ids []ID, target ID) []ID {
	if len(ids) == 0 {
		return []ID{}
	}
	result := make([]ID, 0, len(ids))
	for _, id := range ids {
		if !id.Equals(target) {
			result = append(result, id)
		}
	}
	return result
}

func (id ID) UUID() uuid.UUID      { return uuid.UUID(id) }
func (id ID) String() string       { return uuid.UUID(id).String() }
func (id ID) IsZero() bool         { return uuid.UUID(id) == uuid.Nil }
func (id ID) IsValid() bool        { return !id.IsZero() }
func (id ID) Equals(other ID) bool { return id == other }
func (id ID) Compare(other ID) int {
	return bytes.Compare(id[:], other[:])
}

func (id ID) UUIDPtr() *uuid.UUID {
	if id.IsZero() {
		return nil
	}
	return ptr.To(id.UUID())
}

func (id ID) StringPtr() *string {
	if id.IsZero() {
		return nil
	}
	return ptr.To(id.String())
}

func (id ID) MarshalText() ([]byte, error) {
	if id.IsZero() {
		return nil, nil
	}
	return []byte(id.String()), nil
}

func (id *ID) UnmarshalText(text []byte) error {
	v, err := ParseIDFromString("id", string(text))
	if err != nil {
		return err
	}
	*id = v
	return nil
}

// ============================================================================
// JSON
// ============================================================================

const (
	MaxJSONBytes     = 16 * 1024
	MaxMetadataDepth = 5
)

type JSON struct {
	value map[string]any
}

func NewJSON(v map[string]any) JSON {
	if len(v) == 0 {
		return JSON{}
	}
	return JSON{value: v}
}

func ParseJSON(fieldName string, raw []byte) (JSON, error) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		return JSON{}, nil
	}

	if len(trimmed) > MaxJSONBytes {
		return JSON{}, errs.InvalidArgument("JSON payload is too large.").
			Reason("METADATA_TOO_LARGE").
			FieldViolation(fieldName, fmt.Sprintf("Payload must be %d bytes or fewer.", MaxJSONBytes), "MAX_SIZE_EXCEEDED")
	}

	if !bytes.HasPrefix(trimmed, []byte("{")) || !bytes.HasSuffix(trimmed, []byte("}")) {
		return JSON{}, errs.InvalidArgument("Invalid JSON format.").
			Reason("INVALID_METADATA_FORMAT").
			FieldViolation(fieldName, "Payload must be a JSON object.", "INVALID_TYPE")
	}

	decoder := json.NewDecoder(bytes.NewReader(trimmed))

	var val map[string]any
	if err := decoder.Decode(&val); err != nil {
		return JSON{}, errs.InvalidArgument("Invalid JSON payload.").
			Reason("MALFORMED_JSON").
			FieldViolation(fieldName, "Payload must be valid JSON.", "MALFORMED_JSON")
	}

	if len(val) == 0 {
		return JSON{}, nil
	}

	if err := validateDepthAndTypes(fieldName, val, 1); err != nil {
		return JSON{}, err
	}

	return NewJSON(val), nil
}

func ParseRequiredJSON(fieldName string, raw []byte) (JSON, error) {
	j, err := ParseJSON(fieldName, raw)
	if err != nil {
		return JSON{}, err
	}
	if j.IsZero() {
		return JSON{}, errs.InvalidArgument("JSON payload is required.").
			Reason("METADATA_REQUIRED").
			FieldViolation(fieldName, "Field is required.", "REQUIRED")
	}
	return j, nil
}

func ParseJSONFromString(fieldName, raw string) (JSON, error) {
	cleaned := sanitize.Text(raw)
	if cleaned == "" {
		return JSON{}, nil
	}
	return ParseJSON(fieldName, []byte(cleaned))
}

func ParseRequiredJSONFromString(fieldName, raw string) (JSON, error) {
	j, err := ParseJSONFromString(fieldName, raw)
	if err != nil {
		return JSON{}, err
	}
	if j.IsZero() {
		return JSON{}, errs.InvalidArgument("JSON payload is required.").
			Reason("METADATA_REQUIRED").
			FieldViolation(fieldName, "Field is required.", "REQUIRED")
	}
	return j, nil
}

func ParseJSONFromMap(fieldName string, raw map[string]any) (JSON, error) {
	if len(raw) == 0 {
		return JSON{}, nil
	}

	b, err := json.Marshal(raw)
	if err != nil {
		return JSON{}, errs.InvalidArgument("Invalid JSON structure.").
			Reason("MALFORMED_JSON").
			FieldViolation(fieldName, "Failed to process JSON map.", "INVALID_STRUCTURE")
	}

	return ParseJSON(fieldName, b)
}

func (j JSON) Value() map[string]any {
	if j.IsZero() {
		return nil
	}
	return j.value
}

func (j JSON) ValuePtr() *map[string]any {
	if j.IsZero() {
		return nil
	}
	return ptr.To(j.value)
}

func (j JSON) Bytes() []byte {
	if j.IsZero() {
		return nil
	}
	b, err := j.MarshalJSON()
	if err != nil {
		return nil
	}
	return b
}

func (j JSON) IsZero() bool  { return len(j.value) == 0 }
func (j JSON) IsValid() bool { return !j.IsZero() }

func (j JSON) Equals(other JSON) bool {
	if j.IsZero() && other.IsZero() {
		return true
	}
	b1, err1 := j.MarshalJSON()
	b2, err2 := other.MarshalJSON()
	if err1 != nil || err2 != nil {
		return false
	}
	return bytes.Equal(b1, b2)
}

func (j JSON) MarshalJSON() ([]byte, error) {
	if j.IsZero() {
		return []byte("null"), nil
	}
	return json.Marshal(j.value)
}

func (j *JSON) UnmarshalJSON(data []byte) error {
	v, err := ParseJSON("json", data)
	if err != nil {
		return err
	}
	*j = v
	return nil
}

func validateDepthAndTypes(fieldName string, curr map[string]any, depth int) error {
	if depth > MaxMetadataDepth {
		return errs.InvalidArgument("JSON nesting is too deep.").
			Reason("METADATA_NESTING_EXCEEDED").
			FieldViolation(fieldName, fmt.Sprintf("Nesting level cannot exceed %d.", MaxMetadataDepth), "MAX_DEPTH_EXCEEDED")
	}

	for _, v := range curr {
		if childMap, ok := v.(map[string]any); ok {
			if err := validateDepthAndTypes(fieldName, childMap, depth+1); err != nil {
				return err
			}
		}
	}

	return nil
}

// ============================================================================
// Text
// ============================================================================

type Text struct {
	value string
}

func NewText(v string) Text { return Text{value: v} }

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

func (t Text) Len() int       { return len(t.value) }
func (t Text) RuneCount() int { return utf8.RuneCountInString(t.value) }

func (t Text) MarshalText() ([]byte, error) {
	return []byte(t.value), nil
}

func (t *Text) UnmarshalText(text []byte) error {
	*t = NewText(string(text))
	return nil
}

// ============================================================================
// Timestamp
// ============================================================================

type Timestamp struct {
	value time.Time
}

func NewTimestamp(t time.Time) Timestamp {
	if t.IsZero() {
		return Timestamp{}
	}
	return Timestamp{value: t.UTC()}
}

func NewTimestampFromUnix(sec int64) Timestamp {
	if sec <= 0 {
		return Timestamp{}
	}
	return NewTimestamp(time.Unix(sec, 0))
}

func ParseTimestamp(fieldName, raw string) (Timestamp, error) {
	s := sanitize.Text(raw)
	if s == "" {
		return Timestamp{}, nil
	}

	parsed, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		return Timestamp{}, errs.InvalidArgument("Invalid timestamp format.").
			Reason("INVALID_TIMESTAMP_FORMAT").
			FieldViolation(fieldName, "Timestamp must be a valid RFC 3339 date-time format.", "INVALID_FORMAT")
	}

	return NewTimestamp(parsed), nil
}

func ParseRequiredTimestamp(fieldName, raw string) (Timestamp, error) {
	t, err := ParseTimestamp(fieldName, raw)
	if err != nil {
		return Timestamp{}, err
	}
	if t.IsZero() {
		return Timestamp{}, errs.InvalidArgument("Timestamp is required.").
			Reason("TIMESTAMP_REQUIRED").
			FieldViolation(fieldName, "Field is required.", "REQUIRED")
	}
	return t, nil
}

func Now() Timestamp {
	return NewTimestamp(time.Now())
}

func (t Timestamp) Time() time.Time             { return t.value }
func (t Timestamp) IsZero() bool                { return t.value.IsZero() }
func (t Timestamp) IsValid() bool               { return !t.IsZero() }
func (t Timestamp) Equals(other Timestamp) bool { return t.value.Equal(other.value) }

func (t Timestamp) TimePtr() *time.Time {
	if t.IsZero() {
		return nil
	}
	return ptr.To(t.value)
}

func (t Timestamp) String() string {
	if t.IsZero() {
		return ""
	}
	return t.value.Format(time.RFC3339)
}

func (t Timestamp) StringPtr() *string {
	if t.IsZero() {
		return nil
	}
	return ptr.To(t.String())
}

func (t Timestamp) Unix() int64 {
	if t.IsZero() {
		return 0
	}
	return t.value.Unix()
}

func (t Timestamp) UnixPtr() *int64 {
	if t.IsZero() {
		return nil
	}
	return ptr.To(t.Unix())
}

func (t Timestamp) After(other Timestamp) bool {
	if t.IsZero() {
		return false
	}
	if other.IsZero() {
		return true
	}
	return t.value.After(other.value)
}

func (t Timestamp) Before(other Timestamp) bool {
	if t.IsZero() {
		return !other.IsZero()
	}
	if other.IsZero() {
		return false
	}
	return t.value.Before(other.value)
}

func (t Timestamp) HasPassed(now time.Time) bool {
	if t.IsZero() || now.IsZero() {
		return false
	}
	return now.After(t.value)
}

func (t Timestamp) MarshalText() ([]byte, error) {
	if t.IsZero() {
		return nil, nil
	}
	return []byte(t.String()), nil
}

func (t *Timestamp) UnmarshalText(text []byte) error {
	v, err := ParseTimestamp("timestamp", string(text))
	if err != nil {
		return err
	}
	*t = v
	return nil
}

// ============================================================================
// URL
// ============================================================================

const URLMaxLength = 2048

type URL struct {
	Text
}

func NewURL(v string) URL {
	return URL{Text: NewText(v)}
}

func ParseURL(fieldName, raw string) (URL, error) {
	cleaned := sanitize.URL(raw)
	if cleaned == "" {
		return URL{}, nil
	}

	if len(cleaned) > URLMaxLength {
		return URL{}, errs.InvalidArgument("URL exceeds maximum length.").
			Reason("URL_TOO_LONG").
			FieldViolation(fieldName, fmt.Sprintf("Must not exceed %d characters.", URLMaxLength), "MAX_LENGTH_EXCEEDED")
	}

	parsed, err := url.ParseRequestURI(cleaned)
	if err != nil || parsed.Host == "" || (!strings.HasPrefix(cleaned, "http://") && !strings.HasPrefix(cleaned, "https://")) {
		return URL{}, errs.InvalidArgument("Invalid URL format.").
			Reason("INVALID_URL_FORMAT").
			FieldViolation(fieldName, "URL must be a valid HTTP or HTTPS address.", "INVALID_FORMAT")
	}

	return NewURL(parsed.String()), nil
}

func ParseRequiredURL(fieldName, raw string) (URL, error) {
	u, err := ParseURL(fieldName, raw)
	if err != nil {
		return URL{}, err
	}
	if u.IsZero() {
		return URL{}, errs.InvalidArgument("URL is required.").
			Reason("URL_REQUIRED").
			FieldViolation(fieldName, "Field is required.", "REQUIRED")
	}
	return u, nil
}

func (u URL) Equals(other URL) bool {
	return u.Text.Equals(other.Text)
}

func (u *URL) UnmarshalText(text []byte) error {
	v, err := ParseURL("url", string(text))
	if err != nil {
		return err
	}
	*u = v
	return nil
}
