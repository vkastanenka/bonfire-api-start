package relation

import (
	"cmp"
	"slices"
	"strings"

	"bonfire-api/internal/fields"
	"bonfire-api/internal/user"

	"github.com/google/uuid"
)

const maxPeerTypeLimit int = 1000

type Relation struct {
	user1ID   fields.ID
	user2ID   fields.ID
	actorID   fields.ID
	channelID fields.ID
	relType   Type
	createdAt fields.Timestamp
	updatedAt fields.Timestamp
}

func Reconstitute(
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
		relType:   relType,
		createdAt: createdAt,
		updatedAt: updatedAt,
	}
}

func NewPending(
	user1ID,
	user2ID,
	actorID,
	channelID fields.ID,
	now fields.Timestamp,
) *Relation {
	return Reconstitute(
		user1ID,
		user2ID,
		actorID,
		channelID,
		NewTypePending(),
		now,
		now,
	)
}

func NewFriends(
	user1ID,
	user2ID,
	actorID,
	channelID fields.ID,
	now fields.Timestamp,
) *Relation {
	return Reconstitute(
		user1ID,
		user2ID,
		actorID,
		channelID,
		NewTypeFriends(),
		now,
		now,
	)
}

func NewBlocked(
	user1ID,
	user2ID,
	actorID fields.ID,
	now fields.Timestamp,
) *Relation {
	return Reconstitute(
		user1ID,
		user2ID,
		actorID,
		fields.ID{},
		NewTypeBlocked(),
		now,
		now,
	)
}

func (r *Relation) User1ID() fields.ID          { return r.user1ID }
func (r *Relation) User2ID() fields.ID          { return r.user2ID }
func (r *Relation) ActorID() fields.ID          { return r.actorID }
func (r *Relation) ChannelID() fields.ID        { return r.channelID }
func (r *Relation) Type() Type                  { return r.relType }
func (r *Relation) CreatedAt() fields.Timestamp { return r.createdAt }
func (r *Relation) UpdatedAt() fields.Timestamp { return r.updatedAt }

func (r *Relation) IsPending() bool { return r.relType.IsPending() }
func (r *Relation) IsFriends() bool { return r.relType.IsFriends() }
func (r *Relation) IsBlocked() bool { return r.relType.IsBlocked() }

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

func (r *Relation) PeerIDs(userID fields.ID) []fields.ID {
	return []fields.ID{r.PeerID(userID)}
}

func (r *Relation) Accept(actorID fields.ID, channelID fields.ID, now fields.Timestamp) {
	r.relType = NewTypeFriends()
	r.actorID = actorID
	r.channelID = channelID
	r.touch(now)
}

func (r *Relation) Block(actorID fields.ID, now fields.Timestamp) {
	r.relType = NewTypeBlocked()
	r.actorID = actorID
	r.touch(now)
}

func (r *Relation) touch(at fields.Timestamp) {
	r.updatedAt = at
}

func getPeerDisplayName(p Peer) string {
	if name := p.DisplayName.String(); name != "" {
		return name
	}
	return p.Username.String()
}

func sortPeers(peers []Peer) {
	slices.SortFunc(peers, func(a, b Peer) int {
		nameA := strings.ToLower(getPeerDisplayName(a))
		nameB := strings.ToLower(getPeerDisplayName(b))

		if c := cmp.Compare(nameA, nameB); c != 0 {
			return c
		}

		return a.ID.Compare(b.ID)
	})
}

func SortFriendIDs(friendIDs []fields.ID, users map[fields.ID]*user.User) {
	slices.SortFunc(friendIDs, func(aID, bID fields.ID) int {
		uA := users[aID]
		uB := users[bID]

		// Extract lowercased display names or usernames
		var nameA, nameB string
		if uA != nil {
			if dn := uA.DisplayName().String(); dn != "" {
				nameA = strings.ToLower(dn)
			} else {
				nameA = strings.ToLower(uA.Username().String())
			}
		}
		if uB != nil {
			if dn := uB.DisplayName().String(); dn != "" {
				nameB = strings.ToLower(dn)
			} else {
				nameB = strings.ToLower(uB.Username().String())
			}
		}

		// Primary sort: Display Name / Username
		if c := cmp.Compare(nameA, nameB); c != 0 {
			return c
		}

		// Secondary tie-breaker: ID comparison
		return aID.Compare(bID)
	})
}

func validateIDs(rawActorID, rawPeerID uuid.UUID) (actorID, peerID, u1, u2 fields.ID, err error) {
	if actorID, err = fields.ParseRequiredID("actor_id", rawActorID); err != nil {
		return fields.ID{}, fields.ID{}, fields.ID{}, fields.ID{}, err
	}
	if peerID, err = fields.ParseRequiredID("channel_id", rawPeerID); err != nil {
		return fields.ID{}, fields.ID{}, fields.ID{}, fields.ID{}, err
	}
	if actorID.Equals(peerID) {
		return fields.ID{}, fields.ID{}, fields.ID{}, fields.ID{}, ErrPeerIDInvalid()
	}
	u1, u2 = fields.SortIDs(actorID, peerID)
	return actorID, peerID, u1, u2, nil
}

func validateBlockedActor(actorID fields.ID, rel *Relation) error {
	if rel.IsBlockedActor(actorID) {
		return ErrBlockedActor()
	}
	return nil
}

func validateAccept(actorID fields.ID, rel *Relation) error {
	if rel.Type().IsPending() && !rel.ActorID().Equals(actorID) {
		return ErrNotPending()
	}
	return nil
}
