package db

import (
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// Integer permits signed and unsigned integer types for domain-to-DB mapping.
type Integer interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64
}

// -----------------------------------------------------------------------------
// Domain -> DB (Interpolation via Generics)
// -----------------------------------------------------------------------------

// ToInt2 converts any integer type or custom integer alias into pgtype.Int2.
func ToInt2[T Integer](v T) pgtype.Int2 {
	return pgtype.Int2{
		Int16: int16(v),
		Valid: true,
	}
}

// ToInt2Ptr converts a pointer to any integer type into pgtype.Int2.
func ToInt2Ptr[T Integer](v *T) pgtype.Int2 {
	if v == nil {
		return pgtype.Int2{Valid: false}
	}
	return pgtype.Int2{
		Int16: int16(*v),
		Valid: true,
	}
}

// ToInt4 converts any integer type or custom integer alias into pgtype.Int4.
func ToInt4[T Integer](v T) pgtype.Int4 {
	return pgtype.Int4{
		Int32: int32(v),
		Valid: true,
	}
}

// ToInt4Ptr converts a pointer to any integer type into pgtype.Int4.
func ToInt4Ptr[T Integer](v *T) pgtype.Int4 {
	if v == nil {
		return pgtype.Int4{Valid: false}
	}
	return pgtype.Int4{
		Int32: int32(*v),
		Valid: true,
	}
}

// ToText converts a string or custom string type alias into pgtype.Text.
func ToText[T ~string](s T) pgtype.Text {
	return pgtype.Text{String: string(s), Valid: true}
}

// ToTextPtr converts a pointer to a string or custom string type alias into pgtype.Text.
func ToTextPtr[T ~string](s *T) pgtype.Text {
	if s == nil {
		return pgtype.Text{Valid: false}
	}
	return pgtype.Text{String: string(*s), Valid: true}
}

// ToTimestamptz converts a time.Time into pgtype.Timestamptz in UTC.
func ToTimestamptz(t time.Time) pgtype.Timestamptz {
	if t.IsZero() {
		return pgtype.Timestamptz{Valid: false}
	}
	return pgtype.Timestamptz{
		Time:  t.UTC(),
		Valid: true,
	}
}

// ToTimestamptzPtr converts a *time.Time into pgtype.Timestamptz in UTC.
func ToTimestamptzPtr(t *time.Time) pgtype.Timestamptz {
	if t == nil || t.IsZero() {
		return pgtype.Timestamptz{Valid: false}
	}
	return pgtype.Timestamptz{
		Time:  t.UTC(),
		Valid: true,
	}
}

// ToUUID converts a uuid.UUID into pgtype.UUID.
func ToUUID(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{
		Bytes: id,
		Valid: true,
	}
}

// ToUUIDPtr converts a *uuid.UUID into pgtype.UUID.
func ToUUIDPtr(id *uuid.UUID) pgtype.UUID {
	if id == nil {
		return pgtype.UUID{Valid: false}
	}
	return pgtype.UUID{
		Bytes: *id,
		Valid: true,
	}
}

// ToUUIDs converts a slice of uuid.UUID into []pgtype.UUID.
func ToUUIDs(ids []uuid.UUID) []pgtype.UUID {
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
// DB -> Domain (Direct Concrete Typing)
// -----------------------------------------------------------------------------

// FromInt2 converts pgtype.Int2 into an int, returning 0 if NULL.
func FromInt2(i pgtype.Int2) int {
	if !i.Valid {
		return 0
	}
	return int(i.Int16)
}

// FromInt2Ptr converts pgtype.Int2 into an *int, returning nil if NULL.
func FromInt2Ptr(i pgtype.Int2) *int {
	if !i.Valid {
		return nil
	}
	v := int(i.Int16)
	return &v
}

// FromInt4 converts pgtype.Int4 into an int, returning 0 if NULL.
func FromInt4(i pgtype.Int4) int {
	if !i.Valid {
		return 0
	}
	return int(i.Int32)
}

// FromInt4Ptr converts pgtype.Int4 into an *int, returning nil if NULL.
func FromInt4Ptr(i pgtype.Int4) *int {
	if !i.Valid {
		return nil
	}
	v := int(i.Int32)
	return &v
}

// FromText converts pgtype.Text into a string, returning an empty string if NULL.
func FromText(t pgtype.Text) string {
	if !t.Valid {
		return ""
	}
	return t.String
}

// FromTextPtr converts pgtype.Text into a *string, returning nil if NULL.
func FromTextPtr(t pgtype.Text) *string {
	if !t.Valid {
		return nil
	}
	v := t.String
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

// FromUUID converts pgtype.UUID into uuid.UUID, returning uuid.Nil if NULL.
func FromUUID(id pgtype.UUID) uuid.UUID {
	if !id.Valid {
		return uuid.Nil
	}
	return uuid.UUID(id.Bytes)
}

// FromUUIDPtr converts pgtype.UUID into *uuid.UUID, returning nil if NULL.
func FromUUIDPtr(id pgtype.UUID) *uuid.UUID {
	if !id.Valid {
		return nil
	}
	v := uuid.UUID(id.Bytes)
	return &v
}

// FromUUIDs converts a slice of pgtype.UUID into []uuid.UUID, skipping invalid elements.
func FromUUIDs(ids []pgtype.UUID) []uuid.UUID {
	if ids == nil {
		return nil
	}
	res := make([]uuid.UUID, 0, len(ids))
	for _, id := range ids {
		if !id.Valid {
			continue
		}
		res = append(res, uuid.UUID(id.Bytes))
	}
	return res
}
