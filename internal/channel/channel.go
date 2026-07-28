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

func (c *Channel) ID() uuid.UUID             { return c.id }
func (c *Channel) Type() Type                { return c.channelType }
func (c *Channel) OwnerID() *uuid.UUID       { return c.ownerID }
func (c *Channel) Name() *Name               { return c.name }
func (c *Channel) IconURL() *IconURL         { return c.iconURL }
func (c *Channel) LastMessageID() *uuid.UUID { return c.lastMessageID }
func (c *Channel) CreatedAt() time.Time      { return c.createdAt }
func (c *Channel) UpdatedAt() time.Time      { return c.updatedAt }

func New(chType Type, ownerID *uuid.UUID, name *Name, iconURL *IconURL) (*Channel, error) {
	if !chType.IsValid() {
		return nil, ErrInvalidType
	}

	if chType == TypeDirect {
		if ownerID != nil {
			return nil, ErrInvalidOwnerForType
		}
		if name != nil || iconURL != nil {
			return nil, ErrDirectChannelCannotHaveMetadata
		}
	}

	if chType == TypeGroup && ownerID == nil {
		return nil, ErrOwnerRequired
	}

	now := time.Now().UTC()

	return &Channel{
		id:          uuid.Must(uuid.NewV7()),
		channelType: chType,
		ownerID:     ownerID,
		name:        name,
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
) (*Channel, error) {
	var nameVO *Name
	if name != nil {
		vo, err := NewName(*name)
		if err != nil {
			return nil, err
		}
		nameVO = &vo
	}

	var iconVO *IconURL
	if iconURL != nil {
		vo, err := NewIconURL(*iconURL)
		if err != nil {
			return nil, err
		}
		iconVO = &vo
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

func (c *Channel) UpdateName(newName *Name) error {
	if c.channelType == TypeDirect && newName != nil {
		return ErrDirectChannelCannotHaveMetadata
	}

	c.name = newName
	c.updatedAt = time.Now().UTC()
	return nil
}

func (c *Channel) UpdateIcon(newIcon *IconURL) error {
	if c.channelType == TypeDirect && newIcon != nil {
		return ErrDirectChannelCannotHaveMetadata
	}

	c.iconURL = newIcon
	c.updatedAt = time.Now().UTC()
	return nil
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
