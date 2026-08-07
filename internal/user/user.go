package user

import (
	"errors"
	"fmt"

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

func (u *User) EffectivePresence(now fields.Timestamp) PreferredPresence {
	if u.preferredPresenceUntil.HasPassed(now.Time()) {
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
	now fields.Timestamp,
) (*User, error) {
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

func (u *User) Verify(now fields.Timestamp) {
	if !u.verifiedAt.IsValid() {
		u.verifiedAt = now
		u.touch(now)
	}
}

func (u *User) Disable(now fields.Timestamp) {
	if !u.disabledAt.IsValid() {
		u.disabledAt = now
		u.touch(now)
	}
}

func (u *User) Enable(now fields.Timestamp) {
	if u.disabledAt.IsValid() {
		u.disabledAt = fields.Timestamp{}
		u.touch(now)
	}
}

func (u *User) ScheduleDelete(scheduledAt fields.Timestamp, now fields.Timestamp) {
	u.deleteScheduledAt = scheduledAt
	u.touch(now)
}

func (u *User) CancelDelete(now fields.Timestamp) {
	if u.deleteScheduledAt.IsValid() {
		u.deleteScheduledAt = fields.Timestamp{}
		u.touch(now)
	}
}

func (u *User) UpdateEmail(newEmail Email, now fields.Timestamp) error {
	if err := u.EnsureActive(); err != nil {
		return err
	}
	if !u.email.Equals(newEmail) {
		u.email = newEmail
		u.touch(now)
	}
	return nil
}

func (u *User) UpdateUsername(newUsername Username, now fields.Timestamp) error {
	if err := u.EnsureActive(); err != nil {
		return err
	}
	if !u.username.Equals(newUsername) {
		u.username = newUsername
		u.touch(now)
	}
	return nil
}

func (u *User) UpdatePhone(newPhone Phone, now fields.Timestamp) error {
	if err := u.EnsureActive(); err != nil {
		return err
	}
	if !u.phone.Equals(newPhone) {
		u.phone = newPhone
		u.touch(now)
	}
	return nil
}

func (u *User) UpdatePasswordHash(newHash PasswordHash, now fields.Timestamp) error {
	if err := u.EnsureActive(); err != nil {
		return err
	}
	u.passwordHash = newHash
	u.touch(now)
	return nil
}

func (u *User) UpdateProfile(displayName DisplayName, bio Bio, avatarURL fields.URL, bannerColor fields.HexColor, now fields.Timestamp) error {
	if err := u.EnsureActive(); err != nil {
		return err
	}
	u.displayName = displayName
	u.bio = bio
	u.avatarURL = avatarURL
	u.bannerColor = bannerColor
	u.touch(now)
	return nil
}

func (u *User) UpdatePreferredPresence(p PreferredPresence, until fields.Timestamp, now fields.Timestamp) error {
	if err := u.EnsureActive(); err != nil {
		return err
	}
	u.preferredPresence = p
	u.preferredPresenceUntil = until
	u.touch(now)
	return nil
}

func (u *User) Anonymize(now fields.Timestamp) {
	anonID := u.id.String()

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
	u.disabledAt = now
	u.touch(now)
}

func (u *User) touch(at fields.Timestamp) {
	u.updatedAt = at
}
