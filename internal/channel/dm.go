package channel

import (
	"errors"

	"github.com/google/uuid"
)

var (
	ErrSameUserNotAllowed = errors.New("a user cannot create a dm channel with themselves")
	ErrInvalidUser1ID     = errors.New("invalid user1 id")
	ErrInvalidUser2ID     = errors.New("invalid user2 id")
	ErrInvalidChannelID   = errors.New("invalid channel id")
)

// DM represents a 1-on-1 private messaging link binding two users to a underlying channel.
type DM struct {
	user1ID   uuid.UUID
	user2ID   uuid.UUID
	channelID uuid.UUID
}

// -----------------------------------------------------------------------------
// Getters
// -----------------------------------------------------------------------------

func (d *DM) User1ID() uuid.UUID   { return d.user1ID }
func (d *DM) User2ID() uuid.UUID   { return d.user2ID }
func (d *DM) ChannelID() uuid.UUID { return d.channelID }

// HasParticipant checks if a given user is one of the two members in this DM link.
func (d *DM) HasParticipant(userID uuid.UUID) bool {
	if userID == uuid.Nil {
		return false
	}
	return d.user1ID == userID || d.user2ID == userID
}

// PeerID returns the other user's ID given one participant's ID.
func (d *DM) PeerID(actorID uuid.UUID) (uuid.UUID, bool) {
	if actorID == d.user1ID {
		return d.user2ID, true
	}
	if actorID == d.user2ID {
		return d.user1ID, true
	}
	return uuid.Nil, false
}

// -----------------------------------------------------------------------------
// Constructors / Factory Methods
// -----------------------------------------------------------------------------

// New creates a fresh DM link domain entity.
// It automatically sorts user IDs so user1_id < user2_id, guaranteeing database constraint alignment.
func NewDM(rawUserA, rawUserB, channelID uuid.UUID) (*DM, error) {
	if rawUserA == uuid.Nil {
		return nil, ErrInvalidUser1ID
	}
	if rawUserB == uuid.Nil {
		return nil, ErrInvalidUser2ID
	}
	if rawUserA == rawUserB {
		return nil, ErrSameUserNotAllowed
	}
	if channelID == uuid.Nil {
		return nil, ErrInvalidChannelID
	}

	// Order users deterministically (user1_id < user2_id)
	user1, user2 := OrderUserIDs(rawUserA, rawUserB)

	return &DM{
		user1ID:   user1,
		user2ID:   user2,
		channelID: channelID,
	}, nil
}

// Reconstitute restores an existing DM entity directly from persistence.
func ReconstituteDM(user1ID, user2ID, channelID uuid.UUID) (*DM, error) {
	if user1ID == uuid.Nil {
		return nil, ErrInvalidUser1ID
	}
	if user2ID == uuid.Nil {
		return nil, ErrInvalidUser2ID
	}
	if user1ID == user2ID {
		return nil, ErrSameUserNotAllowed
	}
	if channelID == uuid.Nil {
		return nil, ErrInvalidChannelID
	}

	return &DM{
		user1ID:   user1ID,
		user2ID:   user2ID,
		channelID: channelID,
	}, nil
}

// -----------------------------------------------------------------------------
// Helpers
// -----------------------------------------------------------------------------

// OrderUserIDs ensures the smaller UUID is returned first (lexicographical/bytes comparison).
func OrderUserIDs(a, b uuid.UUID) (user1, user2 uuid.UUID) {
	if a.String() < b.String() {
		return a, b
	}
	return b, a
}
