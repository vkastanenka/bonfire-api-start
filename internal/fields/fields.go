package fields

import (
	"bytes"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/netip"
	"net/url"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"bonfire-api/internal/pkg/ptr"
	"bonfire-api/internal/sanitize"
)

/*
TODO:
1. Fields
2. Sanitize
3. Errors
*/

// ============================================================================
// Cursor
// ============================================================================

const (
	DefaultCursorLimit = 50
	MaxCursorLimit     = 100
)

type Cursor struct {
	id          ID
	beforeLimit int
	afterLimit  int
}

func NewCursor(id ID, beforeLimit, afterLimit int) Cursor {
	return Cursor{
		id:          id,
		beforeLimit: beforeLimit,
		afterLimit:  afterLimit,
	}
}

func ParseCursor(idRaw string, beforeLimitRaw, afterLimitRaw string) (Cursor, error) {
	var (
		id          ID
		beforeLimit int
		afterLimit  int
		err         error
	)

	if idRaw != "" {
		id, err = ParseIDFromString("anchor_message_id", idRaw)
		if err != nil {
			return Cursor{}, err
		}
	}

	if beforeLimitRaw != "" {
		beforeLimit, err = parseCursorLimit("before_limit", beforeLimitRaw)
		if err != nil {
			return Cursor{}, err
		}
	}

	if afterLimitRaw != "" {
		afterLimit, err = parseCursorLimit("after_limit", afterLimitRaw)
		if err != nil {
			return Cursor{}, err
		}
	}

	if id.IsValid() && beforeLimit == 0 && afterLimit == 0 {
		beforeLimit = DefaultCursorLimit
	}

	return NewCursor(id, beforeLimit, afterLimit), nil
}

func parseCursorLimit(fieldName, raw string) (int, error) {
	s := sanitize.Text(raw)
	if s == "" {
		return 0, nil
	}

	val, err := strconv.Atoi(s)
	if err != nil || val < 0 {
		return 0, ErrLimitInvalid(fieldName)
	}

	if val > MaxCursorLimit {
		return 0, ErrLimitExceeded(fieldName)
	}

	return val, nil
}

func (c Cursor) ID() ID           { return c.id }
func (c Cursor) BeforeLimit() int { return c.beforeLimit }
func (c Cursor) AfterLimit() int  { return c.afterLimit }

func (c Cursor) IsZero() bool {
	return c.id.IsZero() && c.beforeLimit == 0 && c.afterLimit == 0
}

func (c Cursor) IsValid() bool { return !c.IsZero() }

func (c Cursor) Equals(other Cursor) bool {
	return c.id.Equals(other.id) &&
		c.beforeLimit == other.beforeLimit &&
		c.afterLimit == other.afterLimit
}

// ============================================================================
// Enum
// ============================================================================

type EnumSpec struct {
	Domain string
	Max    int
	Names  []string
	Bytes  [][]byte
}

type Enum[T IntegerType] struct {
	Integer[T]
	desc *EnumSpec
}

func NewEnum[T IntegerType](val T, desc *EnumSpec) Enum[T] {
	return Enum[T]{
		Integer: NewInteger(val),
		desc:    desc,
	}
}

func (e Enum[T]) IsValid() bool {
	if e.desc == nil {
		return false
	}
	val := e.Integer.Int()
	return val >= 0 && val < e.desc.Max && val < len(e.desc.Names)
}

func (e Enum[T]) Is(target T) bool {
	return e.Value() == target
}

func (e Enum[T]) String() string {
	if e.desc == nil {
		return "UNKNOWN"
	}
	val := e.Integer.Int()
	if e.IsValid() {
		return e.desc.Names[val]
	}
	if len(e.desc.Names) > 0 {
		return e.desc.Names[0]
	}
	return "UNKNOWN"
}

func ParseEnumString[T IntegerType](s string, desc *EnumSpec) (T, bool) {
	var zero T
	if desc == nil {
		return zero, false
	}
	str := strings.TrimSpace(s)
	if str == "" {
		return zero, false
	}
	for i := 0; i < len(desc.Names) && i < desc.Max; i++ {
		if strings.EqualFold(desc.Names[i], str) {
			return T(i), true
		}
	}
	return zero, false
}

func ParseEnumInt[T IntegerType, I IntegerType](raw I, desc *EnumSpec) (T, bool) {
	var zero T
	if desc == nil {
		return zero, false
	}
	val := int(raw)
	if val < 0 || val >= desc.Max || val >= len(desc.Names) {
		return zero, false
	}
	return T(val), true
}

func (e Enum[T]) MarshalText() ([]byte, error) {
	if e.desc == nil {
		return []byte("UNKNOWN"), nil
	}
	val := e.Integer.Int()
	if e.IsValid() && val < len(e.desc.Bytes) && len(e.desc.Bytes[val]) > 0 {
		return e.desc.Bytes[val], nil
	}
	if len(e.desc.Bytes) > 0 && len(e.desc.Bytes[0]) > 0 {
		return e.desc.Bytes[0], nil
	}
	return []byte("UNKNOWN"), nil
}

func (e *Enum[T]) UnmarshalText(text []byte) error {
	if e.desc == nil {
		return ErrEnumInvalidDomain()
	}
	str := string(text)
	val, ok := ParseEnumString[T](str, e.desc)
	if !ok {
		return ErrEnumInvalidValue(str)
	}
	e.Integer = NewInteger(val)
	return nil
}

func (e Enum[T]) MarshalJSON() ([]byte, error) {
	b, err := e.MarshalText()
	if err != nil {
		return nil, err
	}
	return json.Marshal(string(b))
}

func (e *Enum[T]) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		var numericVal T
		if numErr := json.Unmarshal(data, &numericVal); numErr == nil {
			e.Integer = NewInteger(numericVal)
			return nil
		}
		return err
	}
	return e.UnmarshalText([]byte(s))
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
		return HexColor{}, ErrHexColorInvalid(fieldName)
	}

	return NewHexColor(strings.ToUpper(s)), nil
}

func ParseRequiredHexColor(fieldName, raw string) (HexColor, error) {
	color, err := ParseHexColor(fieldName, raw)
	if err != nil {
		return HexColor{}, err
	}
	if color.IsZero() {
		return HexColor{}, ErrHexColorRequired(fieldName)
	}
	return color, nil
}

// ============================================================================
// Int
// ============================================================================

type IntegerType interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64
}

type Integer[T IntegerType] struct {
	val T
}

func NewInteger[T IntegerType](val T) Integer[T] {
	return Integer[T]{val: val}
}

func (i Integer[T]) Int() int {
	return int(i.val)
}

func (i Integer[T]) Value() T {
	return i.val
}

func ParseInteger[T IntegerType](fieldName, raw string) (Integer[T], error) {
	s := sanitize.Text(raw)
	if s == "" {
		return Integer[T]{}, nil
	}

	var zero T
	switch any(zero).(type) {
	case uint, uint8, uint16, uint32, uint64, uintptr:
		parsed, err := strconv.ParseUint(s, 10, 64)
		if err != nil {
			return Integer[T]{}, ErrIntInvalid(fieldName)
		}
		return NewInteger(T(parsed)), nil
	default:
		parsed, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return Integer[T]{}, ErrIntInvalid(fieldName)
		}
		return NewInteger(T(parsed)), nil
	}
}

func ParseRequiredInteger[T IntegerType](fieldName, raw string) (Integer[T], error) {
	i, err := ParseInteger[T](fieldName, raw)
	if err != nil {
		return Integer[T]{}, err
	}
	if i.IsZero() {
		return Integer[T]{}, ErrIntRequired(fieldName)
	}
	return i, nil
}

func (i Integer[T]) IsZero() bool                 { var zero T; return i.val == zero }
func (i Integer[T]) IsValid() bool                { return !i.IsZero() }
func (i Integer[T]) Equals(other Integer[T]) bool { return i.val == other.val }

func (i Integer[T]) ValuePtr() *T {
	if i.IsZero() {
		return nil
	}
	return ptr.To(i.val)
}

func (i Integer[T]) IntPtr() *int {
	if i.IsZero() {
		return nil
	}
	return ptr.To(i.Int())
}

func (i Integer[T]) Int16() int16 {
	return int16(i.val)
}

func (i Integer[T]) Int16Ptr() *int16 {
	if i.IsZero() {
		return nil
	}
	return ptr.To(i.Int16())
}

func (i Integer[T]) String() string {
	return fmt.Sprintf("%d", i.val)
}

func (i Integer[T]) StringPtr() *string {
	if i.IsZero() {
		return nil
	}
	return ptr.To(i.String())
}

func (i Integer[T]) MarshalText() ([]byte, error) {
	if i.IsZero() {
		return nil, nil
	}
	return []byte(i.String()), nil
}

func (i *Integer[T]) UnmarshalText(text []byte) error {
	v, err := ParseInteger[T]("int", string(text))
	if err != nil {
		return err
	}
	*i = v
	return nil
}

func (i Integer[T]) MarshalJSON() ([]byte, error) {
	return json.Marshal(i.val)
}

func (i *Integer[T]) UnmarshalJSON(data []byte) error {
	var val T
	if err := json.Unmarshal(data, &val); err != nil {
		return ErrIntJSONInvalid("int")
	}
	*i = NewInteger(val)
	return nil
}

// ============================================================================
// IP
// ============================================================================

type IP struct {
	value netip.Addr
}

func NewIP(addr netip.Addr) IP {
	if !addr.IsValid() {
		return IP{}
	}
	return IP{value: addr}
}

func ParseIP(fieldName, raw string) (IP, error) {
	s := sanitize.Text(raw)
	if s == "" {
		return IP{}, nil
	}

	addr, err := netip.ParseAddr(s)
	if err != nil {
		return IP{}, ErrIPInvalid(fieldName)
	}

	return NewIP(addr), nil
}

func ParseRequiredIP(fieldName, raw string) (IP, error) {
	ip, err := ParseIP(fieldName, raw)
	if err != nil {
		return IP{}, err
	}
	if ip.IsZero() {
		return IP{}, ErrIPRequired(fieldName)
	}
	return ip, nil
}

func (ip IP) Addr() netip.Addr {
	return ip.value
}

func (ip IP) AddrPtr() *netip.Addr {
	if ip.IsZero() {
		return nil
	}
	return ptr.To(ip.value)
}

func (ip IP) IsZero() bool         { return !ip.value.IsValid() }
func (ip IP) IsValid() bool        { return ip.value.IsValid() }
func (ip IP) Equals(other IP) bool { return ip.value == other.value }

func (ip IP) String() string {
	if ip.IsZero() {
		return ""
	}
	return ip.value.String()
}

func (ip IP) StringPtr() *string {
	if ip.IsZero() {
		return nil
	}
	return ptr.To(ip.String())
}

func (ip IP) MarshalText() ([]byte, error) {
	if ip.IsZero() {
		return nil, nil
	}
	return []byte(ip.String()), nil
}

func (ip *IP) UnmarshalText(text []byte) error {
	v, err := ParseIP("ip", string(text))
	if err != nil {
		return err
	}
	*ip = v
	return nil
}

func (ip IP) MarshalJSON() ([]byte, error) {
	if ip.IsZero() {
		return []byte("null"), nil
	}
	return json.Marshal(ip.value.String())
}

func (ip *IP) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	v, err := ParseIP("ip", s)
	if err != nil {
		return err
	}
	*ip = v
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
		return JSON{}, ErrJSONTooLarge(fieldName)
	}

	if !bytes.HasPrefix(trimmed, []byte("{")) || !bytes.HasSuffix(trimmed, []byte("}")) {
		return JSON{}, ErrJSONInvalidFormat(fieldName)
	}

	decoder := json.NewDecoder(bytes.NewReader(trimmed))

	var val map[string]any
	if err := decoder.Decode(&val); err != nil {
		return JSON{}, ErrJSONMalformed(fieldName)
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
		return JSON{}, ErrJSONRequired(fieldName)
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
		return JSON{}, ErrJSONRequired(fieldName)
	}
	return j, nil
}

func ParseJSONFromMap(fieldName string, raw map[string]any) (JSON, error) {
	if len(raw) == 0 {
		return JSON{}, nil
	}

	b, err := json.Marshal(raw)
	if err != nil {
		return JSON{}, ErrJSONMapInvalid(fieldName)
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
	if j.IsZero() || other.IsZero() {
		return false
	}
	return reflect.DeepEqual(j.value, other.value)
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

func validateDepthAndTypes(fieldName string, v any, depth int) error {
	if depth > MaxMetadataDepth {
		return ErrJSONNestingExceeded(fieldName)
	}

	switch node := v.(type) {
	case map[string]any:
		for _, child := range node {
			if err := validateDepthAndTypes(fieldName, child, depth+1); err != nil {
				return err
			}
		}
	case []any:
		for _, child := range node {
			if err := validateDepthAndTypes(fieldName, child, depth+1); err != nil {
				return err
			}
		}
	}

	return nil
}

func ParseRawJSON[T any](raw []byte) (*T, error) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		var zero T
		return &zero, nil
	}

	if len(trimmed) > MaxJSONBytes {
		return nil, ErrJSONTooLarge("")
	}

	var msg T
	if err := json.Unmarshal(trimmed, &msg); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrJSONMalformed(""), err)
	}

	return &msg, nil
}

// ============================================================================
// OS
// ============================================================================

const (
	MinOSLength = 1
	MaxOSLength = 100
)

type OS struct {
	value string
}

func NewOS(s string) OS {
	return OS{value: s}
}

func ParseOS(fieldName, raw string) (OS, error) {
	s := sanitize.Text(raw)
	if s == "" {
		return OS{}, nil
	}

	if len(s) > MaxOSLength {
		return OS{}, ErrOSTooLong(fieldName)
	}

	return OS{value: s}, nil
}

func ParseRequiredOS(fieldName, raw string) (OS, error) {
	osVal, err := ParseOS(fieldName, raw)
	if err != nil {
		return OS{}, err
	}
	if osVal.IsZero() {
		return OS{}, ErrOSRequired(fieldName)
	}
	return osVal, nil
}

func (osVal OS) String() string { return osVal.value }
func (osVal OS) StringPtr() *string {
	if osVal.IsZero() {
		return nil
	}
	return ptr.To(osVal.value)
}
func (osVal OS) IsZero() bool         { return osVal.value == "" }
func (osVal OS) IsValid() bool        { return !osVal.IsZero() }
func (osVal OS) Equals(other OS) bool { return osVal.value == other.value }

func (osVal OS) MarshalText() ([]byte, error) {
	if osVal.IsZero() {
		return nil, nil
	}
	return []byte(osVal.value), nil
}

func (osVal *OS) UnmarshalText(text []byte) error {
	v, err := ParseOS("os", string(text))
	if err != nil {
		return err
	}
	*osVal = v
	return nil
}

func (osVal OS) MarshalJSON() ([]byte, error) {
	if osVal.IsZero() {
		return []byte("null"), nil
	}
	return json.Marshal(osVal.value)
}

func (osVal *OS) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	v, err := ParseOS("os", s)
	if err != nil {
		return err
	}
	*osVal = v
	return nil
}

// ============================================================================
// Text
// ============================================================================

type Text struct {
	value string
}

func NewText(v string) Text {
	return Text{value: sanitize.Text(v)}
}

func ParseText(raw string) Text {
	return Text{value: sanitize.Text(raw)}
}

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

func (t Text) MarshalJSON() ([]byte, error) {
	if t.IsZero() {
		return []byte("null"), nil
	}
	return json.Marshal(t.value)
}

func (t *Text) UnmarshalJSON(data []byte) error {
	if string(data) == "null" || len(data) == 0 {
		*t = Text{}
		return nil
	}
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	*t = NewText(s)
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
		parsed, err = time.Parse(time.RFC3339, s)
		if err != nil {
			return Timestamp{}, ErrTimestampInvalid(fieldName)
		}
	}

	return NewTimestamp(parsed), nil
}

func ParseRequiredTimestamp(fieldName, raw string) (Timestamp, error) {
	t, err := ParseTimestamp(fieldName, raw)
	if err != nil {
		return Timestamp{}, err
	}
	if t.IsZero() {
		return Timestamp{}, ErrTimestampRequired(fieldName)
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

func (t Timestamp) Add(d time.Duration) Timestamp {
	return NewTimestamp(t.Time().Add(d))
}

func (t Timestamp) Compare(other Timestamp) int {
	if t.IsZero() && other.IsZero() {
		return 0
	}
	if t.IsZero() {
		return -1
	}
	if other.IsZero() {
		return 1
	}
	return t.value.Compare(other.value)
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
// Token
// ============================================================================

// MaxTokenLength prevents allocation DoS attacks when processing untrusted tokens.
const MaxTokenLength = 4096

type Token struct {
	value string
}

func NewToken(raw string) (Token, error) {
	return ParseRequiredToken("token", raw)
}

func ParseToken(fieldName, raw string) (Token, error) {
	s := sanitize.Text(raw)
	if s == "" {
		return Token{}, nil
	}
	if len(s) > MaxTokenLength {
		return Token{}, errors.New("") // ErrTokenInvalid(fieldName)
	}
	return Token{value: s}, nil
}

func ParseRequiredToken(fieldName, raw string) (Token, error) {
	t, err := ParseToken(fieldName, raw)
	if err != nil {
		return Token{}, err
	}
	if t.IsZero() {
		return Token{}, errors.New("") // ErrTokenRequired(fieldName)
	}
	return t, nil
}

func (t Token) String() string {
	return t.value
}

func (t Token) IsZero() bool {
	return t.value == ""
}

func (t Token) StringPtr() *string {
	if t.IsZero() {
		return nil
	}
	return ptr.To(t.value)
}

func (t Token) Equals(other Token) bool {
	if t.IsZero() && other.IsZero() {
		return true
	}
	if t.IsZero() || other.IsZero() {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(t.value), []byte(other.value)) == 1
}

func (t Token) MarshalText() ([]byte, error) {
	if t.IsZero() {
		return nil, nil
	}
	return []byte(t.value), nil
}

// func (t *Token) UnmarshalText(text []byte) error {
// 	v, err := ParseToken("token", string(text))
// 	if err != nil {
// 		return err
// 	}
// 	*h = v
// 	return nil
// }

// Hash creates a 32-byte SHA-256 TokenHash of the raw token string.
func (t Token) Hash() (TokenHash, error) {
	if t.IsZero() {
		return TokenHash{}, nil
	}
	h := sha256.Sum256([]byte(t.value))
	return NewTokenHash(h[:])
}

// ============================================================================
// TokenHash
// ============================================================================

const TokenHashByteLength = 32

type TokenHash [TokenHashByteLength]byte

func NewTokenHash(b []byte) (TokenHash, error) {
	if len(b) != TokenHashByteLength {
		return TokenHash{}, ErrTokenHashInvalid("token_hash")
	}
	var h TokenHash
	copy(h[:], b)
	return h, nil
}

func ParseTokenHashString(fieldName, raw string) (TokenHash, error) {
	s := sanitize.Text(raw)
	if s == "" {
		return TokenHash{}, nil
	}

	decoded, err := hex.DecodeString(s)
	if err != nil || len(decoded) != TokenHashByteLength {
		return TokenHash{}, ErrTokenHashInvalid(fieldName)
	}

	var h TokenHash
	copy(h[:], decoded)
	return h, nil
}

func (h TokenHash) IsZero() bool  { return h == TokenHash{} }
func (h TokenHash) IsValid() bool { return !h.IsZero() }

func (h TokenHash) String() string {
	if h.IsZero() {
		return ""
	}
	return hex.EncodeToString(h[:])
}

func (h TokenHash) StringPtr() *string {
	if h.IsZero() {
		return nil
	}
	s := h.String()
	return &s
}

func (h TokenHash) Equals(other TokenHash) bool {
	return subtle.ConstantTimeCompare(h[:], other[:]) == 1
}

func (h TokenHash) MarshalText() ([]byte, error) {
	return []byte(h.String()), nil
}

func (h *TokenHash) UnmarshalText(text []byte) error {
	v, err := ParseTokenHashString("token_hash", string(text))
	if err != nil {
		return err
	}
	*h = v
	return nil
}

func (h TokenHash) MarshalJSON() ([]byte, error) {
	if h.IsZero() {
		return []byte("null"), nil
	}
	return json.Marshal(h.String())
}

func (h *TokenHash) UnmarshalJSON(data []byte) error {
	if string(data) == "null" || len(data) == 0 {
		*h = TokenHash{}
		return nil
	}

	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}

	return h.UnmarshalText([]byte(s))
}

// ============================================================================
// TraceID
// ============================================================================

const traceIDMaxLength = 256

type TraceID struct {
	Text
}

func NewTraceID(v string) TraceID {
	return TraceID{Text: NewText(v)}
}

func ParseTraceID(raw string) (TraceID, error) {
	cleaned := sanitize.Text(raw)
	if cleaned == "" {
		return TraceID{}, nil
	}

	if utf8.RuneCountInString(cleaned) > traceIDMaxLength {
		return TraceID{}, ErrTraceIDTooLong()
	}

	return NewTraceID(cleaned), nil
}

func ParseRequiredTraceID(raw string) (TraceID, error) {
	traceID, err := ParseTraceID(raw)
	if err != nil {
		return TraceID{}, err
	}
	if traceID.IsZero() {
		return TraceID{}, ErrTraceIDRequired()
	}
	return traceID, nil
}

func (t TraceID) Equals(other TraceID) bool {
	return t.Text.Equals(other.Text)
}

func (t *TraceID) UnmarshalText(text []byte) error {
	v, err := ParseTraceID(string(text))
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
		return URL{}, ErrURLTooLong(fieldName)
	}

	parsed, err := url.ParseRequestURI(cleaned)
	if err != nil || parsed.Host == "" || (!strings.HasPrefix(cleaned, "http://") && !strings.HasPrefix(cleaned, "https://")) {
		return URL{}, ErrURLInvalid(fieldName)
	}

	return NewURL(parsed.String()), nil
}

func ParseRequiredURL(fieldName, raw string) (URL, error) {
	u, err := ParseURL(fieldName, raw)
	if err != nil {
		return URL{}, err
	}
	if u.IsZero() {
		return URL{}, ErrURLRequired(fieldName)
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

// ============================================================================
// UserAgent
// ============================================================================

const (
	MinUserAgentLength = 1
	MaxUserAgentLength = 1000
)

type UserAgent struct {
	value string
}

func NewUserAgent(s string) UserAgent {
	return UserAgent{value: s}
}

func ParseUserAgent(fieldName, raw string) (UserAgent, error) {
	s := sanitize.Text(raw)
	if s == "" {
		return UserAgent{}, nil
	}

	if len(s) < MinUserAgentLength || len(s) > MaxUserAgentLength {
		return UserAgent{}, ErrUserAgentTooLong(fieldName)
	}

	return UserAgent{value: s}, nil
}

func ParseRequiredUserAgent(fieldName, raw string) (UserAgent, error) {
	ua, err := ParseUserAgent(fieldName, raw)
	if err != nil {
		return UserAgent{}, err
	}
	if ua.IsZero() {
		return UserAgent{}, ErrUserAgentRequired(fieldName)
	}
	return ua, nil
}

func (ua UserAgent) String() string { return ua.value }
func (ua UserAgent) StringPtr() *string {
	if ua.IsZero() {
		return nil
	}
	return ptr.To(ua.value)
}
func (ua UserAgent) IsZero() bool                { return ua.value == "" }
func (ua UserAgent) IsValid() bool               { return !ua.IsZero() }
func (ua UserAgent) Equals(other UserAgent) bool { return ua.value == other.value }

func (ua UserAgent) MarshalText() ([]byte, error) {
	if ua.IsZero() {
		return nil, nil
	}
	return []byte(ua.value), nil
}

func (ua *UserAgent) UnmarshalText(text []byte) error {
	v, err := ParseUserAgent("user_agent", string(text))
	if err != nil {
		return err
	}
	*ua = v
	return nil
}

func (ua UserAgent) MarshalJSON() ([]byte, error) {
	if ua.IsZero() {
		return []byte("null"), nil
	}
	return json.Marshal(ua.value)
}

func (ua *UserAgent) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	v, err := ParseUserAgent("user_agent", s)
	if err != nil {
		return err
	}
	*ua = v
	return nil
}
