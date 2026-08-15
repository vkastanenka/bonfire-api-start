package db

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

// Integer is a constraint that permits any integer type (signed or unsigned).
type Integer interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64
}

// -----------------------------------------------------------------------------
// Domain -> DB
// -----------------------------------------------------------------------------

// Int2 converts any integer value (including custom enums) into pgtype.Int2.
func ToInt2[T Integer](v T) pgtype.Int2 {
	return pgtype.Int2{
		Int16: int16(v),
		Valid: true,
	}
}

// Int2Ptr converts a pointer to any integer type into pgtype.Int2.
func ToInt2Ptr[T Integer](v *T) pgtype.Int2 {
	if v == nil {
		return pgtype.Int2{Valid: false}
	}
	return pgtype.Int2{
		Int16: int16(*v),
		Valid: true,
	}
}

// Int4 converts any integer value (including custom enums) into pgtype.Int4.
func ToInt4[T Integer](v T) pgtype.Int4 {
	return pgtype.Int4{
		Int32: int32(v),
		Valid: true,
	}
}

// Int4Ptr converts a pointer to any integer type into pgtype.Int4.
func ToInt4Ptr[T Integer](v *T) pgtype.Int4 {
	if v == nil {
		return pgtype.Int4{Valid: false}
	}
	return pgtype.Int4{
		Int32: int32(*v),
		Valid: true,
	}
}

// Text converts a string or fmt.Stringer into pgtype.Text.
func ToText(s string) pgtype.Text {
	return pgtype.Text{String: s, Valid: true}
}

// TextPtr converts a *string into pgtype.Text.
func ToTextPtr(s *string) pgtype.Text {
	if s == nil {
		return pgtype.Text{Valid: false}
	}
	return pgtype.Text{String: *s, Valid: true}
}

// Timestamptz converts a time.Time into pgtype.Timestamptz in UTC.
func ToTimestamptz(t time.Time) pgtype.Timestamptz {
	if t.IsZero() {
		return pgtype.Timestamptz{Valid: false}
	}
	return pgtype.Timestamptz{
		Time:  t.UTC(),
		Valid: true,
	}
}

// TimestamptzPtr converts a *time.Time into pgtype.Timestamptz in UTC.
func ToTimestamptzPtr(t *time.Time) pgtype.Timestamptz {
	if t == nil || t.IsZero() {
		return pgtype.Timestamptz{Valid: false}
	}
	return pgtype.Timestamptz{
		Time:  t.UTC(),
		Valid: true,
	}
}

// UUID converts a google/uuid.UUID into pgtype.UUID.
func ToUUID[T ~[16]byte](id T) pgtype.UUID {
	return pgtype.UUID{
		Bytes: id,
		Valid: true,
	}
}

// UUIDPtr converts a *google/uuid.UUID into pgtype.UUID.
func ToUUIDPtr[T ~[16]byte](id *T) pgtype.UUID {
	if id == nil {
		return pgtype.UUID{Valid: false}
	}
	return pgtype.UUID{
		Bytes: *id,
		Valid: true,
	}
}

// ToUUIDs converts a slice of 16-byte arrays (e.g., []uuid.UUID) into []pgtype.UUID.
func ToUUIDs[T ~[16]byte](ids []T) []pgtype.UUID {
	if ids == nil {
		return nil
	}
	res := make([]pgtype.UUID, len(ids))
	for i, id := range ids {
		res[i] = pgtype.UUID{
			Bytes: id,
			Valid: true,
		}
	}
	return res
}

// -----------------------------------------------------------------------------
// DB -> Domain
// -----------------------------------------------------------------------------

// FromInt2 converts pgtype.Int2 into any integer type or custom enum, returning zero if NULL.
func FromInt2[T Integer](i pgtype.Int2) T {
	if !i.Valid {
		var zero T
		return zero
	}
	return T(i.Int16)
}

// FromInt2Ptr converts pgtype.Int2 into a pointer to any integer type, returning nil if NULL.
func FromInt2Ptr[T Integer](i pgtype.Int2) *T {
	if !i.Valid {
		return nil
	}
	v := T(i.Int16)
	return &v
}

// FromInt4 converts pgtype.Int4 into any integer type or custom enum, returning zero if NULL.
func FromInt4[T Integer](i pgtype.Int4) T {
	if !i.Valid {
		var zero T
		return zero
	}
	return T(i.Int32)
}

// FromInt4Ptr converts pgtype.Int4 into a pointer to any integer type, returning nil if NULL.
func FromInt4Ptr[T Integer](i pgtype.Int4) *T {
	if !i.Valid {
		return nil
	}
	v := T(i.Int32)
	return &v
}

// FromText converts pgtype.Text into a string or custom string type, returning empty string if NULL.
func FromText[T ~string](t pgtype.Text) T {
	if !t.Valid {
		var zero T
		return zero
	}
	return T(t.String)
}

// FromTextPtr converts pgtype.Text into a *string or *custom string type, returning nil if NULL.
func FromTextPtr[T ~string](t pgtype.Text) *T {
	if !t.Valid {
		return nil
	}
	v := T(t.String)
	return &v
}

// FromTimestamptz converts pgtype.Timestamptz into time.Time, returning zero time if NULL.
func FromTimestamptz(t pgtype.Timestamptz) time.Time {
	if !t.Valid {
		return time.Time{}
	}
	return t.Time
}

// FromTimestamptzPtr converts pgtype.Timestamptz into *time.Time, returning nil if NULL.
func FromTimestamptzPtr(t pgtype.Timestamptz) *time.Time {
	if !t.Valid {
		return nil
	}
	v := t.Time
	return &v
}

// FromUUID converts pgtype.UUID into a 16-byte array (e.g., uuid.UUID), returning zero value if NULL.
func FromUUID[T ~[16]byte](id pgtype.UUID) T {
	if !id.Valid {
		var zero T
		return zero
	}
	return T(id.Bytes)
}

// FromUUIDPtr converts pgtype.UUID into a pointer to a 16-byte array, returning nil if NULL.
func FromUUIDPtr[T ~[16]byte](id pgtype.UUID) *T {
	if !id.Valid {
		return nil
	}
	v := T(id.Bytes)
	return &v
}

// FromUUIDs converts a slice of pgtype.UUID into a slice of 16-byte arrays (e.g., []uuid.UUID).
func FromUUIDs[T ~[16]byte](ids []pgtype.UUID) []T {
	if ids == nil {
		return nil
	}
	res := make([]T, 0, len(ids))
	for _, id := range ids {
		if !id.Valid {
			continue
		}
		res = append(res, T(id.Bytes))
	}
	return res
}
