package db

import (
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// Integer is a constraint that permits any integer type (signed or unsigned).
type Integer interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64
}

// -----------------------------------------------------------------------------
// Domain -> DB Helpers
// -----------------------------------------------------------------------------

// Int2Ptr converts a pointer to any integer type (including custom uint8/int16 enums) into pgtype.Int2.
func Int2Ptr[T Integer](v *T) pgtype.Int2 {
	if v == nil {
		return pgtype.Int2{Valid: false}
	}
	return pgtype.Int2{Int16: int16(*v), Valid: true}
}

// UUID converts a google/uuid.UUID into pgtype.UUID.
// Note: uuid.Nil (00000000-0000-0000-0000-000000000000) is a valid non-null UUID value in PostgreSQL.
func UUID(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{
		Bytes: id,
		Valid: true,
	}
}

// UUIDPtr converts a *google/uuid.UUID into pgtype.UUID.
func UUIDPtr(id *uuid.UUID) pgtype.UUID {
	if id == nil {
		return pgtype.UUID{Valid: false}
	}
	return pgtype.UUID{
		Bytes: *id,
		Valid: true,
	}
}

// Text converts a *string into pgtype.Text.
func Text(s *string) pgtype.Text {
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
