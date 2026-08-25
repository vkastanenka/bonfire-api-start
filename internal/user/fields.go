package user

import (
	"bonfire-api/internal/fields"
	"bonfire-api/internal/sanitize"
	"encoding/json"
	"regexp"
	"strings"
	"time"
)

// ============================================================================
// Bio
// ============================================================================

const MaxBioLength = 190

type Bio struct {
	fields.Text
}

func NewBio(s string) Bio {
	return Bio{Text: fields.NewText(s)}
}

func ParseBio(fieldName, raw string) (Bio, error) {
	s := sanitize.Text(raw)
	if s == "" {
		return Bio{}, nil
	}

	if len(s) > MaxBioLength {
		return Bio{}, ErrBioTooLong(fieldName)
	}

	return NewBio(s), nil
}

// ============================================================================
// DisplayName
// ============================================================================

const MaxDisplayNameLength = 32

type DisplayName struct {
	fields.Text
}

func NewDisplayName(s string) DisplayName {
	return DisplayName{Text: fields.NewText(s)}
}

func ParseDisplayName(fieldName, raw string) (DisplayName, error) {
	s := sanitize.Text(raw)
	if s == "" {
		return DisplayName{}, nil
	}

	if len(s) > MaxDisplayNameLength {
		return DisplayName{}, ErrDisplayNameTooLong(fieldName)
	}

	return NewDisplayName(s), nil
}

func ParseRequiredDisplayName(fieldName, raw string) (DisplayName, error) {
	dn, err := ParseDisplayName(fieldName, raw)
	if err != nil {
		return DisplayName{}, err
	}
	if dn.IsZero() {
		return DisplayName{}, ErrDisplayNameRequired(fieldName)
	}
	return dn, nil
}

// ============================================================================
// Email
// ============================================================================

const MaxEmailLength = 255

var rgxEmail = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

type Email struct {
	fields.Text
}

func NewEmail(s string) Email {
	return Email{Text: fields.NewText(s)}
}

func ParseEmail(fieldName, raw string) (Email, error) {
	s := sanitize.Email(raw)
	if s == "" {
		return Email{}, nil
	}

	if len(s) > MaxEmailLength {
		return Email{}, ErrEmailTooLong(fieldName)
	}

	if !rgxEmail.MatchString(s) {
		return Email{}, ErrEmailInvalid(fieldName)
	}

	return NewEmail(s), nil
}

func ParseRequiredEmail(fieldName, raw string) (Email, error) {
	email, err := ParseEmail(fieldName, raw)
	if err != nil {
		return Email{}, err
	}
	if email.IsZero() {
		return Email{}, ErrEmailRequired(fieldName)
	}
	return email, nil
}

// ============================================================================
// Password
// ============================================================================

const (
	MinPasswordLength = 12
	MaxPasswordLength = 255
)

type Password struct {
	fields.Text
}

func NewPassword(s string) Password {
	return Password{Text: fields.NewText(s)}
}

func ParsePassword(fieldName, raw string) (Password, error) {
	if raw == "" {
		return Password{}, nil
	}

	if len(raw) < MinPasswordLength {
		return Password{}, ErrPasswordTooShort(fieldName)
	}

	if len(raw) > MaxPasswordLength {
		return Password{}, ErrPasswordTooLong(fieldName)
	}

	return NewPassword(raw), nil
}

func ParseRequiredPassword(fieldName, raw string) (Password, error) {
	pw, err := ParsePassword(fieldName, raw)
	if err != nil {
		return Password{}, err
	}
	if pw.IsZero() {
		return Password{}, ErrPasswordRequired(fieldName)
	}
	return pw, nil
}

// ============================================================================
// PasswordHash
// ============================================================================

const (
	MinPasswordHashLength = 50
	MaxPasswordHashLength = 255
)

type PasswordHash struct {
	fields.Text
}

func NewPasswordHash(s string) PasswordHash {
	return PasswordHash{Text: fields.NewText(s)}
}

func ParsePasswordHash(fieldName, raw string) (PasswordHash, error) {
	if raw == "" {
		return PasswordHash{}, nil
	}

	if len(raw) < MinPasswordHashLength {
		return PasswordHash{}, ErrPasswordHashTooShort(fieldName)
	}

	if len(raw) > MaxPasswordHashLength {
		return PasswordHash{}, ErrPasswordHashTooLong(fieldName)
	}

	return NewPasswordHash(raw), nil
}

func ParseRequiredPasswordHash(fieldName, raw string) (PasswordHash, error) {
	ph, err := ParsePasswordHash(fieldName, raw)
	if err != nil {
		return PasswordHash{}, err
	}
	if ph.IsZero() {
		return PasswordHash{}, ErrPasswordHashRequired(fieldName)
	}
	return ph, nil
}

// ============================================================================
// Phone
// ============================================================================

var rgxPhone = regexp.MustCompile(`^\+[1-9]\d{1,14}$`)

type Phone struct {
	fields.Text
}

func NewPhone(s string) Phone {
	return Phone{Text: fields.NewText(s)}
}

func ParsePhone(fieldName, raw string) (Phone, error) {
	s := sanitize.Phone(raw)
	if s == "" {
		return Phone{}, nil
	}

	if !rgxPhone.MatchString(s) {
		return Phone{}, ErrPhoneInvalid(fieldName)
	}

	return NewPhone(s), nil
}

// ============================================================================
// Presence
// ============================================================================

type PresenceValue int

const (
	PresenceUnknown PresenceValue = iota
	PresenceOnline
	PresenceOffline
	PresenceIdle
	PresenceBusy
	PresenceDND
	PresenceInvisible
	presenceMax
)

var presenceSpec = &fields.EnumSpec{
	Domain: "PRESENCE",
	Max:    int(presenceMax),
	Names: []string{
		"UNKNOWN",
		"ONLINE",
		"OFFLINE",
		"IDLE",
		"BUSY",
		"DND",
		"INVISIBLE",
	},
	Bytes: [][]byte{
		[]byte("UNKNOWN"),
		[]byte("ONLINE"),
		[]byte("OFFLINE"),
		[]byte("IDLE"),
		[]byte("BUSY"),
		[]byte("DND"),
		[]byte("INVISIBLE"),
	},
}

type Presence struct {
	fields.Enum[PresenceValue]
}

func NewPresence(val PresenceValue) Presence {
	return Presence{Enum: fields.NewEnum(val, presenceSpec)}
}

func NewPresenceOnline() Presence    { return NewPresence(PresenceOnline) }
func NewPresenceOffline() Presence   { return NewPresence(PresenceOffline) }
func NewPresenceIdle() Presence      { return NewPresence(PresenceIdle) }
func NewPresenceBusy() Presence      { return NewPresence(PresenceBusy) }
func NewPresenceDND() Presence       { return NewPresence(PresenceDND) }
func NewPresenceInvisible() Presence { return NewPresence(PresenceInvisible) }

func ParsePresence[T fields.IntegerType](raw T) (Presence, error) {
	val := PresenceValue(raw)
	if val <= PresenceUnknown || int(val) >= presenceSpec.Max {
		return Presence{}, ErrPresenceInvalid()
	}
	return NewPresence(val), nil
}

func ParsePresenceString(s string) (Presence, error) {
	val, ok := fields.ParseEnumString[PresenceValue](s, presenceSpec)
	if !ok || val <= PresenceUnknown {
		return Presence{}, ErrPresenceInvalid()
	}
	return NewPresence(val), nil
}

func (p Presence) IsOnline() bool    { return p.Is(PresenceOnline) }
func (p Presence) IsOffline() bool   { return p.Is(PresenceOffline) }
func (p Presence) IsIdle() bool      { return p.Is(PresenceIdle) }
func (p Presence) IsBusy() bool      { return p.Is(PresenceBusy) }
func (p Presence) IsDND() bool       { return p.Is(PresenceDND) }
func (p Presence) IsInvisible() bool { return p.Is(PresenceInvisible) }

func isPreferred(p Presence) bool {
	return p.IsIdle() || p.IsBusy() || p.IsDND()
}

// ============================================================================
// PreferredPresence
// ============================================================================

type PreferredPresence struct {
	value Presence
}

func NewPreferredPresence(p Presence) PreferredPresence {
	return PreferredPresence{value: p}
}

func ParsePreferredPresence(fieldName, raw string) (PreferredPresence, error) {
	s := sanitize.Text(raw)
	if s == "" {
		return PreferredPresence{}, nil
	}

	p, err := ParsePresenceString(s)
	if err != nil {
		return PreferredPresence{}, ErrPreferredPresenceInvalid(fieldName)
	}

	return ParsePreferredPresenceFromPresence(fieldName, p)
}

func ParsePreferredPresenceFromInt[T fields.IntegerType](fieldName string, v T) (PreferredPresence, error) {
	if v == 0 {
		return PreferredPresence{}, nil
	}

	p, err := ParsePresence(v)
	if err != nil {
		return PreferredPresence{}, ErrPreferredPresenceInvalid(fieldName)
	}

	return ParsePreferredPresenceFromPresence(fieldName, p)
}

func ParsePreferredPresenceFromPresence(fieldName string, p Presence) (PreferredPresence, error) {
	if isPreferred(p) {
		return NewPreferredPresence(p), nil
	}
	return PreferredPresence{}, ErrPreferredPresenceInvalid(fieldName)
}

func (pp PreferredPresence) Presence() Presence { return pp.value }

func (pp PreferredPresence) IsSet() bool {
	return isPreferred(pp.value)
}

func (pp PreferredPresence) IsZero() bool {
	return pp.value.Value() == PresenceUnknown
}

func (pp PreferredPresence) IsValid() bool {
	return pp.IsZero() || pp.IsSet()
}

func (pp PreferredPresence) String() string {
	if !pp.IsSet() {
		return ""
	}
	return pp.value.String()
}

func (pp PreferredPresence) Equals(other PreferredPresence) bool {
	return pp.value.Value() == other.value.Value()
}

func (pp PreferredPresence) MarshalText() ([]byte, error) {
	if !pp.IsSet() {
		return nil, nil
	}
	return []byte(pp.String()), nil
}

func (pp *PreferredPresence) UnmarshalText(text []byte) error {
	v, err := ParsePreferredPresence("preferred_presence", string(text))
	if err != nil {
		return err
	}
	*pp = v
	return nil
}

func (pp PreferredPresence) MarshalJSON() ([]byte, error) {
	if !pp.IsSet() {
		return []byte("null"), nil
	}
	return json.Marshal(pp.String())
}

func (pp *PreferredPresence) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	v, err := ParsePreferredPresence("preferred_presence", s)
	if err != nil {
		return err
	}
	*pp = v
	return nil
}

// -----------------------------------------------------------------------------
// Preferred Presence Duration
// -----------------------------------------------------------------------------

type PreferredPresenceDurationValue int

const (
	PreferredPresenceDurationUnknown PreferredPresenceDurationValue = iota
	PreferredPresenceDuration15Min
	PreferredPresenceDuration1Hour
	PreferredPresenceDuration8Hours
	PreferredPresenceDuration24Hours
	PreferredPresenceDuration3Days
	PreferredPresenceDurationForever
	preferredPresenceDurationMax
)

var preferredPresenceDurationSpec = &fields.EnumSpec{
	Domain: "PREFERRED_PRESENCE_DURATION",
	Max:    int(preferredPresenceDurationMax),
	Names:  []string{"UNKNOWN", "15_MIN", "1_HOUR", "8_HOURS", "24_HOURS", "3_DAYS", "FOREVER"},
	Bytes:  [][]byte{[]byte("UNKNOWN"), []byte("15_MIN"), []byte("1_HOUR"), []byte("8_HOURS"), []byte("24_HOURS"), []byte("3_DAYS"), []byte("FOREVER")},
}

type PreferredPresenceDuration struct {
	fields.Enum[PreferredPresenceDurationValue]
}

func NewPreferredPresenceDuration(val PreferredPresenceDurationValue) PreferredPresenceDuration {
	return PreferredPresenceDuration{Enum: fields.NewEnum(val, preferredPresenceDurationSpec)}
}

func NewPreferredPresenceDuration15Min() PreferredPresenceDuration {
	return NewPreferredPresenceDuration(PreferredPresenceDuration15Min)
}
func NewPreferredPresenceDuration1Hour() PreferredPresenceDuration {
	return NewPreferredPresenceDuration(PreferredPresenceDuration1Hour)
}
func NewPreferredPresenceDuration8Hours() PreferredPresenceDuration {
	return NewPreferredPresenceDuration(PreferredPresenceDuration8Hours)
}
func NewPreferredPresenceDuration24Hours() PreferredPresenceDuration {
	return NewPreferredPresenceDuration(PreferredPresenceDuration24Hours)
}
func NewPreferredPresenceDuration3Days() PreferredPresenceDuration {
	return NewPreferredPresenceDuration(PreferredPresenceDuration3Days)
}
func NewPreferredPresenceDurationForever() PreferredPresenceDuration {
	return NewPreferredPresenceDuration(PreferredPresenceDurationForever)
}

func (p PreferredPresenceDuration) Is15Min() bool  { return p.Is(PreferredPresenceDuration15Min) }
func (p PreferredPresenceDuration) Is1Hour() bool  { return p.Is(PreferredPresenceDuration1Hour) }
func (p PreferredPresenceDuration) Is8Hours() bool { return p.Is(PreferredPresenceDuration8Hours) }
func (p PreferredPresenceDuration) Is24Hours() bool {
	return p.Is(PreferredPresenceDuration24Hours)
}
func (p PreferredPresenceDuration) Is3Days() bool   { return p.Is(PreferredPresenceDuration3Days) }
func (p PreferredPresenceDuration) IsForever() bool { return p.Is(PreferredPresenceDurationForever) }

func ParsePreferredPresenceDuration[T fields.IntegerType](raw T) (PreferredPresenceDuration, error) {
	val := PreferredPresenceDurationValue(raw)
	if val <= PreferredPresenceDurationUnknown || int(val) >= preferredPresenceDurationSpec.Max {
		return PreferredPresenceDuration{}, ErrPreferredPresenceDurationInvalid()
	}
	return NewPreferredPresenceDuration(val), nil
}

func ParsePreferredPresenceDurationString(s string) (PreferredPresenceDuration, error) {
	val, ok := fields.ParseEnumString[PreferredPresenceDurationValue](s, preferredPresenceDurationSpec)
	if !ok || val <= PreferredPresenceDurationUnknown {
		return PreferredPresenceDuration{}, ErrPreferredPresenceDurationInvalid()
	}
	return NewPreferredPresenceDuration(val), nil
}

func (p PreferredPresenceDuration) ToDuration() (time.Duration, bool) {
	switch p.Value() {
	case PreferredPresenceDuration15Min:
		return 15 * time.Minute, true
	case PreferredPresenceDuration1Hour:
		return time.Hour, true
	case PreferredPresenceDuration8Hours:
		return 8 * time.Hour, true
	case PreferredPresenceDuration24Hours:
		return 24 * time.Hour, true
	case PreferredPresenceDuration3Days:
		return 72 * time.Hour, true
	case PreferredPresenceDurationForever:
		return 0, true
	default:
		return 0, false
	}
}

func (p PreferredPresenceDuration) CalculateUntil(now fields.Timestamp) (fields.Timestamp, error) {
	if p.IsForever() {
		return fields.Timestamp{}, nil
	}

	d, ok := p.ToDuration()
	if !ok {
		return fields.Timestamp{}, ErrPreferredPresenceDurationInvalid()
	}

	return fields.NewTimestamp(now.Time().Add(d)), nil
}

// ============================================================================
// Username
// ============================================================================

const (
	MinUsernameLength = 3
	MaxUsernameLength = 32
)

var rgxUsername = regexp.MustCompile(`^[a-zA-Z0-9]+(?:[._][a-zA-Z0-9]+)*$`)

type Username struct {
	fields.Text
}

func NewUsername(s string) Username {
	return Username{Text: fields.NewText(s)}
}

func ParseUsername(fieldName, raw string) (Username, error) {
	s := sanitize.Text(raw)
	if s == "" {
		return Username{}, nil
	}

	if len(s) < MinUsernameLength {
		return Username{}, ErrUsernameTooShort(fieldName)
	}

	if len(s) > MaxUsernameLength {
		return Username{}, ErrUsernameTooLong(fieldName)
	}

	if !rgxUsername.MatchString(s) {
		return Username{}, ErrUsernameInvalid(fieldName)
	}

	switch strings.ToLower(s) {
	case "admin", "root", "support", "system", "moderator", "bonfire":
		return Username{}, ErrUsernameReserved(fieldName)
	}

	return NewUsername(s), nil
}

func ParseRequiredUsername(fieldName, raw string) (Username, error) {
	u, err := ParseUsername(fieldName, raw)
	if err != nil {
		return Username{}, err
	}
	if u.IsZero() {
		return Username{}, ErrUsernameRequired(fieldName)
	}
	return u, nil
}
