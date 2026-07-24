package relationship

import (
	"bytes"
	"errors"
	"time"

	"bonfire-api/internal/presence"
	"bonfire-api/internal/user"

	"github.com/google/uuid"
)

var (
	ErrSelfRelationship    = errors.New("cannot create relationship with oneself")
	ErrInvalidActor        = errors.New("actor must be one of the relationship participants")
	ErrRelationshipBlocked = errors.New("relationship is blocked")
	ErrCannotAccept        = errors.New("only the target user can accept a pending request")
)

type Relationship struct {
	user1ID   uuid.UUID
	user2ID   uuid.UUID
	actorID   uuid.UUID
	variant   Variant
	createdAt time.Time
	updatedAt time.Time
}

func Request(actorID, targetID uuid.UUID) (*Relationship, error) {
	if actorID == uuid.Nil || targetID == uuid.Nil {
		return nil, errors.New("user IDs cannot be nil")
	}
	if actorID == targetID {
		return nil, ErrSelfRelationship
	}

	user1, user2 := sortUserIDs(actorID, targetID)
	now := time.Now().UTC()

	return &Relationship{
		user1ID:   user1,
		user2ID:   user2,
		actorID:   actorID,
		variant:   VariantPending,
		createdAt: now,
		updatedAt: now,
	}, nil
}

func Reconstitute(
	user1ID, user2ID, actorID uuid.UUID,
	variant Variant,
	createdAt, updatedAt time.Time,
) *Relationship {
	return &Relationship{
		user1ID:   user1ID,
		user2ID:   user2ID,
		actorID:   actorID,
		variant:   variant,
		createdAt: createdAt,
		updatedAt: updatedAt,
	}
}

func (r *Relationship) Accept(actorID uuid.UUID) error {
	if !r.IsParticipant(actorID) {
		return ErrInvalidActor
	}
	if r.variant != VariantPending {
		return errors.New("only pending requests can be accepted")
	}
	if r.actorID == actorID {
		return ErrCannotAccept
	}

	r.variant = VariantFriends
	r.actorID = actorID
	r.updatedAt = time.Now().UTC()
	return nil
}

func (r *Relationship) Block(actorID uuid.UUID) error {
	if !r.IsParticipant(actorID) {
		return ErrInvalidActor
	}

	r.variant = VariantBlocked
	r.actorID = actorID
	r.updatedAt = time.Now().UTC()
	return nil
}

func (r *Relationship) IsParticipant(userID uuid.UUID) bool {
	return userID == r.user1ID || userID == r.user2ID
}

func (r *Relationship) IsPending() bool { return r.variant == VariantPending }
func (r *Relationship) IsFriends() bool { return r.variant == VariantFriends }
func (r *Relationship) IsBlocked() bool { return r.variant == VariantBlocked }

func (r *Relationship) GetPeerID(userID uuid.UUID) uuid.UUID {
	if userID == r.user1ID {
		return r.user2ID
	}
	return r.user1ID
}

func (r *Relationship) User1ID() uuid.UUID   { return r.user1ID }
func (r *Relationship) User2ID() uuid.UUID   { return r.user2ID }
func (r *Relationship) ActorID() uuid.UUID   { return r.actorID }
func (r *Relationship) Variant() Variant     { return r.variant }
func (r *Relationship) CreatedAt() time.Time { return r.createdAt }
func (r *Relationship) UpdatedAt() time.Time { return r.updatedAt }

func sortUserIDs(u1, u2 uuid.UUID) (uuid.UUID, uuid.UUID) {
	if bytes.Compare(u1[:], u2[:]) < 0 {
		return u1, u2
	}
	return u2, u1
}

type Perspective struct {
	userID                uuid.UUID
	peerID                uuid.UUID
	variant               Variant
	actorID               uuid.UUID
	isInitiator           bool
	createdAt             time.Time
	updatedAt             time.Time
	username              user.Username
	displayName           *user.ProfileDisplayName
	avatarURL             *string
	userPreferredPresence presence.Presence
	channelID             *uuid.UUID
}

func ReconstitutePerspective(
	userID, peerID uuid.UUID,
	variant Variant,
	actorID uuid.UUID,
	isInitiator bool,
	createdAt, updatedAt time.Time,
	username user.Username,
	displayName *user.ProfileDisplayName,
	avatarURL *string,
	userPreferredPresence presence.Presence,
	channelID *uuid.UUID,
) *Perspective {
	return &Perspective{
		userID:                userID,
		peerID:                peerID,
		variant:               variant,
		actorID:               actorID,
		isInitiator:           isInitiator,
		createdAt:             createdAt,
		updatedAt:             updatedAt,
		username:              username,
		displayName:           displayName,
		avatarURL:             avatarURL,
		userPreferredPresence: userPreferredPresence,
		channelID:             channelID,
	}
}

func (p *Perspective) UserID() uuid.UUID                     { return p.userID }
func (p *Perspective) PeerID() uuid.UUID                     { return p.peerID }
func (p *Perspective) Variant() Variant                      { return p.variant }
func (p *Perspective) ActorID() uuid.UUID                    { return p.actorID }
func (p *Perspective) IsInitiator() bool                     { return p.isInitiator }
func (p *Perspective) CreatedAt() time.Time                  { return p.createdAt }
func (p *Perspective) UpdatedAt() time.Time                  { return p.updatedAt }
func (p *Perspective) Username() user.Username               { return p.username }
func (p *Perspective) DisplayName() *user.ProfileDisplayName { return p.displayName }
func (p *Perspective) AvatarURL() *string                    { return p.avatarURL }
func (p *Perspective) UserPreferredPresence() presence.Presence {
	return p.userPreferredPresence
}
func (p *Perspective) ChannelID() *uuid.UUID { return p.channelID }
