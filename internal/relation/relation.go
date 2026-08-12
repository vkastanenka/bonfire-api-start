package relation

import (
	"bytes"

	"bonfire-api/internal/fields"
)

type Relation struct {
	user1ID   fields.ID
	user2ID   fields.ID
	actorID   fields.ID
	channelID fields.ID
	relType   Type
	createdAt fields.Timestamp
	updatedAt fields.Timestamp
}

// -----------------------------------------------------------------------------
// Getters
// -----------------------------------------------------------------------------

func (r *Relation) User1ID() fields.ID          { return r.user1ID }
func (r *Relation) User2ID() fields.ID          { return r.user2ID }
func (r *Relation) ActorID() fields.ID          { return r.actorID }
func (r *Relation) ChannelID() fields.ID        { return r.channelID }
func (r *Relation) Type() Type                  { return r.relType }
func (r *Relation) CreatedAt() fields.Timestamp { return r.createdAt }
func (r *Relation) UpdatedAt() fields.Timestamp { return r.updatedAt }

// ============================================================================
// Meta
// ============================================================================

func (r *Relation) IsPending() bool { return r.relType == TypePending }
func (r *Relation) IsFriends() bool { return r.relType == TypeFriends }
func (r *Relation) IsBlocked() bool { return r.relType == TypeBlocked }

func (r *Relation) IsPendingActor(userID fields.ID) bool {
	return r.IsPending() && r.actorID.Equals(userID)
}

func (r *Relation) IsFriendsActor(userID fields.ID) bool {
	return r.IsFriends() && r.actorID.Equals(userID)
}

func (r *Relation) IsBlockedActor(userID fields.ID) bool {
	return r.IsBlocked() && r.actorID.Equals(userID)
}

func (r *Relation) IsParticipant(userID fields.ID) bool {
	return userID.Equals(r.user1ID) || userID.Equals(r.user2ID)
}

func (r *Relation) PeerID(userID fields.ID) fields.ID {
	if r.user1ID.Equals(userID) {
		return r.user2ID
	}
	return r.user1ID
}

// -----------------------------------------------------------------------------
// Mappers
// -----------------------------------------------------------------------------

func New(
	user1ID,
	user2ID,
	actorID,
	channelID fields.ID,
	relType Type,
	createdAt,
	updatedAt fields.Timestamp,
) *Relation {
	return &Relation{
		user1ID:   user1ID,
		user2ID:   user2ID,
		actorID:   actorID,
		channelID: channelID,
		relType:   TypePending,
		createdAt: createdAt,
		updatedAt: updatedAt,
	}
}

func Reconstitute(
	user1ID, user2ID, actorID, channelID fields.ID,
	relType Type,
	createdAt, updatedAt fields.Timestamp,
) *Relation {
	return &Relation{
		user1ID:   user1ID,
		user2ID:   user2ID,
		actorID:   actorID,
		channelID: channelID,
		relType:   relType,
		createdAt: createdAt,
		updatedAt: updatedAt,
	}
}

// -----------------------------------------------------------------------------
// Domain Mutations
// -----------------------------------------------------------------------------

func (r *Relation) Accept(actorID fields.ID, channelID fields.ID, now fields.Timestamp) {
	r.relType = TypeFriends
	r.actorID = actorID
	r.channelID = channelID
	r.touch(now)
}

func (r *Relation) Block(actorID fields.ID, now fields.Timestamp) {
	r.relType = TypeBlocked
	r.actorID = actorID
	r.touch(now)
}

func (r *Relation) touch(at fields.Timestamp) {
	r.updatedAt = at
}

// -----------------------------------------------------------------------------
// Helpers
// -----------------------------------------------------------------------------

func SortUserIDs(u1, u2 fields.ID) (fields.ID, fields.ID) {
	b1 := u1.UUID()
	b2 := u2.UUID()
	if bytes.Compare(b1[:], b2[:]) < 0 {
		return u1, u2
	}
	return u2, u1
}
