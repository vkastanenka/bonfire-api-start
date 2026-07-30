package relationship

import (
	"bytes"
	"errors"
	"time"

	"bonfire-api/internal/user"

	"github.com/google/uuid"
)

var (
	ErrSelfRelationship               = errors.New("cannot create relationship with oneself")
	ErrInvalidActor                   = errors.New("actor must be one of the relationship participants")
	ErrRelationshipBlocked            = errors.New("relationship is blocked")
	ErrCannotAccept                   = errors.New("only the target user can accept a pending request")
	ErrChannelRequiredForFriends      = errors.New("a friend relationship requires a valid channel_id")
	ErrChannelNotAllowedForNonFriends = errors.New("only friend relationships can retain a channel_id")
)

type Relationship struct {
	user1ID   UserID
	user2ID   UserID
	actorID   UserID
	channelID *ChannelID
	variant   Variant
	createdAt time.Time
	updatedAt time.Time
}

// -----------------------------------------------------------------------------
// Getters
// -----------------------------------------------------------------------------

func (r *Relationship) User1ID() UserID       { return r.user1ID }
func (r *Relationship) User2ID() UserID       { return r.user2ID }
func (r *Relationship) ActorID() UserID       { return r.actorID }
func (r *Relationship) ChannelID() *ChannelID { return r.channelID }
func (r *Relationship) Variant() Variant      { return r.variant }
func (r *Relationship) CreatedAt() time.Time  { return r.createdAt }
func (r *Relationship) UpdatedAt() time.Time  { return r.updatedAt }

// -----------------------------------------------------------------------------
// Constructors & Factory Methods
// -----------------------------------------------------------------------------

// New creates a fresh Pending Relationship entity.
func New(actorID, targetID UserID) (*Relationship, error) {
	if !actorID.IsValid() || !targetID.IsValid() {
		return nil, ErrIDNil
	}
	if actorID.Equals(targetID) {
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

// Reconstitute restores an existing Relationship aggregate from persistence.
func Reconstitute(
	rawUser1ID, rawUser2ID, rawActorID uuid.UUID,
	rawChannelID *uuid.UUID,
	rawVariant uint8,
	createdAt, updatedAt time.Time,
) (*Relationship, error) {
	u1, err := NewUserID(rawUser1ID)
	if err != nil {
		return nil, err
	}

	u2, err := NewUserID(rawUser2ID)
	if err != nil {
		return nil, err
	}

	actor, err := NewUserID(rawActorID)
	if err != nil {
		return nil, err
	}

	chID, err := NewChannelIDPtr(rawChannelID)
	if err != nil {
		return nil, err
	}

	variant := Variant(rawVariant)
	if !variant.IsValid() {
		return nil, ErrInvalidVariant
	}

	// Enforce DB invariants at domain boundary
	if variant == VariantFriends && chID == nil {
		return nil, ErrChannelRequiredForFriends
	}
	if variant != VariantFriends && chID != nil {
		return nil, ErrChannelNotAllowedForNonFriends
	}

	return &Relationship{
		user1ID:   u1,
		user2ID:   u2,
		actorID:   actor,
		channelID: chID,
		variant:   variant,
		createdAt: createdAt,
		updatedAt: updatedAt,
	}, nil
}

// -----------------------------------------------------------------------------
// Domain Mutations
// -----------------------------------------------------------------------------

// Accept transitions a pending relationship to friends and attaches its Direct Message Channel.
func (r *Relationship) Accept(actorID UserID, channelID ChannelID) error {
	if !r.IsParticipant(actorID) {
		return ErrInvalidActor
	}
	if r.variant != VariantPending {
		return errors.New("only pending requests can be accepted")
	}
	if r.actorID.Equals(actorID) {
		return ErrCannotAccept
	}
	if !channelID.IsValid() {
		return ErrIDNil
	}

	r.variant = VariantFriends
	r.actorID = actorID
	r.channelID = &channelID
	r.touch()
	return nil
}

// Block transitions any relationship state to blocked and strips channel linkage.
func (r *Relationship) Block(actorID UserID) error {
	if !r.IsParticipant(actorID) {
		return ErrInvalidActor
	}

	r.variant = VariantBlocked
	r.actorID = actorID
	r.channelID = nil
	r.touch()
	return nil
}

// -----------------------------------------------------------------------------
// Helper Methods
// -----------------------------------------------------------------------------

func (r *Relationship) IsParticipant(userID UserID) bool {
	return userID.Equals(r.user1ID) || userID.Equals(r.user2ID)
}

func (r *Relationship) IsPending() bool { return r.variant == VariantPending }
func (r *Relationship) IsFriends() bool { return r.variant == VariantFriends }
func (r *Relationship) IsBlocked() bool { return r.variant == VariantBlocked }

func (r *Relationship) GetPeerID(userID UserID) UserID {
	if userID.Equals(r.user1ID) {
		return r.user2ID
	}
	return r.user1ID
}

func (r *Relationship) touch() {
	r.updatedAt = time.Now().UTC()
}

func sortUserIDs(u1, u2 UserID) (UserID, UserID) {
	b1 := u1.UUID()
	b2 := u2.UUID()
	if bytes.Compare(b1[:], b2[:]) < 0 {
		return u1, u2
	}
	return u2, u1
}

// -----------------------------------------------------------------------------
// Perspective Aggregate
// -----------------------------------------------------------------------------

type Perspective struct {
	userID      UserID
	peerID      UserID
	variant     Variant
	actorID     UserID
	isInitiator bool
	channelID   *ChannelID
	createdAt   time.Time
	updatedAt   time.Time
	username    user.Username
	displayName user.ProfileDisplayName
	avatarURL   *string
}

func ReconstitutePerspective(
	rawUserID, rawPeerID, rawActorID uuid.UUID,
	rawChannelID *uuid.UUID,
	rawVariant uint8,
	isInitiator bool,
	createdAt, updatedAt time.Time,
	rawUsername string,
	rawDisplayName string,
	avatarURL *string,
) (*Perspective, error) {
	uID, err := NewUserID(rawUserID)
	if err != nil {
		return nil, err
	}

	pID, err := NewUserID(rawPeerID)
	if err != nil {
		return nil, err
	}

	actID, err := NewUserID(rawActorID)
	if err != nil {
		return nil, err
	}

	chID, err := NewChannelIDPtr(rawChannelID)
	if err != nil {
		return nil, err
	}

	uname, err := user.NewUsername(rawUsername)
	if err != nil {
		return nil, err
	}

	dName, err := user.NewProfileDisplayName(rawDisplayName)
	if err != nil {
		return nil, err
	}

	variant := Variant(rawVariant)
	if !variant.IsValid() {
		return nil, ErrInvalidVariant
	}

	return &Perspective{
		userID:      uID,
		peerID:      pID,
		variant:     variant,
		actorID:     actID,
		isInitiator: isInitiator,
		channelID:   chID,
		createdAt:   createdAt,
		updatedAt:   updatedAt,
		username:    uname,
		displayName: dName,
		avatarURL:   avatarURL,
	}, nil
}

func (p *Perspective) UserID() UserID                       { return p.userID }
func (p *Perspective) PeerID() UserID                       { return p.peerID }
func (p *Perspective) Variant() Variant                     { return p.variant }
func (p *Perspective) ActorID() UserID                      { return p.actorID }
func (p *Perspective) IsInitiator() bool                    { return p.isInitiator }
func (p *Perspective) ChannelID() *ChannelID                { return p.channelID }
func (p *Perspective) CreatedAt() time.Time                 { return p.createdAt }
func (p *Perspective) UpdatedAt() time.Time                 { return p.updatedAt }
func (p *Perspective) Username() user.Username              { return p.username }
func (p *Perspective) DisplayName() user.ProfileDisplayName { return p.displayName }
func (p *Perspective) AvatarURL() *string                   { return p.avatarURL }
