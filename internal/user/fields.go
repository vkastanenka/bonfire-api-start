package user

import (
	"regexp"
	"strings"
	"unicode/utf8"

	"bonfire-api/internal/errs"
	"bonfire-api/internal/fields"
	"bonfire-api/internal/presence"
	"bonfire-api/internal/sanitize"
)

// ============================================================================
// Bio
// ============================================================================

const MaxBioLength = 190

type Bio struct {
	fields.Text
}

func NewBio(raw string) (Bio, error) {
	s := sanitize.Text(raw)
	if s == "" {
		return Bio{}, nil
	}

	if len([]rune(s)) > MaxBioLength {
		return Bio{}, errs.InvalidArgument("Invalid user bio.").
			Reason("BIO_TOO_LONG").
			FieldViolation("bio", "Bio must not exceed 190 characters.", "MAX_LENGTH_EXCEEDED")
	}

	return Bio{Text: fields.NewText(s)}, nil
}

func (b Bio) Equals(other Bio) bool {
	return b.Text.Equals(other.Text)
}

func (b Bio) MarshalText() ([]byte, error) {
	return fields.MarshalText(b, Bio.String)
}

func (b *Bio) UnmarshalText(text []byte) error {
	val, err := fields.UnmarshalText(text, NewBio)
	if err != nil {
		return err
	}
	*b = val
	return nil
}

// ============================================================================
// DisplayName
// ============================================================================

const MaxDisplayNameLength = 32

type DisplayName struct {
	fields.Text
}

func NewDisplayName(raw string) (DisplayName, error) {
	s := sanitize.Text(raw)
	if s == "" {
		return DisplayName{}, errs.InvalidArgument("Invalid display name.").
			Reason("DISPLAY_NAME_REQUIRED").
			FieldViolation("display_name", "Display name cannot be empty.", "REQUIRED")
	}

	if len([]rune(s)) > MaxDisplayNameLength {
		return DisplayName{}, errs.InvalidArgument("Invalid display name.").
			Reason("DISPLAY_NAME_TOO_LONG").
			FieldViolation("display_name", "Display name must not exceed 32 characters.", "MAX_LENGTH_EXCEEDED")
	}

	return DisplayName{Text: fields.NewText(s)}, nil
}

func (d DisplayName) Equals(other DisplayName) bool {
	return d.Text.Equals(other.Text)
}

func (d DisplayName) MarshalText() ([]byte, error) {
	return fields.MarshalText(d, DisplayName.String)
}

func (d *DisplayName) UnmarshalText(text []byte) (err error) {
	*d, err = fields.UnmarshalText(text, NewDisplayName)
	return err
}

// ============================================================================
// Email
// ============================================================================

const MaxEmailLength = 255

type Email struct {
	fields.Text
}

func NewEmail(raw string) (Email, error) {
	s := sanitize.Email(raw)
	if s == "" {
		return Email{}, errs.InvalidArgument("Invalid email address.").
			Reason("EMAIL_REQUIRED").
			FieldViolation("email", "Email address cannot be empty", "REQUIRED")
	}

	if len(s) > MaxEmailLength {
		return Email{}, errs.InvalidArgument("Invalid email address.").
			Reason("EMAIL_TOO_LONG").
			FieldViolation("email", "Email address must not exceed 255 characters", "MAX_LENGTH_EXCEEDED")
	}

	at := strings.IndexByte(s, '@')
	if at <= 0 || at == len(s)-1 {
		return Email{}, errs.InvalidArgument("Invalid email address.").
			Reason("EMAIL_INVALID_FORMAT").
			FieldViolation("email", "Must be a valid email address", "INVALID_FORMAT")
	}

	domain := s[at+1:]
	dot := strings.IndexByte(domain, '.')
	if dot <= 0 || dot == len(domain)-1 {
		return Email{}, errs.InvalidArgument("Invalid email address.").
			Reason("EMAIL_INVALID_FORMAT").
			FieldViolation("email", "Must be a valid email address", "INVALID_FORMAT")
	}

	return Email{Text: fields.NewText(s)}, nil
}

func (e Email) Equals(other Email) bool {
	return e.Text.Equals(other.Text)
}

func (e Email) MarshalText() ([]byte, error) {
	return fields.MarshalText(e, Email.String)
}

func (e *Email) UnmarshalText(text []byte) (err error) {
	*e, err = fields.UnmarshalText(text, NewEmail)
	return err
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

func NewPassword(raw string) (Password, error) {
	if raw == "" {
		return Password{}, errs.InvalidArgument("Invalid password.").
			Reason("PASSWORD_REQUIRED").
			FieldViolation("password", "Password cannot be empty", "REQUIRED")
	}

	if len(raw) < MinPasswordLength {
		return Password{}, errs.InvalidArgument("Invalid password.").
			Reason("PASSWORD_TOO_SHORT").
			FieldViolation("password", "Password must be at least 12 characters", "MIN_LENGTH_NOT_MET")
	}

	if len(raw) > MaxPasswordLength {
		return Password{}, errs.InvalidArgument("Invalid password.").
			Reason("PASSWORD_TOO_LONG").
			FieldViolation("password", "Password must not exceed 255 characters", "MAX_LENGTH_EXCEEDED")
	}

	return Password{Text: fields.NewText(raw)}, nil
}

func (p Password) Equals(other Password) bool {
	return p.Text.Equals(other.Text)
}

func (p Password) MarshalText() ([]byte, error) {
	return fields.MarshalText(p, Password.String)
}

func (p *Password) UnmarshalText(text []byte) (err error) {
	*p, err = fields.UnmarshalText(text, NewPassword)
	return err
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

func NewPasswordHash(raw string) (PasswordHash, error) {
	if raw == "" {
		return PasswordHash{}, errs.InvalidArgument("Invalid password hash.").
			Reason("PASSWORD_HASH_REQUIRED").
			FieldViolation("password_hash", "Password hash cannot be empty", "REQUIRED")
	}

	if len(raw) < MinPasswordHashLength {
		return PasswordHash{}, errs.InvalidArgument("Invalid password hash.").
			Reason("PASSWORD_HASH_TOO_SHORT").
			FieldViolation("password_hash", "Password hash must be at least 50 characters", "MIN_LENGTH_NOT_MET")
	}

	if len(raw) > MaxPasswordHashLength {
		return PasswordHash{}, errs.InvalidArgument("Invalid password hash.").
			Reason("PASSWORD_HASH_TOO_LONG").
			FieldViolation("password_hash", "Password hash must not exceed 255 characters", "MAX_LENGTH_EXCEEDED")
	}

	return PasswordHash{Text: fields.NewText(raw)}, nil
}

func (p PasswordHash) Equals(other PasswordHash) bool {
	return p.Text.Equals(other.Text)
}

func (p PasswordHash) MarshalText() ([]byte, error) {
	return fields.MarshalText(p, PasswordHash.String)
}

func (p *PasswordHash) UnmarshalText(text []byte) (err error) {
	*p, err = fields.UnmarshalText(text, NewPasswordHash)
	return err
}

// ============================================================================
// Phone
// ============================================================================

var rgxPhone = regexp.MustCompile(`^\+[1-9]\d{1,14}$`)

type Phone struct {
	fields.Text
}

func NewPhone(raw string) (Phone, error) {
	s := sanitize.Phone(raw)
	if s == "" {
		return Phone{}, nil
	}

	if !rgxPhone.MatchString(s) {
		return Phone{}, errs.InvalidArgument("Invalid phone number.").
			Reason("PHONE_INVALID_FORMAT").
			FieldViolation("phone", "Phone must be in international E.164 format (e.g., +1234567890)", "INVALID_FORMAT")
	}

	return Phone{Text: fields.NewText(s)}, nil
}

func (p Phone) Equals(other Phone) bool {
	return p.Text.Equals(other.Text)
}

func (p Phone) MarshalText() ([]byte, error) {
	return fields.MarshalText(p, Phone.String)
}

func (p *Phone) UnmarshalText(text []byte) (err error) {
	*p, err = fields.UnmarshalText(text, NewPhone)
	return err
}

// ============================================================================
// PreferredPresence
// ============================================================================

func ErrPreferredPresenceInvalid() *errs.Error {
	return errs.InvalidArgument("Invalid preferred presence status.").
		Reason("PREFERRED_PRESENCE_INVALID").
		FieldViolation("preferred_presence", "Must be one of: idle, busy, dnd.", "INVALID_ENUM_VALUE")
}

type PreferredPresence struct {
	value presence.Presence
}

func NewPreferredPresence(raw string) (PreferredPresence, error) {
	s := sanitize.Text(raw)
	if s == "" {
		return PreferredPresence{}, nil
	}

	p, err := presence.ParseBytes([]byte(s))
	if err != nil {
		return PreferredPresence{}, ErrPreferredPresenceInvalid()
	}

	return PreferredPresenceFromPresence(p)
}

func PreferredPresenceFromInt16(v int16) (PreferredPresence, error) {
	if v == 0 {
		return PreferredPresence{}, nil
	}

	var p presence.Presence
	var err error

	switch v {
	case int16(presence.PresenceIdle), int16(presence.PresenceBusy), int16(presence.PresenceDND):
		p, err = presence.FromInt16(v)
		if err != nil {
			return PreferredPresence{}, ErrPreferredPresenceInvalid()
		}
	}

	return PreferredPresenceFromPresence(p)
}

func PreferredPresenceFromPresence(p presence.Presence) (PreferredPresence, error) {
	switch p {
	case presence.PresenceIdle, presence.PresenceBusy, presence.PresenceDND:
		return PreferredPresence{value: p}, nil
	}
	return PreferredPresence{}, ErrPreferredPresenceInvalid()
}

func (pp PreferredPresence) Presence() presence.Presence { return pp.value }

func (pp PreferredPresence) String() string {
	if !pp.IsSet() {
		return ""
	}
	return pp.value.String()
}

func (pp PreferredPresence) StringPtr() *string {
	if !pp.IsSet() {
		return nil
	}
	str := pp.String()
	return &str
}

func (pp PreferredPresence) Int16() int16 { return pp.value.Int16() }

func (pp PreferredPresence) Int16Ptr() *int16 {
	if !pp.IsSet() {
		return nil
	}
	return pp.value.Int16Ptr()
}

func (pp PreferredPresence) IsZero() bool { return pp.value == presence.PresenceUnknown }

func (pp PreferredPresence) IsValid() bool { return pp.IsZero() || pp.IsSet() }

func (pp PreferredPresence) IsSet() bool {
	switch pp.value {
	case presence.PresenceIdle, presence.PresenceBusy, presence.PresenceDND:
		return true
	default:
		return false
	}
}

func (pp PreferredPresence) Equals(other PreferredPresence) bool {
	return pp.value == other.value
}

func (pp PreferredPresence) MarshalText() ([]byte, error) {
	if !pp.IsSet() {
		return nil, nil
	}
	return fields.MarshalText(pp, PreferredPresence.String)
}

func (pp *PreferredPresence) UnmarshalText(text []byte) (err error) {
	*pp, err = fields.UnmarshalText(text, NewPreferredPresence)
	return err
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

func NewUsername(raw string) (Username, error) {
	s := sanitize.Text(raw)
	if s == "" {
		return Username{}, errs.InvalidArgument("Invalid username.").
			Reason("USERNAME_REQUIRED").
			FieldViolation("username", "Username cannot be empty", "REQUIRED")
	}

	runeLen := utf8.RuneCountInString(s)
	if runeLen < MinUsernameLength {
		return Username{}, errs.InvalidArgument("Invalid username.").
			Reason("USERNAME_TOO_SHORT").
			FieldViolation("username", "Username must be at least 3 characters", "MIN_LENGTH_NOT_MET")
	}
	if runeLen > MaxUsernameLength {
		return Username{}, errs.InvalidArgument("Invalid username.").
			Reason("USERNAME_TOO_LONG").
			FieldViolation("username", "Username cannot exceed 32 characters", "MAX_LENGTH_EXCEEDED")
	}

	if !rgxUsername.MatchString(s) {
		return Username{}, errs.InvalidArgument("Invalid username.").
			Reason("USERNAME_INVALID_FORMAT").
			FieldViolation("username", "Username must start and end with an alphanumeric character and contain no consecutive dots or underscores", "INVALID_FORMAT")
	}

	switch strings.ToLower(s) {
	case "admin", "root", "support", "system", "moderator", "bonfire":
		return Username{}, errs.InvalidArgument("Invalid username.").
			Reason("USERNAME_RESERVED").
			FieldViolation("username", "This username is reserved and cannot be used", "RESERVED_VALUE")
	}

	return Username{Text: fields.NewText(s)}, nil
}

func (u Username) Equals(other Username) bool {
	return strings.EqualFold(u.String(), other.String())
}

func (u Username) MarshalText() ([]byte, error) {
	return fields.MarshalText(u, Username.String)
}

func (u *Username) UnmarshalText(text []byte) error {
	val, err := fields.UnmarshalText(text, NewUsername)
	if err != nil {
		return err
	}
	*u = val
	return nil
}
