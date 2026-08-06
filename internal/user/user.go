package user

import (
	"errors"
	"time"

	"bonfire-api/internal/pkg/ptr"
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
	phone                  *Phone
	bio                    *Bio
	avatarURL              *URL
	bannerColor            *HexCode
	preferredPresence      *PreferredPresence
	preferredPresenceUntil *time.Time
	verifiedAt             *time.Time
	disabledAt             *time.Time
	deleteScheduledAt      *time.Time
	createdAt              time.Time
	updatedAt              time.Time
}

func (u *User) ID() ID                                { return u.id }
func (u *User) Email() Email                          { return u.email }
func (u *User) Username() Username                    { return u.username }
func (u *User) DisplayName() DisplayName              { return u.displayName }
func (u *User) PasswordHash() Password                { return u.passwordHash }
func (u *User) Phone() *Phone                         { return u.phone }
func (u *User) Bio() *Bio                             { return u.bio }
func (u *User) AvatarURL() *URL                       { return u.avatarURL }
func (u *User) BannerColor() *HexCode                 { return u.bannerColor }
func (u *User) PreferredPresence() *PreferredPresence { return u.preferredPresence }
func (u *User) PreferredPresenceUntil() *time.Time    { return u.preferredPresenceUntil }
func (u *User) VerifiedAt() *time.Time                { return u.verifiedAt }
func (u *User) DisabledAt() *time.Time                { return u.disabledAt }
func (u *User) DeleteScheduledAt() *time.Time         { return u.deleteScheduledAt }
func (u *User) CreatedAt() time.Time                  { return u.createdAt }
func (u *User) UpdatedAt() time.Time                  { return u.updatedAt }

func (u *User) IsVerified() bool             { return u.verifiedAt != nil }
func (u *User) IsDisabled() bool             { return u.disabledAt != nil }
func (u *User) IsScheduledForDeletion() bool { return u.deleteScheduledAt != nil }

func New(
	id ID,
	email Email,
	username Username,
	displayName DisplayName,
	passwordHash Password,
) (*User, error) {
	now := time.Now().UTC()
	return &User{
		id:           id,
		email:        email,
		username:     username,
		displayName:  displayName,
		passwordHash: passwordHash,
		createdAt:    now,
		updatedAt:    now,
	}, nil
}

func Reconstitute(
	id ID,
	email Email,
	username Username,
	displayName DisplayName,
	passwordHash Password,
	phone *Phone,
	bio *Bio,
	avatarURL *URL,
	bannerColor *HexCode,
	preferredPresence *PreferredPresence,
	preferredPresenceUntil *time.Time,
	verifiedAt *time.Time,
	disabledAt *time.Time,
	deleteScheduledAt *time.Time,
	createdAt, updatedAt time.Time,
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
		preferredPresenceUntil: ptr.Map(preferredPresenceUntil, time.Time.UTC),
		verifiedAt:             ptr.Map(verifiedAt, time.Time.UTC),
		disabledAt:             ptr.Map(disabledAt, time.Time.UTC),
		deleteScheduledAt:      ptr.Map(deleteScheduledAt, time.Time.UTC),
		createdAt:              createdAt.UTC(),
		updatedAt:              updatedAt.UTC(),
	}
}

func (u *User) Verify(at time.Time) {
	if u.verifiedAt == nil {
		u.verifiedAt = ptr.To(at.UTC())
		u.touchAt(at)
	}
}

func (u *User) Disable(at time.Time) {
	if u.disabledAt == nil {
		u.disabledAt = ptr.To(at.UTC())
		u.touchAt(at)
	}
}

func (u *User) Enable() {
	if u.disabledAt != nil {
		u.disabledAt = nil
		u.touch()
	}
}

func (u *User) ScheduleDeletion(at time.Time) {
	u.deleteScheduledAt = ptr.To(at.UTC())
	u.touchAt(at)
}

func (u *User) CancelDeletion() {
	if u.deleteScheduledAt != nil {
		u.deleteScheduledAt = nil
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

func (u *User) UpdatePhone(newPhone *Phone) error {
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
	bio *Bio,
	avatarURL *URL,
	bannerColor *HexCode,
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

func (u *User) SetPreferredPresence(p *PreferredPresence, until *time.Time) error {
	if err := u.ensureActive(); err != nil {
		return err
	}

	u.preferredPresence = p
	u.preferredPresenceUntil = ptr.Map(until, time.Time.UTC)
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
	u.updatedAt = time.Now().UTC()
}

func (u *User) touchAt(at time.Time) {
	u.updatedAt = at.UTC()
}
