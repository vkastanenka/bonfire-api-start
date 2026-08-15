package channel

import (
	"bonfire-api/internal/fields"
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrDirectChannelCannotHaveMetadata = errors.New("direct channels cannot have a name or icon")
)

type Channel struct {
	id            fields.ID
	chType        Type
	name          Name
	iconURL       fields.URL
	lastMessageID fields.ID
	lastMessageAt fields.Timestamp
	createdAt     fields.Timestamp
	updatedAt     fields.Timestamp
}

// -----------------------------------------------------------------------------
// Getters
// -----------------------------------------------------------------------------

func (c *Channel) ID() fields.ID                   { return c.id }
func (c *Channel) Type() Type                      { return c.chType }
func (c *Channel) Name() Name                      { return c.name }
func (c *Channel) IconURL() fields.URL             { return c.iconURL }
func (c *Channel) LastMessageID() fields.ID        { return c.lastMessageID }
func (c *Channel) LastMessageAt() fields.Timestamp { return c.lastMessageAt }
func (c *Channel) CreatedAt() fields.Timestamp     { return c.createdAt }
func (c *Channel) UpdatedAt() fields.Timestamp     { return c.updatedAt }

// -----------------------------------------------------------------------------
// Constructors / Factory Methods
// -----------------------------------------------------------------------------

// New creates a fresh Channel domain entity.
func New(chType Type, name *Name, iconURL *IconURL) (*Channel, error) {
	if !chType.IsValid() {
		return nil, ErrInvalidType
	}

	switch chType {
	case TypeDirect:
		if name != nil || iconURL != nil {
			return nil, ErrDirectChannelCannotHaveMetadata
		}
	}

	now := time.Now().UTC()

	rawID := uuid.Must(uuid.NewV7())
	id, err := NewID(rawID)
	if err != nil {
		return nil, err
	}

	return &Channel{
		id:        id,
		chType:    chType,
		name:      name,
		iconURL:   iconURL,
		createdAt: now,
		updatedAt: now,
	}, nil
}

// Reconstitute restores an existing Channel aggregate from persistence.
func Reconstitute(
	rawID uuid.UUID,
	rawType int16,
	rawName *string,
	rawIconURL *string,
	rawLastMessageID *uuid.UUID,
	createdAt, updatedAt time.Time,
) (*Channel, error) {
	id, err := NewID(rawID)
	if err != nil {
		return nil, err
	}

	chType, err := ParseType(rawType)
	if err != nil {
		return nil, err
	}

	name, err := NewName(rawName)
	if err != nil {
		return nil, err
	}

	iconURL, err := NewIconURL(rawIconURL)
	if err != nil {
		return nil, err
	}

	switch chType {
	case TypeDirect:
		if name != nil || iconURL != nil {
			return nil, ErrDirectChannelCannotHaveMetadata
		}
	}

	msgID, err := NewMessageIDPtr(rawLastMessageID)
	if err != nil {
		return nil, err
	}

	return &Channel{
		id:            id,
		chType:        chType,
		name:          name,
		iconURL:       iconURL,
		lastMessageID: msgID,
		createdAt:     createdAt,
		updatedAt:     updatedAt,
	}, nil
}

// -----------------------------------------------------------------------------
// Domain Mutations
// -----------------------------------------------------------------------------

// SetName updates the channel's display name.
func (c *Channel) SetName(newName *Name) error {
	if c.chType == TypeDirect {
		return ErrDirectChannelCannotHaveMetadata
	}

	if c.name.Equals(newName) {
		return nil
	}

	c.name = newName
	c.touch()
	return nil
}

// SetIcon updates the channel's icon URL.
func (c *Channel) SetIcon(newIcon *IconURL) error {
	if c.chType == TypeDirect {
		return ErrDirectChannelCannotHaveMetadata
	}

	if c.iconURL.Equals(newIcon) {
		return nil
	}

	c.iconURL = newIcon
	c.touch()
	return nil
}

// SetLastMessage records the latest message posted to the channel.
func (c *Channel) SetLastMessage(messageID MessageID) error {
	if !messageID.IsValid() {
		return ErrIDNil
	}

	if c.lastMessageID != nil && c.lastMessageID.Equals(messageID) {
		return nil
	}

	msgID := messageID
	c.lastMessageID = &msgID
	c.touch()
	return nil
}

func (c *Channel) touch() {
	c.updatedAt = time.Now().UTC()
}

// UpdateMeta attempts to update name and/or icon URL, returning true if any state changed.
func (c *Channel) UpdateMeta(rawName, rawIconURL *string) (bool, error) {
	var updated bool

	if rawName != nil {
		nameVO, err := NewName(rawName)
		if err != nil {
			return false, err
		}
		err = c.SetName(nameVO)
		if err != nil {
			return false, err
		}
		updated = true
	}

	if rawIconURL != nil {
		iconVO, err := NewIconURL(rawIconURL)
		if err != nil {
			return false, err
		}
		err = c.SetIcon(iconVO)
		if err != nil {
			return false, err
		}
		updated = true
	}

	return updated, nil
}
