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
func RemoveID(target uuid.UUID, ids []uuid.UUID) []uuid.UUID {
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
