package db

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// 	   ID                     pgtype.UUID        `json:"id"`
//     Email                  string             `json:"email"`
//     Username               string             `json:"username"`
//     DisplayName            string             `json:"display_name"`
//     PasswordHash           string             `json:"password_hash"`
//     Phone                  pgtype.Text        `json:"phone"`
//     Bio                    pgtype.Text        `json:"bio"`
//     AvatarUrl              pgtype.Text        `json:"avatar_url"`
//     BannerColor            pgtype.Text        `json:"banner_color"`
//     PreferredPresence      pgtype.Int2        `json:"preferred_presence"`
//     PreferredPresenceUntil pgtype.Timestamptz `json:"preferred_presence_until"`
//     VerifiedAt             pgtype.Timestamptz `json:"verified_at"`
//     DisabledAt             pgtype.Timestamptz `json:"disabled_at"`
//     DeleteScheduledAt      pgtype.Timestamptz `json:"delete_scheduled_at"`
//     CreatedAt              pgtype.Timestamptz `json:"created_at"`
//     UpdatedAt              pgtype.Timestamptz `json:"updated_at"`

// Integer is a constraint that permits any integer type (signed or unsigned).
type Integer interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64
}

// StringLike permits any string underlying type or types implementing fmt.Stringer.
type Stringer interface {
	fmt.Stringer
}

// -----------------------------------------------------------------------------
// Domain -> DB Helpers
// -----------------------------------------------------------------------------

// Int2 converts any integer value (including custom enums) into pgtype.Int2.
func Int2[T Integer](v T) pgtype.Int2 {
	return pgtype.Int2{
		Int16: int16(v),
		Valid: true,
	}
}

// Int2Ptr converts a pointer to any integer type into pgtype.Int2.
func Int2Ptr[T Integer](v *T) pgtype.Int2 {
	if v == nil {
		return pgtype.Int2{Valid: false}
	}
	return pgtype.Int2{
		Int16: int16(*v),
		Valid: true,
	}
}

// Text converts a string or fmt.Stringer into pgtype.Text.
func Text(s string) pgtype.Text {
	return pgtype.Text{String: s, Valid: true}
}

// TextPtr converts a *string into pgtype.Text.
func TextPtr(s *string) pgtype.Text {
	if s == nil {
		return pgtype.Text{Valid: false}
	}
	return pgtype.Text{String: *s, Valid: true}
}

// Timestamptz converts a time.Time into pgtype.Timestamptz in UTC.
func Timestamptz(t time.Time) pgtype.Timestamptz {
	if t.IsZero() {
		return pgtype.Timestamptz{Valid: false}
	}
	return pgtype.Timestamptz{
		Time:  t.UTC(),
		Valid: true,
	}
}

// TimestamptzPtr converts a *time.Time into pgtype.Timestamptz in UTC.
func TimestamptzPtr(t *time.Time) pgtype.Timestamptz {
	if t == nil || t.IsZero() {
		return pgtype.Timestamptz{Valid: false}
	}
	return pgtype.Timestamptz{
		Time:  t.UTC(),
		Valid: true,
	}
}

// UUID converts a google/uuid.UUID into pgtype.UUID.
func UUID[T ~[16]byte](id T) pgtype.UUID {
	return pgtype.UUID{
		Bytes: id,
		Valid: true,
	}
}

// UUIDPtr converts a *google/uuid.UUID into pgtype.UUID.
func UUIDPtr[T ~[16]byte](id *T) pgtype.UUID {
	if id == nil {
		return pgtype.UUID{Valid: false}
	}
	return pgtype.UUID{
		Bytes: *id,
		Valid: true,
	}
}

// StringerPtr converts a pointer to any type whose pointer receiver implements fmt.Stringer into pgtype.Text.
// Works seamlessly with value object pointers like *channel.Name, *channel.IconURL, etc.
func StringerPtr[T any, PT interface {
	*T
	fmt.Stringer
}](v *T) pgtype.Text {
	if v == nil {
		return pgtype.Text{Valid: false}
	}
	return pgtype.Text{String: PT(v).String(), Valid: true}
}

// -----------------------------------------------------------------------------
// DB -> Domain Helpers
// -----------------------------------------------------------------------------

// UUIDPtrFromDB converts pgtype.UUID into a *google/uuid.UUID.
func UUIDPtrFromDB(u pgtype.UUID) *uuid.UUID {
	if !u.Valid {
		return nil
	}
	id := uuid.UUID(u.Bytes)
	return &id
}

// StringPtr converts pgtype.Text into a *string.
func StringPtr(s pgtype.Text) *string {
	if !s.Valid {
		return nil
	}
	return &s.String
}

// TimePtr converts pgtype.Timestamptz into a *time.Time.
func TimePtr(t pgtype.Timestamptz) *time.Time {
	if !t.Valid {
		return nil
	}
	utcTime := t.Time.UTC()
	return &utcTime
}

// Int16Ptr converts pgtype.Int2 into a pointer to a target Integer/Enum type.
func Int16Ptr[T Integer](i pgtype.Int2) *T {
	if !i.Valid {
		return nil
	}
	val := T(i.Int16)
	return &val
}

func ToStringPtr[T any, PT interface {
	*T
	fmt.Stringer
}](v *T) *string {
	if v == nil {
		return nil
	}
	s := PT(v).String()
	return &s
}
