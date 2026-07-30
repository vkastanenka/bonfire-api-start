package relationship

import (
	"errors"
	"strings"

	"github.com/google/uuid"
)

var (
	ErrIDNil     = errors.New("id cannot be nil or zero-value")
	ErrIDInvalid = errors.New("invalid uuid format")
)

// -----------------------------------------------------------------------------
// UserID (Domain Reference)
// -----------------------------------------------------------------------------

type UserID uuid.UUID

func NewUserID(raw uuid.UUID) (UserID, error) {
	if raw == uuid.Nil {
		return UserID{}, ErrIDNil
	}
	return UserID(raw), nil
}

func NewUserIDPtr(raw *uuid.UUID) (*UserID, error) {
	if raw == nil || *raw == uuid.Nil {
		return nil, nil
	}
	id, err := NewUserID(*raw)
	if err != nil {
		return nil, err
	}
	return &id, nil
}

func ParseUserID(raw string) (UserID, error) {
	parsed, err := uuid.Parse(strings.TrimSpace(raw))
	if err != nil {
		return UserID{}, ErrIDInvalid
	}
	return NewUserID(parsed)
}

func (id UserID) UUID() uuid.UUID          { return uuid.UUID(id) }
func (id UserID) String() string           { return uuid.UUID(id).String() }
func (id UserID) IsValid() bool            { return uuid.UUID(id) != uuid.Nil }
func (id UserID) Equals(other UserID) bool { return id == other }

// NewUserIDs validates a slice of raw UUIDs for batch operations.
func NewUserIDs(raws []uuid.UUID) ([]UserID, error) {
	if len(raws) == 0 {
		return nil, nil
	}
	ids := make([]UserID, 0, len(raws))
	for _, raw := range raws {
		id, err := NewUserID(raw)
		if err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}

// -----------------------------------------------------------------------------
// ChannelID (External / Optional Domain Reference)
// -----------------------------------------------------------------------------

type ChannelID uuid.UUID

func NewChannelID(raw uuid.UUID) (ChannelID, error) {
	if raw == uuid.Nil {
		return ChannelID{}, ErrIDNil
	}
	return ChannelID(raw), nil
}

func NewChannelIDPtr(raw *uuid.UUID) (*ChannelID, error) {
	if raw == nil || *raw == uuid.Nil {
		return nil, nil
	}
	id, err := NewChannelID(*raw)
	if err != nil {
		return nil, err
	}
	return &id, nil
}

func ParseChannelID(raw string) (ChannelID, error) {
	parsed, err := uuid.Parse(strings.TrimSpace(raw))
	if err != nil {
		return ChannelID{}, ErrIDInvalid
	}
	return NewChannelID(parsed)
}

func (id ChannelID) UUID() uuid.UUID             { return uuid.UUID(id) }
func (id ChannelID) String() string              { return uuid.UUID(id).String() }
func (id ChannelID) IsValid() bool               { return uuid.UUID(id) != uuid.Nil }
func (id ChannelID) Equals(other ChannelID) bool { return id == other }

func (id *ChannelID) EqualsPtr(other *ChannelID) bool {
	if id == nil && other == nil {
		return true
	}
	if id == nil || other == nil {
		return false
	}
	return *id == *other
}
