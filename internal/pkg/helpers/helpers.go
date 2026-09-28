package helpers

import (
	"github.com/google/uuid"
)

// FromPtr returns the value pointed to by v, or the type's zero value if v is nil.
func FromPtr[T any](v *T) T {
	if v == nil {
		var zero T
		return zero
	}
	return *v
}

// DedupeIDs removes duplicate and nil UUIDs from a slice while preserving order.
func DedupeIDs(ids []uuid.UUID) []uuid.UUID {
	if len(ids) == 0 {
		return nil
	}
	seen := make(map[uuid.UUID]struct{}, len(ids))
	result := make([]uuid.UUID, 0, len(ids))
	for _, id := range ids {
		if id == uuid.Nil {
			continue
		}
		if _, exists := seen[id]; !exists {
			seen[id] = struct{}{}
			result = append(result, id)
		}
	}
	return result
}

// RemoveID filters out all instances of a target UUID from a slice.
func RemoveID(ids []uuid.UUID, target uuid.UUID) []uuid.UUID {
	if len(ids) == 0 {
		return nil
	}
	result := make([]uuid.UUID, 0, len(ids))
	for _, id := range ids {
		if id != target {
			result = append(result, id)
		}
	}
	return result
}

// ContainsID reports whether target UUID exists within the provided slice of UUIDs.
func ContainsID(ids []uuid.UUID, target uuid.UUID) bool {
	for _, id := range ids {
		if id == target {
			return true
		}
	}
	return false
}

// UUIDToString converts a uuid.UUID to string, encoding uuid.Nil as an empty string.
func UUIDToString(id uuid.UUID) string {
	if id == uuid.Nil {
		return ""
	}
	return id.String()
}

// StringToUUID parses a string into a uuid.UUID, treating empty strings as uuid.Nil.
func StringToUUID(s string) (uuid.UUID, error) {
	if s == "" {
		return uuid.Nil, nil
	}
	return uuid.Parse(s)
}
