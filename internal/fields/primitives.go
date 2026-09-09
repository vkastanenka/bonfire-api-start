package fields

import (
	"bonfire-api/internal/errs"
	"bonfire-api/internal/sanitize"
	"bytes"

	"github.com/google/uuid"
)

// ============================================================================
// ID
// ============================================================================

type ID uuid.UUID

func NewID() (ID, error) {
	u, err := uuid.NewV7()
	if err != nil {
		return ID{}, errs.Internal("unable to create new id").Wrap(err)
	}
	return ID(u), nil
}

func ParseID(raw uuid.UUID) ID {
	return ID(raw)
}

func ParseIDString(raw string) (ID, error) {
	s := sanitize.UUID(raw)
	if s == "" {
		return ID{}, nil
	}

	u, err := uuid.Parse(s)
	if err != nil {
		return ID{}, errs.InvalidArgument("invalid id string").Wrap(err)
	}

	return ParseID(u), nil
}

func ParseIDs(raws []uuid.UUID) []ID {
	if len(raws) == 0 {
		return nil
	}

	ids := make([]ID, 0, len(raws))
	for _, raw := range raws {
		id := ParseID(raw)
		if id.IsZero() {
			continue
		}
		ids = append(ids, id)
	}

	return ids
}

func ToUUIDs(ids []ID) []uuid.UUID {
	if len(ids) == 0 {
		return nil
	}
	result := make([]uuid.UUID, len(ids))
	for i, id := range ids {
		result[i] = id.UUID()
	}
	return result
}

func DedupeIDs(ids []ID) []ID {
	if len(ids) == 0 {
		return nil
	}
	seen := make(map[ID]struct{}, len(ids))
	result := make([]ID, 0, len(ids))
	for _, id := range ids {
		if id.IsZero() {
			continue
		}
		if _, exists := seen[id]; !exists {
			seen[id] = struct{}{}
			result = append(result, id)
		}
	}
	return result
}

func RemoveID(target ID, ids []ID) []ID {
	if len(ids) == 0 {
		return nil
	}
	result := make([]ID, 0, len(ids))
	for _, id := range ids {
		if !id.Equals(target) {
			result = append(result, id)
		}
	}
	return result
}

func SortIDPair(id1, id2 ID) (ID, ID) {
	if id1.Compare(id2) < 0 {
		return id1, id2
	}
	return id2, id1
}

func (id ID) UUID() uuid.UUID      { return uuid.UUID(id) }
func (id ID) String() string       { return uuid.UUID(id).String() }
func (id ID) IsZero() bool         { return uuid.UUID(id) == uuid.Nil }
func (id ID) IsValid() bool        { return !id.IsZero() }
func (id ID) Equals(other ID) bool { return id == other }
func (id ID) Version() int         { return int(uuid.UUID(id).Version()) }
func (id ID) IsV7() bool           { return id.Version() == 7 }
func (id ID) IsV4() bool           { return id.Version() == 4 }

func (id ID) Compare(other ID) int {
	return bytes.Compare(id[:], other[:])
}

func (id ID) UUIDPtr() *uuid.UUID {
	if id.IsZero() {
		return nil
	}
	uid := id.UUID()
	return &uid
}

func (id ID) StringPtr() *string {
	if id.IsZero() {
		return nil
	}
	s := id.String()
	return &s
}

func (id ID) MarshalText() ([]byte, error) {
	if id.IsZero() {
		return []byte(""), nil
	}
	return []byte(id.String()), nil
}

func (id *ID) UnmarshalText(text []byte) error {
	parsed, err := ParseIDString(string(text))
	if err != nil {
		return err
	}
	*id = parsed
	return nil
}

func (id ID) MarshalJSON() ([]byte, error) {
	if id.IsZero() {
		return []byte("null"), nil
	}
	return []byte(`"` + id.String() + `"`), nil
}

func (id *ID) UnmarshalJSON(data []byte) error {
	if len(data) == 0 || bytes.Equal(data, []byte("null")) {
		*id = ID{}
		return nil
	}

	parsed, err := ParseIDString(string(data))
	if err != nil {
		return err
	}

	*id = parsed
	return nil
}
