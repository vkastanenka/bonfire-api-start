package channel

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrInvalidOwnerForType = errors.New("direct channels cannot have an owner")
	ErrOwnerRequired       = errors.New("group channels require an owner")
)

type Channel struct {
	id            uuid.UUID
	channelType   Type
	ownerID       *uuid.UUID
	name          *string
	iconURL       *string
	lastMessageID *uuid.UUID
	createdAt     time.Time
	updatedAt     time.Time
}

func (c *Channel) ID() uuid.UUID             { return c.id }
func (c *Channel) Type() Type                { return c.channelType }
func (c *Channel) OwnerID() *uuid.UUID       { return c.ownerID }
func (c *Channel) Name() *string             { return c.name }
func (c *Channel) IconURL() *string          { return c.iconURL }
func (c *Channel) LastMessageID() *uuid.UUID { return c.lastMessageID }
func (c *Channel) CreatedAt() time.Time      { return c.createdAt }
func (c *Channel) UpdatedAt() time.Time      { return c.updatedAt }

// Domain Factory Method
func New(chType Type, ownerID *uuid.UUID, name *Name, iconURL *string) (*Channel, error) {
	if !chType.IsValid() {
		return nil, ErrInvalidType
	}

	if chType == TypeDirect && ownerID != nil {
		return nil, ErrInvalidOwnerForType
	}
	if chType == TypeGroup && ownerID == nil {
		return nil, ErrOwnerRequired
	}

	now := time.Now().UTC()
	var nameStr *string
	if name != nil {
		s := name.String()
		nameStr = &s
	}

	return &Channel{
		id:          uuid.Must(uuid.NewV7()),
		channelType: chType,
		ownerID:     ownerID,
		name:        nameStr,
		iconURL:     iconURL,
		createdAt:   now,
		updatedAt:   now,
	}, nil
}

func Reconstitute(
	id uuid.UUID,
	chType Type,
	ownerID *uuid.UUID,
	name *string,
	iconURL *string,
	lastMessageID *uuid.UUID,
	createdAt, updatedAt time.Time,
) *Channel {
	return &Channel{
		id:            id,
		channelType:   chType,
		ownerID:       ownerID,
		name:          name,
		iconURL:       iconURL,
		lastMessageID: lastMessageID,
		createdAt:     createdAt,
		updatedAt:     updatedAt,
	}
}

func (c *Channel) UpdateName(newName *Name) {
	now := time.Now().UTC()
	if newName == nil {
		c.name = nil
	} else {
		s := newName.String()
		c.name = &s
	}
	c.updatedAt = now
}

func (c *Channel) UpdateIcon(iconURL *string) {
	c.iconURL = iconURL
	c.updatedAt = time.Now().UTC()
}

func (c *Channel) TransferOwnership(newOwnerID uuid.UUID) error {
	if c.channelType == TypeDirect {
		return ErrInvalidOwnerForType
	}
	c.ownerID = &newOwnerID
	c.updatedAt = time.Now().UTC()
	return nil
}

func (c *Channel) SetLastMessage(messageID uuid.UUID) {
	c.lastMessageID = &messageID
	c.updatedAt = time.Now().UTC()
}
