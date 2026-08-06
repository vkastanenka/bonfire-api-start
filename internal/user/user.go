package user

import (
	"errors"
	"fmt"
	"time"
)

var (
	ErrUserDisabled          = errors.New("cannot perform action on a disabled user account")
	ErrUserScheduledDeletion = errors.New("cannot perform action on an account scheduled for deletion")
)

type User struct {
	id                     ID
	email                  Email
	username               Username
	displayName            DisplayName
	passwordHash           Password
	phone                  Phone
	bio                    Bio
	avatarURL              URL
	bannerColor            HexColor
	preferredPresence      PreferredPresence
	preferredPresenceUntil Timestamp
	verifiedAt             Timestamp
	disabledAt             Timestamp
	deleteScheduledAt      Timestamp
	createdAt              Timestamp
	updatedAt              Timestamp
}

// Getters
func (u *User) ID() ID                               { return u.id }
func (u *User) Email() Email                         { return u.email }
func (u *User) Username() Username                   { return u.username }
func (u *User) DisplayName() DisplayName             { return u.displayName }
func (u *User) PasswordHash() Password               { return u.passwordHash }
func (u *User) Phone() Phone                         { return u.phone }
func (u *User) Bio() Bio                             { return u.bio }
func (u *User) AvatarURL() URL                       { return u.avatarURL }
func (u *User) BannerColor() HexColor                { return u.bannerColor }
func (u *User) PreferredPresence() PreferredPresence { return u.preferredPresence }
func (u *User) PreferredPresenceUntil() Timestamp    { return u.preferredPresenceUntil }
func (u *User) VerifiedAt() Timestamp                { return u.verifiedAt }
func (u *User) DisabledAt() Timestamp                { return u.disabledAt }
func (u *User) DeleteScheduledAt() Timestamp         { return u.deleteScheduledAt }
func (u *User) CreatedAt() Timestamp                 { return u.createdAt }
func (u *User) UpdatedAt() Timestamp                 { return u.updatedAt }

// Domain State Checks
func (u *User) IsVerified() bool             { return u.verifiedAt.IsValid() }
func (u *User) IsDisabled() bool             { return u.disabledAt.IsValid() }
func (u *User) IsScheduledForDeletion() bool { return u.deleteScheduledAt.IsValid() }

func (u *User) EffectivePresence() PreferredPresence {
	if u.preferredPresenceUntil.HasPassed(time.Now().UTC()) {
		return PreferredPresence{}
	}
	return u.preferredPresence
}

// Factory for creating brand-new users
func New(
	id ID,
	email Email,
	username Username,
	displayName DisplayName,
	passwordHash Password,
) (*User, error) {
	now := time.Now().UTC()
	ts := NewTimestampFromTime(now)

	return &User{
		id:           id,
		email:        email,
		username:     username,
		displayName:  displayName,
		passwordHash: passwordHash,
		createdAt:    ts,
		updatedAt:    ts,
	}, nil
}

// Reconstitute hydrates an existing domain model from storage
func Reconstitute(
	id ID,
	email Email,
	username Username,
	displayName DisplayName,
	passwordHash Password,
	phone Phone,
	bio Bio,
	avatarURL URL,
	bannerColor HexColor,
	preferredPresence PreferredPresence,
	preferredPresenceUntil Timestamp,
	verifiedAt Timestamp,
	disabledAt Timestamp,
	deleteScheduledAt Timestamp,
	createdAt, updatedAt Timestamp,
) *User {
	return &User{
		id:                     id,
		email:                  email,
		username:               username,
		displayName:            displayName,
		passwordHash:           passwordHash,
		phone:                  phone,
		bio:                    bio,
		avatarURL:              avatarURL,
		bannerColor:            bannerColor,
		preferredPresence:      preferredPresence,
		preferredPresenceUntil: preferredPresenceUntil,
		verifiedAt:             verifiedAt,
		disabledAt:             disabledAt,
		deleteScheduledAt:      deleteScheduledAt,
		createdAt:              createdAt,
		updatedAt:              updatedAt,
	}
}

// Domain Mutations

func (u *User) Verify() {
	if !u.verifiedAt.IsValid() {
		ts := NewTimestampFromTime(time.Now().UTC())
		u.verifiedAt = ts
		u.touchAt(ts)
	}
}

func (u *User) Disable() {
	if !u.disabledAt.IsValid() {
		ts := NewTimestampFromTime(time.Now().UTC())
		u.disabledAt = ts
		u.touchAt(ts)
	}
}

func (u *User) Enable() {
	if u.disabledAt.IsValid() {
		u.disabledAt = Timestamp{}
		u.touch()
	}
}

func (u *User) ScheduleDelete(at time.Time) {
	ts := NewTimestampFromTime(at)
	u.deleteScheduledAt = ts
	u.touchAt(ts)
}

func (u *User) CancelDeletion() {
	if u.deleteScheduledAt.IsValid() {
		u.deleteScheduledAt = Timestamp{}
		u.touch()
	}
}

func (u *User) UpdateEmail(newEmail Email) error {
	if err := u.EnsureActive(); err != nil {
		return err
	}
	if !u.email.Equals(newEmail) {
		u.email = newEmail
		u.touch()
	}
	return nil
}

func (u *User) UpdateUsername(newUsername Username) error {
	if err := u.EnsureActive(); err != nil {
		return err
	}
	if !u.username.Equals(newUsername) {
		u.username = newUsername
		u.touch()
	}
	return nil
}

func (u *User) UpdatePhone(newPhone Phone) error {
	if err := u.EnsureActive(); err != nil {
		return err
	}
	u.phone = newPhone
	u.touch()
	return nil
}

func (u *User) UpdatePassword(newHash Password) error {
	if err := u.EnsureActive(); err != nil {
		return err
	}
	u.passwordHash = newHash
	u.touch()
	return nil
}

func (u *User) UpdateProfile(
	displayName DisplayName,
	bio Bio,
	avatarURL URL,
	bannerColor HexColor,
) error {
	if err := u.EnsureActive(); err != nil {
		return err
	}

	u.displayName = displayName
	u.bio = bio
	u.avatarURL = avatarURL
	u.bannerColor = bannerColor
	u.touch()
	return nil
}

func (u *User) SetPreferredPresence(p PreferredPresence, until Timestamp) error {
	if err := u.EnsureActive(); err != nil {
		return err
	}

	u.preferredPresence = p
	u.preferredPresenceUntil = until
	u.touch()
	return nil
}

func (u *User) EnsureActive() error {
	if u.IsDisabled() {
		return ErrUserDisabled
	}
	if u.IsScheduledForDeletion() {
		return ErrUserScheduledDeletion
	}
	return nil
}

func (u *User) Anonymize() {
	anonID := u.id.String()

	// Construct anonymous value types directly using unsafe/internal mechanics or fallback formats
	u.email, _ = NewEmail(fmt.Sprintf("deleted-%s@deleted.invalid", anonID))
	u.username, _ = NewUsername(fmt.Sprintf("deleted_%s", anonID[:8]))
	u.displayName, _ = NewDisplayName("Deleted User")
	u.passwordHash = Password{}
	u.phone = Phone{}
	u.bio = Bio{}
	u.avatarURL = URL{}
	u.bannerColor = HexColor{}
	u.preferredPresence = PreferredPresence{}
	u.preferredPresenceUntil = Timestamp{}
	u.verifiedAt = Timestamp{}
	u.deleteScheduledAt = Timestamp{}

	ts := NewTimestampFromTime(time.Now().UTC())
	u.disabledAt = ts
	u.touchAt(ts)
}

func (u *User) touch() {
	u.updatedAt = NewTimestampFromTime(time.Now().UTC())
}

func (u *User) touchAt(at Timestamp) {
	u.updatedAt = at
}
