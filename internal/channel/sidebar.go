package channel

import (
	"time"

	"github.com/google/uuid"
)

// UserSidebarItem represents an aggregated read model combining a channel
// with a user's membership state for rendering sidebar UI lists.
type UserSidebarItem struct {
	Member  Member
	Channel Channel
}

// ReconstituteSidebarItem constructs a UserSidebarItem from flat persistence projections.
func ReconstituteSidebarItem(
	channelID, userID uuid.UUID,
	joinedAt time.Time,
	lastReadMessageID *uuid.UUID,
	mentionCount int32,
	chType Type,
	ownerID *uuid.UUID,
	name, iconURL *string,
	lastMessageID *uuid.UUID,
	createdAt, updatedAt time.Time,
) (*UserSidebarItem, error) {
	ch, err := Reconstitute(
		channelID,
		chType,
		ownerID,
		name,
		iconURL,
		lastMessageID,
		createdAt,
		updatedAt,
	)
	if err != nil {
		return nil, err
	}

	mem, err := ReconstituteMember(
		channelID,
		userID,
		joinedAt,
		lastReadMessageID,
		mentionCount,
	)
	if err != nil {
		return nil, err
	}

	return &UserSidebarItem{
		Member:  *mem,
		Channel: *ch,
	}, nil
}
