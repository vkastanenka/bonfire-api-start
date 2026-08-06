package user

import (
	"errors"
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

func (u *User) IsVerified() bool             { return u.verifiedAt.IsValid() }
func (u *User) IsDisabled() bool             { return u.disabledAt.IsValid() }
func (u *User) IsScheduledForDeletion() bool { return u.deleteScheduledAt.IsValid() }

func New(
	id ID,
	email Email,
	username Username,
	displayName DisplayName,
	passwordHash Password,
) (*User, error) {
	now := time.Now().UTC()
	timestamp := NewTimestampFromTime(&now)
	return &User{
		id:           id,
		email:        email,
		username:     username,
		displayName:  displayName,
		passwordHash: passwordHash,
		createdAt:    timestamp,
		updatedAt:    timestamp,
	}, nil
}

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

func (u *User) Verify() {
	if !u.verifiedAt.IsValid() {
		now := time.Now().UTC()
		u.verifiedAt = NewTimestampFromTime(&now)
		u.touchAt(NewTimestampFromTime(&now))
	}
}

func (u *User) Disable() {
	if !u.disabledAt.IsValid() {
		now := time.Now().UTC()
		u.disabledAt = NewTimestampFromTime(&now)
		u.touchAt(NewTimestampFromTime(&now))
	}
}

func (u *User) Enable() {
	if u.disabledAt.IsValid() {
		u.disabledAt = Timestamp{}
		u.touch()
	}
}

func (u *User) ScheduleDelete(at time.Time) {
	u.deleteScheduledAt = NewTimestampFromTime(&at)
	u.touchAt(NewTimestampFromTime(&at))
}

func (u *User) CancelDeletion() {
	if u.deleteScheduledAt.IsValid() {
		u.deleteScheduledAt = Timestamp{}
		u.touch()
	}
}

func (u *User) UpdateEmail(newEmail Email) error {
	if err := u.ensureActive(); err != nil {
		return err
	}
	if !u.email.Equals(newEmail) {
		u.email = newEmail
		u.touch()
	}
	return nil
}

func (u *User) UpdateUsername(newUsername Username) error {
	if err := u.ensureActive(); err != nil {
		return err
	}
	if !u.username.Equals(newUsername) {
		u.username = newUsername
		u.touch()
	}
	return nil
}

func (u *User) UpdatePhone(newPhone Phone) error {
	if err := u.ensureActive(); err != nil {
		return err
	}
	u.phone = newPhone
	u.touch()
	return nil
}

func (u *User) UpdatePassword(newHash Password) error {
	if err := u.ensureActive(); err != nil {
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
	if err := u.ensureActive(); err != nil {
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
	if err := u.ensureActive(); err != nil {
		return err
	}

	u.preferredPresence = p
	u.preferredPresenceUntil = NewTimestampFromTime(until.value)
	u.touch()
	return nil
}

func (u *User) ensureActive() error {
	if u.IsDisabled() {
		return ErrUserDisabled
	}
	if u.IsScheduledForDeletion() {
		return ErrUserScheduledDeletion
	}
	return nil
}

func (u *User) touch() {
	now := time.Now()
	u.updatedAt = NewTimestampFromTime(&now)
}

func (u *User) touchAt(at Timestamp) {
	u.updatedAt = NewTimestampFromTime(at.value)
}
