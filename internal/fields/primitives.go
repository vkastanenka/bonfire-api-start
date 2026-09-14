package fields

import (
	"bytes"

	"github.com/google/uuid"
)

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

func SortIDPair(id1, id2 uuid.UUID) (uuid.UUID, uuid.UUID) {
	if bytes.Compare(id1[:], id2[:]) < 0 {
		return id1, id2
	}
	return id2, id1
}
