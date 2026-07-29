package channel

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrInvalidOwnerForType             = errors.New("direct channels cannot have an owner")
	ErrOwnerRequired                   = errors.New("group channels require an owner")
	ErrDirectChannelCannotHaveMetadata = errors.New("direct channels cannot have a name or icon")
	ErrNotGroupOwner                   = errors.New("only the group owner can perform this action")
	ErrInvalidOwnerID                  = errors.New("invalid owner id")
)

type Channel struct {
	id            uuid.UUID
	channelType   Type
	ownerID       *uuid.UUID
	name          *Name
	iconURL       *IconURL
	lastMessageID *uuid.UUID
	createdAt     time.Time
	updatedAt     time.Time
}

// -----------------------------------------------------------------------------
// Getters
// -----------------------------------------------------------------------------

func (c *Channel) ID() uuid.UUID             { return c.id }
func (c *Channel) Type() Type                { return c.channelType }
func (c *Channel) OwnerID() *uuid.UUID       { return c.ownerID }
func (c *Channel) Name() *Name               { return c.name }
func (c *Channel) IconURL() *IconURL         { return c.iconURL }
func (c *Channel) LastMessageID() *uuid.UUID { return c.lastMessageID }
func (c *Channel) CreatedAt() time.Time      { return c.createdAt }
func (c *Channel) UpdatedAt() time.Time      { return c.updatedAt }

// IsOwner returns true if the actor is the designated owner of the channel.
func (c *Channel) IsOwner(actorID uuid.UUID) bool {
	if c.ownerID == nil || actorID == uuid.Nil {
		return false
	}
	return *c.ownerID == actorID
}

// -----------------------------------------------------------------------------
// Constructors / Factory Methods
// -----------------------------------------------------------------------------

// New creates a fresh Channel domain entity.
func New(chType Type, ownerID *ID, name *Name, iconURL *IconURL) (*Channel, error) {
	if !chType.IsValid() {
		return nil, ErrInvalidType
	}

	switch chType {
	case TypeDirect:
		if ownerID != nil {
			return nil, ErrInvalidOwnerForType
		}
		if name != nil || iconURL != nil {
			return nil, ErrDirectChannelCannotHaveMetadata
		}

	case TypeGroup:
		if ownerID == nil || !ownerID.IsValid() {
			return nil, ErrOwnerRequired
		}
	}

	now := time.Now().UTC()

	var ownerUUID *uuid.UUID
	if ownerID != nil {
		u := ownerID.UUID()
		ownerUUID = &u
	}

	return &Channel{
		id:          uuid.Must(uuid.NewV7()),
		channelType: chType,
		ownerID:     ownerUUID,
		name:        name,
		iconURL:     iconURL,
		createdAt:   now,
		updatedAt:   now,
	}, nil
}

// Reconstitute restores an existing Channel aggregate from persistence.
func Reconstitute(
	id uuid.UUID,
	chType Type,
	ownerID *uuid.UUID,
	name *string,
	iconURL *string,
	lastMessageID *uuid.UUID,
	createdAt, updatedAt time.Time,
) (*Channel, error) {
	var nameVO *Name
	if name != nil {
		vo, err := NewName(name)
		if err != nil {
			return nil, err
		}
		nameVO = vo
	}

	var iconVO *IconURL
	if iconURL != nil {
		vo, err := NewIconURL(iconURL)
		if err != nil {
			return nil, err
		}
		iconVO = vo
	}

	return &Channel{
		id:            id,
		channelType:   chType,
		ownerID:       ownerID,
		name:          nameVO,
		iconURL:       iconVO,
		lastMessageID: lastMessageID,
		createdAt:     createdAt,
		updatedAt:     updatedAt,
	}, nil
}

// -----------------------------------------------------------------------------
// Domain Mutations
// -----------------------------------------------------------------------------

// UpdateName updates the channel's display name.
func (c *Channel) UpdateName(newName *Name) error {
	if c.channelType == TypeDirect {
		return ErrDirectChannelCannotHaveMetadata
	}

	// Skip mutation if value object has not changed
	if c.name.Equals(newName) {
		return nil
	}

	c.name = newName
	c.touch()
	return nil
}

// UpdateIcon updates the channel's icon URL.
func (c *Channel) UpdateIcon(newIcon *IconURL) error {
	if c.channelType == TypeDirect {
		return ErrDirectChannelCannotHaveMetadata
	}

	// Skip mutation if value object has not changed
	if c.iconURL.Equals(newIcon) {
		return nil
	}

	c.iconURL = newIcon
	c.touch()
	return nil
}

// TransferOwnership reassigns the group channel owner.
func (c *Channel) TransferOwnership(newOwnerID uuid.UUID) error {
	if c.channelType == TypeDirect {
		return ErrInvalidOwnerForType
	}
	if newOwnerID == uuid.Nil {
		return ErrInvalidOwnerID
	}
	if c.IsOwner(newOwnerID) {
		return nil
	}

	c.ownerID = &newOwnerID
	c.touch()
	return nil
}

// SetLastMessage records the latest message posted to the channel.
func (c *Channel) SetLastMessage(messageID uuid.UUID) {
	if messageID == uuid.Nil {
		return
	}
	if c.lastMessageID != nil && *c.lastMessageID == messageID {
		return
	}

	c.lastMessageID = &messageID
	c.touch()
}

func (c *Channel) touch() {
	c.updatedAt = time.Now().UTC()
}
