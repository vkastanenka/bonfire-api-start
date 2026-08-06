package user

import (
	"errors"
	"fmt"
	"time"

	"bonfire-api/internal/fields"
)

var (
	ErrUserDisabled          = errors.New("cannot perform action on a disabled user account")
	ErrUserScheduledDeletion = errors.New("cannot perform action on an account scheduled for deletion")
)

type User struct {
	id                     fields.ID
	email                  Email
	username               Username
	displayName            DisplayName
	passwordHash           PasswordHash
	phone                  Phone
	bio                    Bio
	avatarURL              fields.URL
	bannerColor            fields.HexColor
	preferredPresence      PreferredPresence
	preferredPresenceUntil fields.Timestamp
	verifiedAt             fields.Timestamp
	disabledAt             fields.Timestamp
	deleteScheduledAt      fields.Timestamp
	createdAt              fields.Timestamp
	updatedAt              fields.Timestamp
}

func (u *User) ID() fields.ID                            { return u.id }
func (u *User) Email() Email                             { return u.email }
func (u *User) Username() Username                       { return u.username }
func (u *User) DisplayName() DisplayName                 { return u.displayName }
func (u *User) PasswordHash() PasswordHash               { return u.passwordHash }
func (u *User) Phone() Phone                             { return u.phone }
func (u *User) Bio() Bio                                 { return u.bio }
func (u *User) AvatarURL() fields.URL                    { return u.avatarURL }
func (u *User) BannerColor() fields.HexColor             { return u.bannerColor }
func (u *User) PreferredPresence() PreferredPresence     { return u.preferredPresence }
func (u *User) PreferredPresenceUntil() fields.Timestamp { return u.preferredPresenceUntil }
func (u *User) VerifiedAt() fields.Timestamp             { return u.verifiedAt }
func (u *User) DisabledAt() fields.Timestamp             { return u.disabledAt }
func (u *User) DeleteScheduledAt() fields.Timestamp      { return u.deleteScheduledAt }
func (u *User) CreatedAt() fields.Timestamp              { return u.createdAt }
func (u *User) UpdatedAt() fields.Timestamp              { return u.updatedAt }

func (u *User) IsVerified() bool             { return u.verifiedAt.IsValid() }
func (u *User) IsDisabled() bool             { return u.disabledAt.IsValid() }
func (u *User) IsScheduledForDeletion() bool { return u.deleteScheduledAt.IsValid() }

func (u *User) EffectivePresence(now time.Time) PreferredPresence {
	if u.preferredPresenceUntil.HasPassed(now) {
		return PreferredPresence{}
	}
	return u.preferredPresence
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

func New(
	id fields.ID,
	email Email,
	username Username,
	displayName DisplayName,
	passwordHash PasswordHash,
	now time.Time,
) (*User, error) {
	ts := fields.NewTimestampFromTime(now)
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

func Reconstitute(
	id fields.ID,
	email Email,
	username Username,
	passwordHash PasswordHash,
	phone Phone,
	displayName DisplayName,
	bio Bio,
	avatarURL fields.URL,
	bannerColor fields.HexColor,
	preferredPresence PreferredPresence,
	preferredPresenceUntil, verifiedAt, disabledAt, deleteScheduledAt, createdAt, updatedAt fields.Timestamp,
) *User {
	return &User{
		id:                     id,
		email:                  email,
		username:               username,
		passwordHash:           passwordHash,
		phone:                  phone,
		displayName:            displayName,
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

func (u *User) Verify(now time.Time) {
	if !u.verifiedAt.IsValid() {
		ts := fields.NewTimestampFromTime(now)
		u.verifiedAt = ts
		u.touch(ts)
	}
}

func (u *User) Disable(now time.Time) {
	if !u.disabledAt.IsValid() {
		ts := fields.NewTimestampFromTime(now)
		u.disabledAt = ts
		u.touch(ts)
	}
}

func (u *User) Enable(now time.Time) {
	if u.disabledAt.IsValid() {
		u.disabledAt = fields.Timestamp{}
		u.touch(fields.NewTimestampFromTime(now))
	}
}

func (u *User) ScheduleDelete(scheduledAt time.Time, now time.Time) {
	u.deleteScheduledAt = fields.NewTimestampFromTime(scheduledAt)
	u.touch(fields.NewTimestampFromTime(now))
}

func (u *User) CancelDelete(now time.Time) {
	if u.deleteScheduledAt.IsValid() {
		u.deleteScheduledAt = fields.Timestamp{}
		u.touch(fields.NewTimestampFromTime(now))
	}
}

func (u *User) UpdateEmail(newEmail Email, now time.Time) error {
	if err := u.EnsureActive(); err != nil {
		return err
	}
	if !u.email.Equals(newEmail) {
		u.email = newEmail
		u.touch(fields.NewTimestampFromTime(now))
	}
	return nil
}

func (u *User) UpdateUsername(newUsername Username, now time.Time) error {
	if err := u.EnsureActive(); err != nil {
		return err
	}
	if !u.username.Equals(newUsername) {
		u.username = newUsername
		u.touch(fields.NewTimestampFromTime(now))
	}
	return nil
}

func (u *User) UpdatePhone(newPhone Phone, now time.Time) error {
	if err := u.EnsureActive(); err != nil {
		return err
	}
	if !u.phone.Equals(newPhone) {
		u.phone = newPhone
		u.touch(fields.NewTimestampFromTime(now))
	}
	return nil
}

func (u *User) UpdatePasswordHash(newHash PasswordHash, now time.Time) error {
	if err := u.EnsureActive(); err != nil {
		return err
	}
	u.passwordHash = newHash
	u.touch(fields.NewTimestampFromTime(now))
	return nil
}

func (u *User) UpdateProfile(displayName DisplayName, bio Bio, avatarURL fields.URL, bannerColor fields.HexColor, now time.Time) error {
	if err := u.EnsureActive(); err != nil {
		return err
	}
	u.displayName = displayName
	u.bio = bio
	u.avatarURL = avatarURL
	u.bannerColor = bannerColor
	u.touch(fields.NewTimestampFromTime(now))
	return nil
}

func (u *User) UpdatePreferredPresence(p PreferredPresence, until fields.Timestamp, now time.Time) error {
	if err := u.EnsureActive(); err != nil {
		return err
	}
	u.preferredPresence = p
	u.preferredPresenceUntil = until
	u.touch(fields.NewTimestampFromTime(now))
	return nil
}

func (u *User) Anonymize(now time.Time) {
	anonID := u.id.String()
	ts := fields.NewTimestampFromTime(now)

	u.email = Email{value: fmt.Sprintf("deleted-%s@deleted.invalid", anonID)}
	u.username = Username{value: fmt.Sprintf("deleted_%s", anonID[:8])}
	u.displayName = DisplayName{value: "Deleted User"}
	u.bio = Bio{}
	u.avatarURL = fields.URL{}
	u.bannerColor = fields.HexColor{}
	u.passwordHash = PasswordHash{}
	u.phone = Phone{}
	u.preferredPresence = PreferredPresence{}
	u.preferredPresenceUntil = fields.Timestamp{}
	u.verifiedAt = fields.Timestamp{}
	u.deleteScheduledAt = fields.Timestamp{}
	u.disabledAt = ts
	u.touch(ts)
}

func (u *User) touch(at fields.Timestamp) {
	u.updatedAt = at
}
