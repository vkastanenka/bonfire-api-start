package channel

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"slices"
	"sort"
	"time"

	"bonfire-api/internal/db"
	"bonfire-api/internal/errs"
	"bonfire-api/internal/outbox"
	"bonfire-api/internal/pkg/ptr"
	"bonfire-api/internal/relationship"

	"github.com/google/uuid"
)

const MinMembers = 1
const MaxMembers = 10

var (
	ErrNotParticipant  = errors.New("user is not a participant of this channel")
	ErrCannotEditOther = errors.New("cannot edit or delete a message authored by someone else")
	ErrNotOwner        = errors.New("only the channel owner can perform this action")
)

type OutboxRepository interface {
	Publish(ctx context.Context, variant string, payload any) (*outbox.Event, error)
}

type RelationshipRepository interface {
	Get(ctx context.Context, user1ID, user2ID uuid.UUID) (*relationship.Relationship, error)
	HasBlockBetweenUserAndPeers(ctx context.Context, userID uuid.UUID, peerIDs []uuid.UUID) (bool, error)
}

type Repository interface {
	// ============================================================================
	// CHANNELS
	// ============================================================================
	Create(ctx context.Context, ch *Channel) (*Channel, error)
	CreateDM(ctx context.Context, dm *DM) (*DM, error)
	Delete(ctx context.Context, id uuid.UUID) error
	FindDM(ctx context.Context, user1ID uuid.UUID, user2ID uuid.UUID) (*Channel, error)
	Get(ctx context.Context, id uuid.UUID) (*Channel, error)
	GetForMember(ctx context.Context, channelID uuid.UUID, memberID uuid.UUID) (*Channel, error)
	HasMessagesAfter(ctx context.Context, channelID uuid.UUID, createdAt time.Time, id uuid.UUID) (bool, error)
	HasMessagesBefore(ctx context.Context, channelID uuid.UUID, createdAt time.Time, id uuid.UUID) (bool, error)
	ListByUser(ctx context.Context, userID uuid.UUID) ([]Channel, error)
	Update(ctx context.Context, ch *Channel) error
	UpdateLastMessage(ctx context.Context, channelID uuid.UUID, messageID *uuid.UUID, updatedAt time.Time) error

	// ============================================================================
	// CHANNEL MEMBERS
	// ============================================================================
	IsMember(ctx context.Context, channelID uuid.UUID, userID uuid.UUID) (bool, error)
	MemberAddBatch(ctx context.Context, members []*Member) error
	MemberGet(ctx context.Context, channelID uuid.UUID, userID uuid.UUID) (*Member, error)
	MemberGetUnreadCount(ctx context.Context, channelID uuid.UUID, userID uuid.UUID) (int32, error)
	MemberIncrementMentionCountBatch(ctx context.Context, channelID uuid.UUID, userIDs []uuid.UUID) error
	MemberListByChannel(ctx context.Context, channelID uuid.UUID) ([]*Member, error)
	MemberRemove(ctx context.Context, channelID uuid.UUID, userID uuid.UUID) error
	MemberUpdateReadState(ctx context.Context, channelID uuid.UUID, userID uuid.UUID, messageID *uuid.UUID, lastReadAt time.Time) error

	// ============================================================================
	// MESSAGES
	// ============================================================================
	MessageCreate(ctx context.Context, msg *Message) error
	MessageDelete(ctx context.Context, id uuid.UUID) error
	MessageGet(ctx context.Context, id uuid.UUID) (*Message, error)
	MessageGetFirstUnread(ctx context.Context, channelID uuid.UUID, userID uuid.UUID) (*Message, error)
	MessageGetLatest(ctx context.Context, channelID uuid.UUID) (*Message, error)
	MessageListByChannelAfter(ctx context.Context, channelID uuid.UUID, cursorCreatedAt *time.Time, cursorID *uuid.UUID, limit int32) ([]Message, error)
	MessageListByChannelAround(ctx context.Context, channelID uuid.UUID, cursorCreatedAt time.Time, cursorID uuid.UUID, halfLimit int32) ([]Message, error)
	MessageListByChannelBefore(ctx context.Context, channelID uuid.UUID, cursorCreatedAt *time.Time, cursorID *uuid.UUID, limit int32) ([]Message, error)
	MessageListPinnedByChannel(ctx context.Context, channelID uuid.UUID) ([]Message, error)
	MessageListReplies(ctx context.Context, replyToMessageID uuid.UUID) ([]Message, error)
	MessageSetPinned(ctx context.Context, id uuid.UUID, isPinned bool) error
	MessageUpdateContent(ctx context.Context, msg *Message) error

	// ============================================================================
	// MESSAGE REACTIONS
	// ============================================================================
	ReactionAdd(ctx context.Context, messageID uuid.UUID, userID uuid.UUID, emoji string) (*Reaction, error)
	ReactionListByMessage(ctx context.Context, messageID uuid.UUID) ([]Reaction, error)
	ReactionRemove(ctx context.Context, messageID uuid.UUID, userID uuid.UUID, emoji Emoji) error
	ReactionSummarizeByMessage(ctx context.Context, messageID uuid.UUID, currentUserID uuid.UUID) ([]ReactionSummary, error)
}

type PresenceStore interface {
}

type TypingStore interface {
	SetTyping(ctx context.Context, channelID, userID uuid.UUID) error
}

type Tx interface {
	ExecTx(ctx context.Context, fn func(txCtx context.Context) error) error
}

type Service struct {
	repo         Repository
	outbox       OutboxRepository
	relationship RelationshipRepository
	typingStore  TypingStore
	tx           Tx
}

func NewService(repo Repository, outbox OutboxRepository, relationship RelationshipRepository, typingStore TypingStore, tx Tx) *Service {
	return &Service{
		repo:         repo,
		outbox:       outbox,
		relationship: relationship,
		typingStore:  typingStore,
		tx:           tx,
	}
}

// type ServiceI interface {
// 	// --- Channel Management ---
// 	Create(ctx context.Context, rawCreatorID uuid.UUID, rawMemberIDs []uuid.UUID) (*Channel, error)
// 	Get(ctx context.Context, channelID uuid.UUID, actorID uuid.UUID) (*Channel, error)
// 	GetDM(ctx context.Context, actorID uuid.UUID, targetUserID uuid.UUID) (*Channel, error) // Added
// 	ListByUser(ctx context.Context, userID uuid.UUID) ([]Channel, error)
// 	Update(ctx context.Context, input UpdateChannelParams) (*Channel, error)
// 	Delete(ctx context.Context, channelID uuid.UUID, actorID uuid.UUID) error

// 	// --- Member Management ---
// 	AddMembers(ctx context.Context, channelID uuid.UUID, actorID uuid.UUID, memberIDs []uuid.UUID) error
// 	RemoveMember(ctx context.Context, channelID uuid.UUID, actorID uuid.UUID, targetUserID uuid.UUID) error
// 	ListMembers(ctx context.Context, channelID uuid.UUID, actorID uuid.UUID) ([]*Member, error)
// 	UpdateNotificationSettings(ctx context.Context, channelID, userID uuid.UUID, isMuted bool) error // Added

// 	// --- Messaging ---
// 	SendMessage(ctx context.Context, params SendMessageParams) (*Message, error) // Updated for attachments
// 	EditMessage(ctx context.Context, messageID uuid.UUID, authorID uuid.UUID, newRawContent string) (*Message, error)
// 	DeleteMessage(ctx context.Context, messageID uuid.UUID, actorID uuid.UUID) error
// 	ListMessages(ctx context.Context, params ListMessagesParams) (*ListMessagesResult, error)
// 	GetInitialChannelMessages(ctx context.Context, channelID uuid.UUID, userID uuid.UUID, limit int32) (*InitialMessagesResult, error)
// 	SearchMessages(ctx context.Context, channelID, actorID uuid.UUID, query string, limit, offset int32) ([]*Message, error) // Added

// 	// --- Pins & Reactions ---
// 	ToggleMessagePin(ctx context.Context, messageID uuid.UUID, userID uuid.UUID, isPinned bool) (*Message, error)
// 	ListPinnedMessages(ctx context.Context, channelID, actorID uuid.UUID) ([]*Message, error) // Added
// 	AddReaction(ctx context.Context, messageID uuid.UUID, userID uuid.UUID, rawEmoji string) error
// 	RemoveReaction(ctx context.Context, messageID uuid.UUID, userID uuid.UUID, rawEmoji string) error

// 	// --- Unreads & Ephemeral Signals ---
// 	GetFirstUnreadMessage(ctx context.Context, channelID uuid.UUID, userID uuid.UUID) (*Message, error)
// 	MarkAsRead(ctx context.Context, channelID uuid.UUID, userID uuid.UUID, messageID uuid.UUID) error
// 	UpdateMemberReadState(ctx context.Context, channelID uuid.UUID, userID uuid.UUID, messageID *uuid.UUID, readAt time.Time) error
// 	SendTypingSignal(ctx context.Context, channelID uuid.UUID, userID uuid.UUID) error
// }

// ============================================================================
// CHANNELS
// ============================================================================

// Create creates a new DM or Group channel and adds participants atomically.
// If a 2-person channel (DM) already exists between the users, it returns the existing channel.
func (s *Service) Create(ctx context.Context, rawCreatorID uuid.UUID, rawMemberIDs []uuid.UUID) (*Channel, error) {
	creatorID, err := NewID(rawCreatorID)
	if err != nil {
		return nil, errs.InvalidArgument("invalid creator ID").Wrap(err)
	}

	memberIDs, err := NewIDs(rawMemberIDs)
	if err != nil {
		return nil, errs.InvalidArgument("invalid member ID in list").Wrap(err)
	}

	// 1. Deduplicate creator + members into a set
	totalMembers := make(map[ID]struct{}, len(memberIDs)+1)
	totalMembers[creatorID] = struct{}{}
	for _, id := range memberIDs {
		totalMembers[id] = struct{}{}
	}

	// 2. Business Rule Guard: Member Limits
	if len(totalMembers) < MinMembers {
		return nil, errs.InvalidArgument(fmt.Sprintf("channels must have at least %d members", MinMembers))
	}
	if len(totalMembers) > MaxMembers {
		return nil, errs.InvalidArgument(fmt.Sprintf("channels cannot exceed %d members", MaxMembers))
	}

	// 3. Extract Peers
	peerUUIDs := make([]uuid.UUID, 0, len(totalMembers)-1)
	for id := range totalMembers {
		if !id.Equals(creatorID) {
			peerUUIDs = append(peerUUIDs, id.UUID())
		}
	}

	// 4. Privacy Guard: Check Block Relationships (Creator vs. All Peers)
	if len(peerUUIDs) > 0 {
		hasBlock, err := s.relationship.HasBlockBetweenUserAndPeers(ctx, creatorID.UUID(), peerUUIDs)
		if err != nil {
			return nil, err
		}
		if hasBlock {
			return nil, errs.InvalidArgument("cannot create channel with one or more specified users")
		}
	}

	// Build & sort all member UUIDs deterministically for deadlocking safety
	dedupedUUIDs := make([]uuid.UUID, 0, len(totalMembers))
	dedupedUUIDs = append(dedupedUUIDs, creatorID.UUID())
	dedupedUUIDs = append(dedupedUUIDs, peerUUIDs...)

	slices.SortFunc(dedupedUUIDs, func(a, b uuid.UUID) int {
		return bytes.Compare(a[:], b[:])
	})

	// 5. Fast Path for Direct Messages (2-member channels)
	isDM := len(totalMembers) == 2
	if isDM {
		existing, err := s.repo.FindDM(ctx, dedupedUUIDs[0], dedupedUUIDs[1])
		if err == nil {
			return existing, nil
		}
		if !errs.IsNotFound(err) {
			return nil, err
		}
	}

	// 6. Instantiate Domain Entities Outside the Transaction
	channelType := TypeGroup
	var creatorRef *ID = &creatorID
	if isDM {
		channelType = TypeDirect
		creatorRef = nil // DMs have no owner/creator reference
	}

	ch, err := New(channelType, creatorRef, nil, nil)
	if err != nil {
		return nil, errs.InvalidArgument(err.Error()).Wrap(err)
	}

	var dm *DM
	if isDM {
		dm, err = NewDM(dedupedUUIDs[0], dedupedUUIDs[1], ch.ID())
		if err != nil {
			return nil, errs.InvalidArgument("invalid dm properties").Wrap(err)
		}
	}

	memberBatch := make([]*Member, 0, len(dedupedUUIDs))
	for _, u := range dedupedUUIDs {
		m, err := NewMember(ch.ID(), u)
		if err != nil {
			return nil, errs.InvalidArgument("invalid member properties").Wrap(err)
		}
		memberBatch = append(memberBatch, m)
	}

	// 7. Atomic Persistence Block
	var newChannel *Channel

	err = s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		row, err := s.repo.Create(txCtx, ch)
		if err != nil {
			return err
		}

		if isDM {
			if _, err := s.repo.CreateDM(txCtx, dm); err != nil {
				return err
			}
		}

		if err := s.repo.MemberAddBatch(txCtx, memberBatch); err != nil {
			return err
		}

		newChannel = row
		return nil
	})

	if err != nil {
		// Fallback for Concurrent Creation Race Condition on DMs
		if isDM && db.IsAlreadyExistsError(err) {
			winningCh, findErr := s.repo.FindDM(ctx, dedupedUUIDs[0], dedupedUUIDs[1])
			if findErr != nil {
				return nil, findErr
			}
			return winningCh, nil
		}
		return nil, err
	}

	return newChannel, nil
}

// Get fetches channel details if the actor is a member.
func (s *Service) Get(ctx context.Context, rawChannelID, rawActorID uuid.UUID) (*Channel, error) {
	channelID, err := NewID(rawChannelID)
	if err != nil {
		return nil, errs.InvalidArgument("invalid channel ID").Wrap(err)
	}

	actorID, err := NewID(rawActorID)
	if err != nil {
		return nil, errs.InvalidArgument("invalid actor ID").Wrap(err)
	}

	ch, err := s.repo.GetForMember(ctx, channelID.UUID(), actorID.UUID())
	if err != nil {
		return nil, err
	}

	return ch, nil
}

// ListByUser retrieves all channels that the specified user is a participant of.
func (s *Service) ListByUser(ctx context.Context, rawUserID uuid.UUID) ([]Channel, error) {
	userID, err := NewID(rawUserID)
	if err != nil {
		return nil, errs.InvalidArgument("invalid user ID").Wrap(err)
	}

	channels, err := s.repo.ListByUser(ctx, userID.UUID())
	if err != nil {
		return nil, err
	}

	return channels, nil
}

// type SidebarChannel struct {
// 	ChannelID         uuid.UUID
// 	UserID            uuid.UUID
// 	Type              int16
// 	Name              *string
// 	IconURL           *string
// 	OwnerID           *uuid.UUID
// 	LastReadMessageID *uuid.UUID
// 	LastMessageID     *uuid.UUID
// 	MentionCount      int32
// 	PeerUserID        *uuid.UUID
// 	Presence          presence.Presence // Hydrated dynamically from Redis
// 	UpdatedAt         time.Time
// }

// // ListByUser retrieves all sidebar channels for a user with real-time presence merged from cache.
// func (s *Service) ListByUser(ctx context.Context, rawUserID uuid.UUID) ([]SidebarChannel, error) {
// 	userID, err := NewID(rawUserID)
// 	if err != nil {
// 		return nil, errs.InvalidArgument("invalid user ID").Wrap(err)
// 	}

// 	rows, err := s.repo.ListByUser(ctx, userID.UUID())
// 	if err != nil {
// 		return nil, err
// 	}

// 	if len(rows) == 0 {
// 		return []SidebarChannel{}, nil
// 	}

// 	// 1. Collect unique peer user IDs from DMs
// 	peerIDs := make([]uuid.UUID, 0, len(rows))
// 	peerIDSet := make(map[uuid.UUID]struct{})

// 	for _, row := range rows {
// 		if row.PeerUserID.Valid {
// 			id := uuid.UUID(row.PeerUserID.Bytes)
// 			if _, exists := peerIDSet[id]; !exists {
// 				peerIDSet[id] = struct{}{}
// 				peerIDs = append(peerIDs, id)
// 			}
// 		}
// 	}

// 	// 2. Single MGET call to Redis for all DMs in the sidebar
// 	presenceMap, err := s.presenceStore.GetPresenceBulk(ctx, peerIDs)
// 	if err != nil {
// 		// Log error if needed; fallback to offline so sidebar rendering doesn't crash on cache failure
// 		presenceMap = make(map[uuid.UUID]presence.Presence)
// 	}

// 	// 3. Assemble response DTOs
// 	channels := make([]SidebarChannel, len(rows))
// 	for i, row := range rows {
// 		item := SidebarChannel{
// 			ChannelID:         uuid.UUID(row.ChannelID.Bytes),
// 			UserID:            uuid.UUID(row.UserID.Bytes),
// 			Type:              row.ChannelType,
// 			Name:              db.StringPtr(row.ChannelName),
// 			IconURL:           db.StringPtr(row.ChannelIconUrl),
// 			OwnerID:           db.UUIDPtrFromPG(row.ChannelOwnerID),
// 			LastReadMessageID: db.UUIDPtrFromPG(row.LastReadMessageID),
// 			LastMessageID:     db.UUIDPtrFromPG(row.ChannelLastMessageID),
// 			MentionCount:      row.MentionCount,
// 			UpdatedAt:         row.ChannelUpdatedAt.Time,
// 			Presence:          presence.PresenceOffline, // Default fallback
// 		}

// 		if row.PeerUserID.Valid {
// 			peerID := uuid.UUID(row.PeerUserID.Bytes)
// 			item.PeerUserID = &peerID
// 			if p, ok := presenceMap[peerID]; ok {
// 				item.Presence = p
// 			}
// 		}

// 		channels[i] = item
// 	}

// 	return channels, nil
// }

type UpdateChannelInput struct {
	ChannelID uuid.UUID
	ActorID   uuid.UUID
	Name      *string
	IconURL   *string
}

// Update encapsulates metadata changes requested by a user
func (s *Service) Update(ctx context.Context, input UpdateChannelInput) (*Channel, error) {
	ch, err := s.repo.Get(ctx, input.ChannelID)
	if err != nil {
		return nil, err
	}

	// 1. Guard against metadata modification on Direct Message channels
	if ch.Type() == TypeDirect {
		if input.Name != nil || input.IconURL != nil {
			return nil, ErrDirectChannelCannotHaveMetadata
		}
		return ch, nil
	}

	var updated bool

	// 2. Update Name (with true value equality check)
	if input.Name != nil {
		nameVO, err := NewName(input.Name)
		if err != nil {
			return nil, err
		}

		if !ch.Name().Equals(nameVO) {
			if err := ch.UpdateName(nameVO); err != nil {
				return nil, err
			}
			updated = true
		}
	}

	// 3. Update Icon URL (with true value equality check)
	if input.IconURL != nil {
		iconVO, err := NewIconURL(input.IconURL)
		if err != nil {
			return nil, err
		}

		if !ch.IconURL().Equals(iconVO) {
			if err := ch.UpdateIcon(iconVO); err != nil {
				return nil, err
			}
			updated = true
		}
	}

	// 4. Early return if state did not change
	if !updated {
		return ch, nil
	}

	// 5. Persist entity changes and emit Outbox event in a single atomic transaction
	err = s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		if err := s.repo.Update(txCtx, ch); err != nil {
			return err
		}

		payload := ChannelUpdatedPayload{
			ChannelID: ch.ID(),
			ActorID:   input.ActorID,
			Name:      db.ToStringPtr(ch.Name()),
			IconURL:   db.ToStringPtr(ch.IconURL()),
			UpdatedAt: ch.UpdatedAt(),
		}

		_, err := s.outbox.Publish(txCtx, "channel.updated", payload)
		return err
	})
	if err != nil {
		return nil, err
	}

	return ch, nil
}

// Delete permanently or soft deletes a channel aggregate.
// Note: Direct Messages cannot be deleted; users should hide/close DMs instead.
func (s *Service) Delete(ctx context.Context, rawChannelID, rawActorID uuid.UUID) error {
	channelID, err := NewID(rawChannelID)
	if err != nil {
		return errs.InvalidArgument("invalid channel ID").Wrap(err)
	}

	actorID, err := NewID(rawActorID)
	if err != nil {
		return errs.InvalidArgument("invalid actor ID").Wrap(err)
	}

	return s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		ch, err := s.repo.Get(txCtx, channelID.UUID())
		if err != nil {
			return err
		}

		if ch.Type() == TypeDirect {
			return errs.FailedPrecondition("direct messages cannot be deleted")
		}

		if !ch.IsOwner(actorID.UUID()) {
			return errs.PermissionDenied("only the owner can delete this channel").Wrap(ErrNotGroupOwner)
		}

		if err := s.repo.Delete(txCtx, channelID.UUID()); err != nil {
			return err
		}

		_, err = s.outbox.Publish(txCtx, EventChannelDeleted, DeletedPayload{
			ChannelID: channelID.UUID(),
			ActorID:   actorID.UUID(),
		})
		if err != nil {
			return err
		}

		return nil
	})
}

// ============================================================================
// MEMBERS
// ============================================================================

// AddMembers adds a batch of users to an existing group channel or upgrades a DM to a group channel.
func (s *Service) AddMembers(ctx context.Context, channelID uuid.UUID, actorID uuid.UUID, memberIDs []uuid.UUID) error {
	if channelID == uuid.Nil {
		return errs.InvalidArgument("channel ID cannot be nil")
	}
	if actorID == uuid.Nil {
		return errs.InvalidArgument("actor ID cannot be nil")
	}
	if len(memberIDs) == 0 {
		return errs.InvalidArgument("member IDs cannot be empty")
	}

	// Verify actor is in the channel before proceeding
	isMember, err := s.repo.IsMember(ctx, channelID, actorID)
	if err != nil {
		return err
	}
	if !isMember {
		return errs.PermissionDenied("you must be a member of this channel to add others").Wrap(ErrNotParticipant)
	}

	return s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		ch, err := s.repo.Get(txCtx, channelID)
		if err != nil {
			return err
		}

		// 1. IF DIRECT MESSAGE: Upgrade DM to a new Group Channel
		if ch.Type() == TypeDirect {
			existingMembers, err := s.repo.MemberListByChannel(txCtx, channelID)
			if err != nil {
				return err
			}

			// Extract UUIDs from existing members
			allMemberIDs := make([]uuid.UUID, 0, len(existingMembers)+len(memberIDs))
			for _, m := range existingMembers {
				allMemberIDs = append(allMemberIDs, m.UserID())
			}

			// Append new invitee IDs
			allMemberIDs = append(allMemberIDs, memberIDs...)

			// Create the new group channel (s.Create validates MaxMembers internally)
			_, err = s.Create(txCtx, actorID, allMemberIDs)
			if err != nil {
				return err
			}
			return nil
		}

		// 2. IF GROUP CHANNEL: Enforce Max Member Limit before insertion
		existingMembers, err := s.repo.MemberListByChannel(txCtx, channelID)
		if err != nil {
			return err
		}

		// Map existing members for fast O(1) deduplication lookup
		existingMap := make(map[uuid.UUID]struct{}, len(existingMembers))
		for _, m := range existingMembers {
			existingMap[m.UserID()] = struct{}{}
		}

		// Deduplicate incoming memberIDs and filter out existing members
		newMembersToInsert := make([]*Member, 0, len(memberIDs))
		seenIncoming := make(map[uuid.UUID]struct{}, len(memberIDs))

		for _, mID := range memberIDs {
			if mID == uuid.Nil {
				continue
			}
			// Skip if already in channel or duplicate in this request
			if _, exists := existingMap[mID]; exists {
				continue
			}
			if _, seen := seenIncoming[mID]; seen {
				continue
			}

			seenIncoming[mID] = struct{}{}

			member, err := NewMember(channelID, mID)
			if err != nil {
				return errs.InvalidArgument("invalid member id").Wrap(err)
			}
			newMembersToInsert = append(newMembersToInsert, member)
		}

		if len(newMembersToInsert) == 0 {
			return errs.InvalidArgument("all provided users are already members of this channel")
		}

		// Check capacity rule: Existing + New Unique Members <= MaxMembers
		totalProjectedMembers := len(existingMembers) + len(newMembersToInsert)
		if totalProjectedMembers > MaxMembers {
			return errs.InvalidArgument(
				fmt.Sprintf("group channels cannot exceed %d members (current: %d, adding: %d)",
					MaxMembers, len(existingMembers), len(newMembersToInsert)),
			)
		}

		// Persist new members in batch
		if err := s.repo.MemberAddBatch(txCtx, newMembersToInsert); err != nil {
			return err
		}

		// Extract IDs actually added for event payload
		addedIDs := make([]uuid.UUID, 0, len(newMembersToInsert))
		for _, m := range newMembersToInsert {
			addedIDs = append(addedIDs, m.UserID())
		}

		// _, err = s.outbox.Publish(txCtx, "channel.members_added", ChannelMembersAddedPayload{
		// 	ChannelID: channelID,
		// 	ActorID:   actorID,
		// 	MemberIDs: addedIDs,
		// })
		// if err != nil {
		// 	return err
		// }

		return nil
	})
}

// ListMembers returns all member records for a channel after confirming the requesting user is a participant.
func (s *Service) ListMembers(ctx context.Context, channelID, actorID uuid.UUID) ([]*Member, error) {
	if channelID == uuid.Nil || actorID == uuid.Nil {
		return nil, errs.InvalidArgument("channel ID and actor ID cannot be nil")
	}

	// 1. Authorization Guard: Ensure the requesting user belongs to the channel
	ok, err := s.repo.IsMember(ctx, channelID, actorID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, errs.PermissionDenied("you are not a member of this channel").Wrap(ErrNotParticipant)
	}

	// 2. Fetch member roster
	members, err := s.repo.MemberListByChannel(ctx, channelID)
	if err != nil {
		return nil, err
	}

	return members, nil
}

// RemoveMember removes a user from a channel, or handles leaving/channel cleanup if they are the last member.
func (s *Service) RemoveMember(ctx context.Context, channelID uuid.UUID, actorID uuid.UUID, targetUserID uuid.UUID) error {
	if channelID == uuid.Nil || actorID == uuid.Nil || targetUserID == uuid.Nil {
		return errs.InvalidArgument("channel ID, actor ID, and target user ID cannot be nil")
	}

	return s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		ch, err := s.repo.Get(txCtx, channelID)
		if err != nil {
			return err
		}

		// 1. Direct Messages cannot have members removed.
		// In 1-on-1 DMs, "leaving" should trigger a channel close/delete for the user rather than member removal.
		if ch.Type() == TypeDirect {
			return errs.InvalidArgument("cannot remove members from a direct message channel; delete or close the DM instead")
		}

		// 2. Authorization: Check that actor is in the channel
		actorIsMember, err := s.repo.IsMember(txCtx, channelID, actorID)
		if err != nil {
			return err
		}
		if !actorIsMember {
			return errs.PermissionDenied("you must be a member of this channel to perform this action").Wrap(ErrNotParticipant)
		}

		// 3. Verify target user is actually a member of the channel
		targetIsMember, err := s.repo.IsMember(txCtx, channelID, targetUserID)
		if err != nil {
			return err
		}
		if !targetIsMember {
			return errs.NotFound("target user is not a member of this channel")
		}

		// 4. Check current member roster to determine remaining count
		existingMembers, err := s.repo.MemberListByChannel(txCtx, channelID)
		if err != nil {
			return err
		}

		// 5. IF LAST MEMBER: Clean up the entire channel
		if len(existingMembers) <= 1 {
			if err := s.repo.Delete(txCtx, channelID); err != nil {
				return err
			}

			// _, err = s.outbox.Publish(txCtx, "channel.deleted", ChannelDeletedPayload{
			// 	ChannelID: channelID,
			// 	ActorID:   actorID,
			// })
			// return err
		}

		// 6. IF MULTIPLE MEMBERS: Remove target user from channel_members
		if err := s.repo.MemberRemove(txCtx, channelID, targetUserID); err != nil {
			return err
		}

		// // 7. Emit outbox event for gateway fanout
		// _, err = s.outbox.Publish(txCtx, "channel.member_removed", ChannelMemberRemovedPayload{
		// 	ChannelID:    channelID,
		// 	ActorID:      actorID,
		// 	TargetUserID: targetUserID,
		// })
		// if err != nil {
		// 	return err
		// }

		return nil
	})
}

// UpdateMemberReadState updates the user's read marker for a channel and emits an outbox event.
func (s *Service) UpdateMemberReadState(ctx context.Context, channelID, userID uuid.UUID, messageID *uuid.UUID, readAt time.Time) error {
	if channelID == uuid.Nil || userID == uuid.Nil {
		return errs.InvalidArgument("channel ID and user ID cannot be nil")
	}

	if readAt.IsZero() {
		readAt = time.Now().UTC()
	}

	// Wrap in transaction so DB update and outbox payload commit atomically
	return s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		// 1. Authorization Guard: Check channel membership
		isMember, err := s.repo.IsMember(txCtx, channelID, userID)
		if err != nil {
			return err
		}
		if !isMember {
			return errs.PermissionDenied("you are not a member of this channel").Wrap(ErrNotParticipant)
		}

		// 2 & 3. Persist via Repository (Monotonicity handled via CASE/GREATEST in SQL)
		if err := s.repo.MemberUpdateReadState(txCtx, channelID, userID, messageID, readAt); err != nil {
			return err
		}

		// 4. Side Effects: Publish to outbox for Gateway/WebSocket worker consumption
		_, err = s.outbox.Publish(txCtx, EventChannelReadUpdated, ChannelReadUpdatedPayload{
			ChannelID:         channelID,
			UserID:            userID,
			LastReadMessageID: messageID,
			LastReadAt:        readAt,
		})
		return err
	})
}

var (
	ErrCannotModifyDM       = errors.New("cannot modify properties of a direct message channel")
	ErrInvalidIconURL       = errors.New("icon URL must be between 3 and 2048 characters")
	ErrCannotTransferToSelf = errors.New("cannot transfer ownership to current owner")
)

// 	// ============================================================================
// 	// MESSAGING & PAGINATION
// 	// ============================================================================

// SendMessage validates user membership, constructs content value object, persists, and publishes the event.
func (s *Service) SendMessage(ctx context.Context, channelID, authorID uuid.UUID, rawContent string, replyToID *uuid.UUID) (*Message, error) {
	if channelID == uuid.Nil || authorID == uuid.Nil {
		return nil, errs.InvalidArgument("channel ID and author ID cannot be nil")
	}

	// 1. Authorization Guard: Fast-fail if not a participant
	ok, err := s.repo.IsMember(ctx, channelID, authorID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, errs.PermissionDenied("you are not a member of this channel").Wrap(ErrNotParticipant)
	}

	// 1b. Reply Target Validation: Ensure target exists and belongs to the same channel
	if replyToID != nil {
		parentMsg, err := s.repo.MessageGet(ctx, *replyToID)
		if err != nil {
			return nil, errs.NotFound("reply target message not found").Wrap(err)
		}
		if parentMsg.ChannelID() != channelID {
			return nil, errs.InvalidArgument("cannot reply to a message in a different channel")
		}
	}

	// 2. Domain Value Object & Entity Creation (Fail fast before opening tx)
	contentVO, err := NewContent(rawContent)
	if err != nil {
		return nil, errs.InvalidArgument(err.Error()).Wrap(err)
	}

	msg, err := NewMessage(channelID, &authorID, replyToID, contentVO)
	if err != nil {
		return nil, errs.InvalidArgument(err.Error()).Wrap(err)
	}

	// 3. Atomic Transaction: Persist Message + Write to Outbox Table
	err = s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		if err := s.repo.MessageCreate(txCtx, msg); err != nil {
			return err
		}

		if err := s.repo.UpdateLastMessage(txCtx, msg.ChannelID(), ptr.To(msg.ID()), msg.CreatedAt()); err != nil {
			return err
		}

		_, err := s.outbox.Publish(txCtx, EventMessageCreated, MessageCreatedPayload{
			MessageID: msg.ID(),
			ChannelID: msg.ChannelID(),
			AuthorID:  msg.AuthorID(),
			Content:   msg.Content().String(),
			ReplyToID: msg.ReplyToMessageID(),
			CreatedAt: msg.CreatedAt(),
		})
		return err
	})
	if err != nil {
		return nil, err
	}

	return msg, nil
}

// EditMessage allows the author to update message content.
func (s *Service) EditMessage(ctx context.Context, messageID, authorID uuid.UUID, newRawContent string) (*Message, error) {
	if messageID == uuid.Nil || authorID == uuid.Nil {
		return nil, errs.InvalidArgument("message ID and author ID cannot be nil")
	}

	contentVO, err := NewContent(newRawContent)
	if err != nil {
		return nil, errs.InvalidArgument(err.Error()).Wrap(err)
	}

	var updated *Message

	err = s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		msg, err := s.repo.MessageGet(txCtx, messageID)
		if err != nil {
			return err
		}

		if msg.AuthorID() == nil || *msg.AuthorID() != authorID {
			return errs.PermissionDenied("cannot edit messages sent by another user").Wrap(ErrCannotEditOther)
		}

		msg.EditContent(contentVO)

		if err := s.repo.MessageUpdateContent(txCtx, msg); err != nil {
			return err
		}

		_, err = s.outbox.Publish(txCtx, EventMessageUpdated, MessageUpdatedPayload{
			MessageID: msg.ID(),
			ChannelID: msg.ChannelID(),
			AuthorID:  msg.AuthorID(),
			Content:   msg.Content().String(),
			EditedAt:  msg.EditedAt(),
		})
		if err != nil {
			return err
		}

		updated = msg
		return nil
	})
	if err != nil {
		return nil, err
	}

	return updated, nil
}

// DeleteMessage deletes a message if requested by its author or an authorized user.
func (s *Service) DeleteMessage(ctx context.Context, messageID, actorID uuid.UUID) error {
	if messageID == uuid.Nil || actorID == uuid.Nil {
		return errs.InvalidArgument("message ID and actor ID cannot be nil")
	}

	return s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		msg, err := s.repo.MessageGet(txCtx, messageID)
		if err != nil {
			return err
		}

		// 1. Authorization Guard: Check if actor is author
		isAuthor := msg.AuthorID() != nil && *msg.AuthorID() == actorID
		if !isAuthor {
			return errs.PermissionDenied("cannot delete messages sent by another user").Wrap(ErrCannotEditOther)
		}

		// Fetch channel to check if this message is the channel's last_message_id
		ch, err := s.repo.Get(txCtx, msg.ChannelID())
		if err != nil {
			return err
		}

		// 2. Persist deletion in database
		if err := s.repo.MessageDelete(txCtx, messageID); err != nil {
			return err
		}

		// 3. If deleted message was the last_message_id, recalculate and update
		if ch.LastMessageID() != nil && *ch.LastMessageID() == messageID {
			latestMsg, err := s.repo.MessageGetLatest(txCtx, msg.ChannelID())
			if err != nil {
				return err
			}

			var nextMsgID *uuid.UUID
			var updatedAt = time.Now().UTC()

			if latestMsg != nil {
				id := latestMsg.ID()
				nextMsgID = &id
				updatedAt = latestMsg.CreatedAt()
			}

			if err := s.repo.UpdateLastMessage(txCtx, msg.ChannelID(), nextMsgID, updatedAt); err != nil {
				return err
			}
		}

		// 4. Publish outbox event
		now := time.Now().UTC()
		_, err = s.outbox.Publish(txCtx, EventMessageDeleted, MessageDeletedPayload{
			MessageID: msg.ID(),
			ChannelID: msg.ChannelID(),
			ActorID:   actorID,
			DeletedAt: now,
		})
		return err
	})
}

// ToggleMessagePin handles pinning or unpinning a message for channel members.
func (s *Service) ToggleMessagePin(ctx context.Context, messageID, userID uuid.UUID, isPinned bool) (*Message, error) {
	if messageID == uuid.Nil || userID == uuid.Nil {
		return nil, errs.InvalidArgument("message ID and user ID cannot be nil")
	}

	var updated *Message

	err := s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		msg, err := s.repo.MessageGet(txCtx, messageID)
		if err != nil {
			return err
		}

		// 1. Authorization Guard: Check channel membership for the pin action
		ok, err := s.repo.IsMember(txCtx, msg.ChannelID(), userID)
		if err != nil {
			return err
		}
		if !ok {
			return errs.PermissionDenied("you are not a member of this channel").Wrap(ErrNotParticipant)
		}

		// 2. Mutate state & persist
		msg.SetPinned(isPinned)

		if err := s.repo.MessageSetPinned(txCtx, msg.ID(), isPinned); err != nil {
			return err
		}

		// 3. Publish outbox event
		_, err = s.outbox.Publish(txCtx, EventMessagePinned, MessagePinnedPayload{
			MessageID: msg.ID(),
			ChannelID: msg.ChannelID(),
			IsPinned:  msg.IsPinned(),
		})
		if err != nil {
			return err
		}

		updated = msg
		return nil
	})
	if err != nil {
		return nil, err
	}

	return updated, nil
}

// TODO: aggregate reactions into message

// PaginationDirection defines the direction for fetching channel message history.
type PaginationDirection string

const (
	DirectionBefore PaginationDirection = "before"
	DirectionAfter  PaginationDirection = "after"
	DirectionAround PaginationDirection = "around"
)

type ListMessagesParams struct {
	ChannelID uuid.UUID
	UserID    uuid.UUID
	Direction PaginationDirection
	TargetID  *uuid.UUID // Required for BEFORE/AFTER when paging; optional/omitted for top of feed
	Limit     int32      // Requested limit (e.g., 50)
}

type ListMessagesResult struct {
	Messages      []Message `json:"messages"`
	HasMoreBefore bool      `json:"has_more_before"`
	HasMoreAfter  bool      `json:"has_more_after"`
}

type InitialMessagesResult struct {
	Messages      []Message  `json:"messages"`
	FirstUnreadID *uuid.UUID `json:"first_unread_id,omitempty"`
	HasMoreBefore bool       `json:"has_more_before"`
	HasMoreAfter  bool       `json:"has_more_after"`
}

// GetInitialChannelMessages orchestrates cold channel loads and unread positioning.
func (s *Service) GetInitialChannelMessages(ctx context.Context, channelID, userID uuid.UUID, limit int32) (*InitialMessagesResult, error) {
	if channelID == uuid.Nil || userID == uuid.Nil {
		return nil, errs.InvalidArgument("channel ID and user ID cannot be nil")
	}

	// 1. Check for first unread message
	firstUnread, err := s.repo.MessageGetFirstUnread(ctx, channelID, userID)
	if err != nil && !errs.IsNotFound(err) {
		return nil, err
	}

	var (
		listRes  *ListMessagesResult
		unreadID *uuid.UUID
	)

	// 2. Branching strategy
	if firstUnread != nil {
		id := firstUnread.ID()
		unreadID = &id

		// Jump to first unread message using DirectionAround
		listRes, err = s.ListMessages(ctx, ListMessagesParams{
			ChannelID: channelID,
			UserID:    userID,
			Direction: DirectionAround,
			TargetID:  unreadID,
			Limit:     limit,
		})
		if err != nil {
			return nil, err
		}
	} else {
		// User is fully caught up or channel is empty; fetch latest messages going backward
		listRes, err = s.ListMessages(ctx, ListMessagesParams{
			ChannelID: channelID,
			UserID:    userID,
			Direction: DirectionBefore,
			TargetID:  nil,
			Limit:     limit,
		})
		if err != nil {
			return nil, err
		}
	}

	return &InitialMessagesResult{
		Messages:      listRes.Messages,
		FirstUnreadID: unreadID,
		HasMoreBefore: listRes.HasMoreBefore,
		HasMoreAfter:  listRes.HasMoreAfter,
	}, nil
}

func (s *Service) ListMessages(ctx context.Context, params ListMessagesParams) (*ListMessagesResult, error) {
	if params.ChannelID == uuid.Nil || params.UserID == uuid.Nil {
		return nil, errs.InvalidArgument("channel ID and user ID cannot be nil")
	}

	if params.Limit <= 0 {
		params.Limit = 50
	} else if params.Limit > 100 {
		params.Limit = 100
	}

	// 1. Authorization check
	ok, err := s.repo.IsMember(ctx, params.ChannelID, params.UserID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, errs.PermissionDenied("you are not a member of this channel").Wrap(ErrNotParticipant)
	}

	var (
		cursorCreatedAt *time.Time
		cursorID        *uuid.UUID
	)

	// 2. Resolve cursor message if TargetID is provided
	if params.TargetID != nil && *params.TargetID != uuid.Nil {
		targetMsg, err := s.repo.MessageGet(ctx, *params.TargetID)
		if err != nil {
			return nil, err
		}
		if targetMsg.ChannelID() != params.ChannelID {
			return nil, errs.InvalidArgument("target message does not belong to specified channel")
		}
		createdAt := targetMsg.CreatedAt()
		cursorCreatedAt = &createdAt
		cursorID = params.TargetID
	}

	result := &ListMessagesResult{
		Messages: make([]Message, 0),
	}

	switch params.Direction {
	case DirectionBefore:
		// Request limit + 1 to check for older messages
		msgs, err := s.repo.MessageListByChannelBefore(ctx, params.ChannelID, cursorCreatedAt, cursorID, params.Limit+1)
		if err != nil {
			return nil, err
		}

		if int32(len(msgs)) > params.Limit {
			result.HasMoreBefore = true
			msgs = msgs[:params.Limit] // Trim the extra overflow item
		}

		// Edge Case D Fix: ListMessagesBefore fetches DESC (newest to oldest).
		// Reverse in-place so returned slice is strictly ASC (chronological).
		reverseMessages(msgs)
		result.Messages = msgs

		// Edge Case C Fix: Avoid false positive for HasMoreAfter when cursor is already at boundary.
		if cursorCreatedAt != nil && cursorID != nil {
			hasAfter, err := s.repo.HasMessagesAfter(ctx, params.ChannelID, *cursorCreatedAt, *cursorID)
			if err != nil {
				return nil, err
			}
			result.HasMoreAfter = hasAfter
		} else {
			result.HasMoreAfter = false
		}

	case DirectionAfter:
		// Request limit + 1 to check for newer messages
		msgs, err := s.repo.MessageListByChannelAfter(ctx, params.ChannelID, cursorCreatedAt, cursorID, params.Limit+1)
		if err != nil {
			return nil, err
		}

		if int32(len(msgs)) > params.Limit {
			result.HasMoreAfter = true
			msgs = msgs[:params.Limit] // Trim the extra overflow item
		}

		// ListMessagesAfter query returns ASC (chronological), so order is preserved as-is.
		result.Messages = msgs

		// Edge Case C Fix: Avoid false positive for HasMoreBefore when cursor is already at boundary.
		if cursorCreatedAt != nil && cursorID != nil {
			hasBefore, err := s.repo.HasMessagesBefore(ctx, params.ChannelID, *cursorCreatedAt, *cursorID)
			if err != nil {
				return nil, err
			}
			result.HasMoreBefore = hasBefore
		} else {
			result.HasMoreBefore = false
		}

	case DirectionAround:
		if cursorCreatedAt == nil || cursorID == nil {
			return nil, errs.InvalidArgument("target message ID is required for DirectionAround")
		}

		halfLimit := params.Limit / 2

		// Fetch older + target + newer using halfLimit
		msgs, err := s.repo.MessageListByChannelAround(ctx, params.ChannelID, *cursorCreatedAt, *cursorID, halfLimit)
		if err != nil {
			return nil, err
		}

		// Process and slice window boundaries around target message
		result.Messages, result.HasMoreBefore, result.HasMoreAfter = processAroundResults(msgs, *cursorID, halfLimit)

	default:
		return nil, errs.InvalidArgument(fmt.Sprintf("unsupported list direction: %s", params.Direction))
	}

	return result, nil
}

// processAroundResults ensures chronological ordering, locates the target message, and evaluates overflow boundaries.
func processAroundResults(msgs []Message, targetID uuid.UUID, halfLimit int32) ([]Message, bool, bool) {
	if len(msgs) == 0 {
		return msgs, false, false
	}

	// Edge Case D Guard: Guarantee chronological order (oldest -> newest) before slicing indexes
	if len(msgs) > 1 && msgs[0].CreatedAt().After(msgs[len(msgs)-1].CreatedAt()) {
		reverseMessages(msgs)
	} else {
		// Fallback stable sort if timestamp order is mixed or non-deterministic
		sort.SliceStable(msgs, func(i, j int) bool {
			if msgs[i].CreatedAt().Equal(msgs[j].CreatedAt()) {
				return msgs[i].ID().String() < msgs[j].ID().String()
			}
			return msgs[i].CreatedAt().Before(msgs[j].CreatedAt())
		})
	}

	targetIdx := -1
	for i, m := range msgs {
		if m.ID() == targetID {
			targetIdx = i
			break
		}
	}

	if targetIdx == -1 {
		return msgs, false, false
	}

	// Count elements older than target (before targetIdx)
	olderCount := int32(targetIdx)
	hasMoreBefore := olderCount > halfLimit

	// Count elements newer than target (after targetIdx)
	newerCount := int32(len(msgs) - 1 - targetIdx)
	hasMoreAfter := newerCount > halfLimit

	startIdx := 0
	if hasMoreBefore {
		startIdx = int(olderCount - halfLimit)
	}

	endIdx := len(msgs)
	if hasMoreAfter {
		endIdx = targetIdx + 1 + int(halfLimit)
	}

	return msgs[startIdx:endIdx], hasMoreBefore, hasMoreAfter
}

// reverseMessages inverts a slice of Message entities in place.
func reverseMessages(msgs []Message) {
	for i, j := 0, len(msgs)-1; i < j; i, j = i+1, j-1 {
		msgs[i], msgs[j] = msgs[j], msgs[i]
	}
}

// GetFirstUnreadMessage retrieves the first unread message for a user in a channel.
func (s *Service) GetFirstUnreadMessage(ctx context.Context, channelID, userID uuid.UUID) (*Message, error) {
	if channelID == uuid.Nil || userID == uuid.Nil {
		return nil, errs.InvalidArgument("channel ID and user ID cannot be nil")
	}

	ok, err := s.repo.IsMember(ctx, channelID, userID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, errs.PermissionDenied("you are not a member of this channel").Wrap(ErrNotParticipant)
	}

	msg, err := s.repo.MessageGetFirstUnread(ctx, channelID, userID)
	if err != nil {
		if errs.IsNotFound(err) {
			return nil, nil // No unread messages
		}
		return nil, err
	}

	return msg, nil
}

// AddReaction adds a reaction emoji to a message.
func (s *Service) AddReaction(ctx context.Context, messageID, userID uuid.UUID, rawEmoji string) error {
	emojiVO, err := NewEmoji(rawEmoji)
	if err != nil {
		return errs.InvalidArgument(err.Error()).Wrap(err)
	}

	return s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		msg, err := s.repo.MessageGet(txCtx, messageID)
		if err != nil {
			if errs.IsNotFound(err) {
				return errs.NotFound("message not found").Wrap(err)
			}
			return err
		}

		if _, err := s.repo.ReactionAdd(txCtx, messageID, userID, emojiVO.String()); err != nil {
			return err
		}

		_, err = s.outbox.Publish(txCtx, EventReactionAdded, ReactionPayload{
			MessageID: messageID,
			ChannelID: msg.ChannelID(),
			UserID:    userID,
			Emoji:     emojiVO.String(),
		})
		return err
	})
}

// RemoveReaction removes a user reaction from a message.
func (s *Service) RemoveReaction(ctx context.Context, messageID, userID uuid.UUID, rawEmoji string) error {
	emojiVO, err := NewEmoji(rawEmoji)
	if err != nil {
		return errs.InvalidArgument(err.Error()).Wrap(err)
	}

	return s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		msg, err := s.repo.MessageGet(txCtx, messageID)
		if err != nil {
			if errs.IsNotFound(err) {
				return errs.NotFound("message not found").Wrap(err)
			}
			return err
		}

		if err := s.repo.ReactionRemove(txCtx, messageID, userID, emojiVO); err != nil {
			return err
		}

		_, err = s.outbox.Publish(txCtx, EventReactionRemoved, ReactionPayload{
			MessageID: messageID,
			ChannelID: msg.ChannelID(),
			UserID:    userID,
			Emoji:     emojiVO.String(),
		})
		return err
	})
}

// MarkAsRead updates a member's unread position and clears their mention count.
func (s *Service) MarkAsRead(ctx context.Context, channelID, userID, messageID uuid.UUID) error {
	ok, err := s.repo.IsMember(ctx, channelID, userID)
	if err != nil {
		return err
	}
	if !ok {
		return errs.PermissionDenied("you are not a member of this channel").Wrap(ErrNotParticipant)
	}

	now := time.Now().UTC()
	return s.repo.MemberUpdateReadState(ctx, channelID, userID, &messageID, now)
}

// SendTypingSignal records an active typing state in Redis.
func (s *Service) SendTypingSignal(ctx context.Context, channelID, userID uuid.UUID) error {
	ok, err := s.repo.IsMember(ctx, channelID, userID)
	if err != nil {
		return err
	}
	if !ok {
		return errs.PermissionDenied("you are not a member of this channel").Wrap(ErrNotParticipant)
	}

	if err := s.typingStore.SetTyping(ctx, channelID, userID); err != nil {
		return errs.Internal("failed to set typing indicator").Wrap(err)
	}

	return nil
}

func sortUUIDs(a, b uuid.UUID) (uuid.UUID, uuid.UUID) {
	if a.String() < b.String() {
		return a, b
	}
	return b, a
}
