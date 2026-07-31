package channel

import (
	"context"
	"errors"
	"fmt"
	"time"

	"bonfire-api/internal/db"
	"bonfire-api/internal/errs"
	"bonfire-api/internal/outbox"
	"bonfire-api/internal/pkg/ptr"

	"github.com/google/uuid"
)

const MinMembers = 1
const MaxMembers = 10
const MaxPeers = 9

var (
	ErrNotParticipant  = errors.New("user is not a participant of this channel")
	ErrCannotEditOther = errors.New("cannot edit or delete a message authored by someone else")
	ErrNotOwner        = errors.New("only the channel owner can perform this action")
)

type OutboxRepository interface {
	Publish(ctx context.Context, variant string, payload any) (*outbox.Event, error)
}

type RelationshipRepository interface {
	HasBlockBetweenUserAndPeers(ctx context.Context, userID uuid.UUID, peerIDs []uuid.UUID) (bool, error)
}

type Repository interface {
	Create(ctx context.Context, ch *Channel) (*Channel, error)
	Delete(ctx context.Context, id uuid.UUID) error
	Get(ctx context.Context, id uuid.UUID) (*Channel, error)
	GetForMember(ctx context.Context, channelID uuid.UUID, memberID uuid.UUID) (*Channel, error)
	GetForMemberUpdate(ctx context.Context, channelID, memberID uuid.UUID) (*Channel, error)
	HasMessagesAfter(ctx context.Context, channelID uuid.UUID, createdAt time.Time, id uuid.UUID) (bool, error)
	HasMessagesBefore(ctx context.Context, channelID uuid.UUID, createdAt time.Time, id uuid.UUID) (bool, error)
	IsMember(ctx context.Context, channelID uuid.UUID, userID uuid.UUID) (bool, error)
	MemberAddBatch(ctx context.Context, members []*Member) error
	MemberCount(ctx context.Context, channelID uuid.UUID) (int32, error)
	MemberGet(ctx context.Context, channelID uuid.UUID, userID uuid.UUID) (*Member, error)
	MemberGetUnreadCount(ctx context.Context, channelID uuid.UUID, userID uuid.UUID) (int32, error)
	MemberIncrementMentionCountBatch(ctx context.Context, channelID uuid.UUID, userIDs []uuid.UUID) error
	MemberListByChannel(ctx context.Context, channelID uuid.UUID) ([]*Member, error)
	MemberListItemsByChannel(
		ctx context.Context,
		channelID uuid.UUID,
		cursorCreatedAt *time.Time,
		cursorUserID *uuid.UUID,
		limit int32,
	) ([]*MemberListItem, error)
	MemberRemove(ctx context.Context, channelID uuid.UUID, userID uuid.UUID) error
	MemberResetMentionCount(ctx context.Context, channelID uuid.UUID, userID uuid.UUID) error
	MemberUpdateLastRead(ctx context.Context, channelID uuid.UUID, userID uuid.UUID, messageID *uuid.UUID, lastReadAt time.Time) error
	MessageCreate(ctx context.Context, msg *Message) (*Message, error)
	MessageDelete(ctx context.Context, id uuid.UUID) error
	MessageGet(ctx context.Context, messageID uuid.UUID) (*Message, error)
	MessageGetAggregate(ctx context.Context, id uuid.UUID, userID *uuid.UUID) (*MessageAggregate, error)
	MessageGetFirstUnread(ctx context.Context, channelID uuid.UUID, userID uuid.UUID) (*Message, error)
	MessageGetLatest(ctx context.Context, channelID uuid.UUID) (*Message, error)
	MessageSetPinned(ctx context.Context, msg *Message) (*Message, error)
	MessageUpdateContent(ctx context.Context, msg *Message) (*Message, error)
	MemberUpdateRead(
		ctx context.Context,
		channelID uuid.UUID,
		userID uuid.UUID,
		messageID *uuid.UUID,
		lastReadAt time.Time,
	) error
	ReactionAdd(ctx context.Context, messageID uuid.UUID, userID uuid.UUID, emoji string) (*Reaction, error)
	ReactionRemove(ctx context.Context, messageID uuid.UUID, userID uuid.UUID, emoji Emoji) error
	Update(ctx context.Context, ch *Channel) (*Channel, error)
	UpdateLastMessage(ctx context.Context, ch *Channel) (*Channel, error)
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

// ============================================================================
// CHANNELS
// ============================================================================

// CreateGroup creates a new channel with TypeGroup Type = 2 with channel members.
func (s *Service) CreateGroup(ctx context.Context, rawActorID uuid.UUID, rawMemberIDs []uuid.UUID) (*Channel, error) {
	// Prevent requiring O(n) by constraining array length
	if len(rawMemberIDs) > MaxPeers {
		return nil, errs.InvalidArgument(fmt.Sprintf("member list cannot exceed %d items", MaxPeers))
	}

	// Parse inputs
	actorID, err := NewID(rawActorID)
	if err != nil {
		return nil, errs.InvalidArgument("invalid creator ID").Wrap(err)
	}

	memberIDs, err := NewIDs(rawMemberIDs)
	if err != nil {
		return nil, errs.InvalidArgument("invalid member ID in list").Wrap(err)
	}

	// Dedup ids
	totalMembers := make(map[ID]struct{}, len(memberIDs)+1)
	totalMembers[actorID] = struct{}{}
	for _, id := range memberIDs {
		totalMembers[id] = struct{}{}
	}

	// Member Limits Guard
	if len(totalMembers) < MinMembers {
		return nil, errs.InvalidArgument(fmt.Sprintf("group DMs require at least %d members", MinMembers))
	}
	if len(totalMembers) > MaxMembers {
		return nil, errs.InvalidArgument(fmt.Sprintf("group DMs cannot exceed %d members", MaxMembers))
	}

	// Privacy Guard: Ensure creator hasn't blocked/been blocked by any invited peer
	if len(totalMembers) > 1 {
		peerUUIDs := make([]uuid.UUID, 0, len(totalMembers)-1)
		for id := range totalMembers {
			if !id.Equals(actorID) {
				peerUUIDs = append(peerUUIDs, id.UUID())
			}
		}
		hasBlock, err := s.relationship.HasBlockBetweenUserAndPeers(ctx, actorID.UUID(), peerUUIDs)
		if err != nil {
			return nil, err
		}
		if hasBlock {
			return nil, errs.InvalidArgument("cannot create group DM containing blocked users")
		}
	}

	// Instantiate Entities
	ch, err := New(TypeGroup, nil, nil)
	if err != nil {
		return nil, errs.InvalidArgument(err.Error()).Wrap(err)
	}

	memberBatch := make([]*Member, 0, len(totalMembers))
	for id := range totalMembers {
		m, err := NewMember(uuid.UUID(ch.ID()), id.UUID())
		if err != nil {
			return nil, errs.InvalidArgument("invalid member properties").Wrap(err)
		}
		memberBatch = append(memberBatch, m)
	}

	var newChannel *Channel
	err = s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		// Create channel
		row, err := s.repo.Create(txCtx, ch)
		if err != nil {
			return err
		}

		// Add members
		if err := s.repo.MemberAddBatch(txCtx, memberBatch); err != nil {
			return err
		}

		// TODO: Publish outbox event

		newChannel = row
		return nil
	})

	if err != nil {
		return nil, err
	}

	// Return new channel
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

type UpdateMetaInput struct {
	ChannelID uuid.UUID
	ActorID   uuid.UUID
	Name      *string
	IconURL   *string
}

// Update encapsulates metadata changes requested by a user
func (s *Service) UpdateMeta(ctx context.Context, input UpdateMetaInput) (*Channel, error) {
	ch, err := s.repo.Get(ctx, input.ChannelID)
	if err != nil {
		return nil, err
	}

	updated, err := ch.UpdateMeta(input.Name, input.IconURL)
	if err != nil {
		return nil, err
	}

	if !updated {
		return ch, nil
	}

	// 5. Persist entity changes and emit Outbox event in a single atomic transaction
	err = s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		if _, err := s.repo.Update(txCtx, ch); err != nil {
			return err
		}

		payload := ChannelUpdatedPayload{
			ChannelID: uuid.UUID(ch.ID()),
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

// ============================================================================
// MEMBERS
// ============================================================================

// AddMembers adds a batch of users to an existing group channel or upgrades a DM to a group channel.
func (s *Service) AddMembers(ctx context.Context, rawChannelID uuid.UUID, rawActorID uuid.UUID, rawMemberIDs []uuid.UUID) error {
	// At least one member ID required
	if len(rawMemberIDs) == 0 {
		return errs.InvalidArgument("member IDs cannot be empty")
	}

	// Prevent requiring O(n) work by constraining array length
	if len(rawMemberIDs) > MaxPeers {
		return errs.InvalidArgument(fmt.Sprintf("member list cannot exceed %d items", MaxPeers))
	}

	// Parse inputs
	channelID, err := NewID(rawChannelID)
	if err != nil {
		return errs.InvalidArgument("invalid channel ID").Wrap(err)
	}

	actorID, err := NewID(rawActorID)
	if err != nil {
		return errs.InvalidArgument("invalid actor ID").Wrap(err)
	}

	memberIDs, err := NewIDs(rawMemberIDs)
	if err != nil {
		return errs.InvalidArgument("invalid member ID in list").Wrap(err)
	}

	// Deduplicate rawMemberIDs and strip out actorID
	totalPeers := make(map[ID]struct{}, len(memberIDs))
	for _, id := range memberIDs {
		if !id.Equals(actorID) {
			totalPeers[id] = struct{}{}
		}
	}

	if len(totalPeers) == 0 {
		return errs.InvalidArgument("cannot add yourself as the only new member")
	}

	peerUUIDs := make([]uuid.UUID, 0, len(totalPeers))
	for id := range totalPeers {
		peerUUIDs = append(peerUUIDs, id.UUID())
	}

	// Verify no blocked relationship exists between actor and prospective members
	hasBlock, err := s.relationship.HasBlockBetweenUserAndPeers(ctx, actorID.UUID(), peerUUIDs)
	if err != nil {
		return err
	}
	if hasBlock {
		return errs.InvalidArgument("cannot create group DM containing blocked users")
	}

	// -------------------------------------------------------------------------
	// 2. Transactional Lock & State Mutations
	// -------------------------------------------------------------------------
	return s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		// Verify actor membership AND lock the channel row atomically
		ch, err := s.repo.GetForMemberUpdate(txCtx, channelID.UUID(), actorID.UUID())
		if err != nil {
			return err
		}

		// B. IF DIRECT MESSAGE: Upgrade DM to a new Group Channel
		if ch.Type() == TypeDirect {
			_, err = s.CreateGroup(txCtx, actorID.UUID(), peerUUIDs)
			if err != nil {
				return err
			}
			return nil
		}

		// Fetch existing members under the lock
		existingMembers, err := s.repo.MemberListByChannel(txCtx, channelID.UUID())
		if err != nil {
			return err
		}

		// C. IF GROUP CHANNEL: Enforce Max Member Limit & Batch Insert
		existingMap := make(map[uuid.UUID]struct{}, len(existingMembers))
		for _, m := range existingMembers {
			existingMap[uuid.UUID(m.UserID())] = struct{}{}
		}

		// Filter out members already present in the channel
		newMembersToInsert := make([]*Member, 0, len(peerUUIDs))
		for _, mUUID := range peerUUIDs {
			if _, exists := existingMap[mUUID]; exists {
				continue
			}

			member, err := NewMember(channelID.UUID(), mUUID)
			if err != nil {
				return errs.InvalidArgument("invalid member id").Wrap(err)
			}
			newMembersToInsert = append(newMembersToInsert, member)
		}

		if len(newMembersToInsert) == 0 {
			return errs.InvalidArgument("all provided users are already members of this channel")
		}

		// Enforce capacity rule: Existing + New Unique Members <= MaxMembers
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

		// D. Outbox Event
		// addedIDs := make([]uuid.UUID, 0, len(newMembersToInsert))
		// for _, m := range newMembersToInsert {
		// 	addedIDs = append(addedIDs, m.UserID())
		// }

		// _, err = s.outbox.Publish(txCtx, "channel.members_added", ChannelMembersAddedPayload{
		// 	ChannelID: channelID.UUID(),
		// 	ActorID:   actorID.UUID(),
		// 	MemberIDs: addedIDs,
		// })
		// if err != nil {
		// 	return err
		// }

		return nil
	})
}

// Leave handles a user leaving a channel, performing cleanup if they are the last member.
func (s *Service) Leave(ctx context.Context, rawChannelID uuid.UUID, rawActorID uuid.UUID) error {
	channelID, err := NewID(rawChannelID)
	if err != nil {
		return errs.InvalidArgument("invalid channel ID").Wrap(err)
	}

	actorID, err := NewID(rawActorID)
	if err != nil {
		return errs.InvalidArgument("invalid actor ID").Wrap(err)
	}

	return s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		// Lock channel row & verify membership via FOR UPDATE
		ch, err := s.repo.GetForMemberUpdate(txCtx, channelID.UUID(), actorID.UUID())
		if err != nil {
			return err
		}

		if ch.Type() == TypeDirect {
			return errs.InvalidArgument("cannot leave a direct message channel")
		}

		// Efficiently count remaining members without fetching row data
		count, err := s.repo.MemberCount(txCtx, channelID.UUID())
		if err != nil {
			return err
		}

		// 3. If last member, delete channel (cascades related tables) and exit
		if count <= 1 {
			if err := s.repo.Delete(txCtx, channelID.UUID()); err != nil {
				return err
			}

			// _, err = s.outbox.Publish(txCtx, "channel.deleted", ChannelDeletedPayload{
			// 	ChannelID: channelID,
			// 	ActorID:   actorID,
			// })
			// return err

			return nil // Critical: Return early so we don't attempt MemberRemove on a deleted channel
		}

		// 4. Remove member record
		if err := s.repo.MemberRemove(txCtx, channelID.UUID(), actorID.UUID()); err != nil {
			return err
		}

		// // 5. Emit outbox event for gateway fanout
		// _, err = s.outbox.Publish(txCtx, "channel.member_removed", ChannelMemberRemovedPayload{
		// 	ChannelID:    channelID,
		// 	ActorID:      actorID,
		// 	TargetUserID: actorID,
		// })
		// return err

		return nil
	})
}

// ListMembers returns keyset-paginated member roster records for a channel
// after confirming the requesting user is a participant.
func (s *Service) ListMembers(
	ctx context.Context,
	rawChannelID, rawActorID uuid.UUID,
	cursorCreatedAt *time.Time,
	cursorUserID *uuid.UUID,
	limit int32,
) ([]*MemberListItem, error) {
	// Parse inputs
	channelID, err := NewID(rawChannelID)
	if err != nil {
		return nil, errs.InvalidArgument("invalid channel ID").Wrap(err)
	}

	actorID, err := NewID(rawActorID)
	if err != nil {
		return nil, errs.InvalidArgument("invalid actor ID").Wrap(err)
	}

	if limit <= 0 {
		return nil, errs.InvalidArgument("limit must be greater than 0")
	}

	// 1. Authorization Guard: Ensure the requesting user belongs to the channel
	ok, err := s.repo.IsMember(ctx, channelID.UUID(), actorID.UUID())
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, errs.PermissionDenied("you are not a member of this channel").Wrap(ErrNotParticipant)
	}

	// 2. Fetch paginated member roster projection
	members, err := s.repo.MemberListItemsByChannel(
		ctx,
		channelID.UUID(),
		cursorCreatedAt,
		cursorUserID,
		limit,
	)
	if err != nil {
		return nil, err
	}

	// TODO: Add presence

	return members, nil
}

// UpdateMemberRead updates the user's read marker for a channel and emits an outbox event.
func (s *Service) UpdateMemberRead(
	ctx context.Context,
	rawChannelID, rawActorID uuid.UUID,
	messageID *uuid.UUID,
	readAt time.Time,
) error {
	channelID, err := NewID(rawChannelID)
	if err != nil {
		return errs.InvalidArgument("invalid channel ID").Wrap(err)
	}

	actorID, err := NewID(rawActorID)
	if err != nil {
		return errs.InvalidArgument("invalid actor ID").Wrap(err)
	}

	if readAt.IsZero() {
		readAt = time.Now()
	}
	readAt = readAt.UTC()

	// Wrap in transaction so DB update and outbox payload commit atomically
	return s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		// 1. Authorization Guard: Check channel membership
		isMember, err := s.repo.IsMember(txCtx, channelID.UUID(), actorID.UUID())
		if err != nil {
			return err
		}
		if !isMember {
			return errs.PermissionDenied("you are not a member of this channel").Wrap(ErrNotParticipant)
		}

		// 2. Persist via Repository (Monotonicity handled via SQL GREATEST/CASE)
		if err := s.repo.MemberUpdateRead(txCtx, channelID.UUID(), actorID.UUID(), messageID, readAt); err != nil {
			return err
		}

		// 3. Side Effects: Publish to outbox for Gateway fanout
		_, err = s.outbox.Publish(txCtx, EventChannelReadUpdated, ChannelReadUpdatedPayload{
			ChannelID:         channelID.UUID(),
			UserID:            actorID.UUID(),
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

// ============================================================================
// MESSAGES
// ============================================================================

// SendMessage validates domain invariants, persists the message, updates channel metadata, and publishes outbox events.
func (s *Service) SendMessage(ctx context.Context, rawChannelID, rawAuthorID uuid.UUID, rawContent *string, replyToID *uuid.UUID) (*Message, error) {
	channelID, err := NewID(rawChannelID)
	if err != nil {
		return nil, errs.InvalidArgument("invalid channel ID").Wrap(err)
	}

	authorID, err := NewID(rawAuthorID)
	if err != nil {
		return nil, errs.InvalidArgument("invalid author ID").Wrap(err)
	}

	contentVO, err := NewContentPtr(rawContent)
	if err != nil {
		return nil, errs.InvalidArgument(err.Error()).Wrap(err)
	}

	var contentStr *string
	if contentVO != nil {
		contentStr = ptr.To(contentVO.String())
	}

	msgVO, err := NewMessage(channelID.UUID(), ptr.To(authorID.UUID()), replyToID, contentStr)
	if err != nil {
		return nil, errs.InvalidArgument(err.Error()).Wrap(err)
	}

	var msg *Message
	err = s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		ch, err := s.repo.GetForMemberUpdate(txCtx, channelID.UUID(), authorID.UUID())
		if err != nil {
			return err
		}

		if replyToID != nil {
			parentMsg, err := s.repo.MessageGet(txCtx, *replyToID)
			if err != nil {
				return errs.NotFound("reply target message not found").Wrap(err)
			}
			if parentMsg.ChannelID() != channelID {
				return errs.InvalidArgument("cannot reply to a message in a different channel")
			}
		}

		msg, err = s.repo.MessageCreate(txCtx, msgVO)
		if err != nil {
			return err
		}

		ch.SetLastMessage(msg.ID())

		_, err = s.repo.UpdateLastMessage(txCtx, ch)
		if err != nil {
			return err
		}

		// Use local authorID to prevent nil pointer issues if AuthorID is optional on Message
		if err := s.repo.MemberUpdateRead(txCtx, channelID.UUID(), authorID.UUID(), ptr.To(msg.ID().UUID()), msg.CreatedAt()); err != nil {
			return err
		}

		// _, err = s.outbox.Publish(txCtx, EventMessageCreated, MessageCreatedPayload{
		// 	MessageID: msg.ID(),
		// 	ChannelID: msg.ChannelID(),
		// 	AuthorID:  msg.AuthorID(),
		// 	Content:   msg.Content().String(),
		// 	ReplyToID: msg.ReplyToMessageID(),
		// 	CreatedAt: msg.CreatedAt(),
		// })
		// return err

		return nil
	})
	if err != nil {
		return nil, err
	}

	return msg, nil
}

// // EditMessage allows the author to update message content.
// func (s *Service) EditMessage(ctx context.Context, messageID, authorID uuid.UUID, newRawContent string) (*Message, error) {
// 	if messageID == uuid.Nil || authorID == uuid.Nil {
// 		return nil, errs.InvalidArgument("message ID and author ID cannot be nil")
// 	}

// 	contentVO, err := NewContent(newRawContent)
// 	if err != nil {
// 		return nil, errs.InvalidArgument(err.Error()).Wrap(err)
// 	}

// 	var updated *Message

// 	err = s.tx.ExecTx(ctx, func(txCtx context.Context) error {
// 		msg, err := s.repo.MessageGet(txCtx, messageID)
// 		if err != nil {
// 			return err
// 		}

// 		if msg.AuthorID() == nil || *msg.AuthorID() != authorID {
// 			return errs.PermissionDenied("cannot edit messages sent by another user").Wrap(ErrCannotEditOther)
// 		}

// 		msg.EditContent(contentVO)

// 		if err := s.repo.MessageUpdateContent(txCtx, msg); err != nil {
// 			return err
// 		}

// 		_, err = s.outbox.Publish(txCtx, EventMessageUpdated, MessageUpdatedPayload{
// 			MessageID: msg.ID(),
// 			ChannelID: msg.ChannelID(),
// 			AuthorID:  msg.AuthorID(),
// 			Content:   msg.Content().String(),
// 			EditedAt:  msg.EditedAt(),
// 		})
// 		if err != nil {
// 			return err
// 		}

// 		updated = msg
// 		return nil
// 	})
// 	if err != nil {
// 		return nil, err
// 	}

// 	return updated, nil
// }

// // DeleteMessage deletes a message if requested by its author or an authorized user.
// func (s *Service) DeleteMessage(ctx context.Context, messageID, actorID uuid.UUID) error {
// 	if messageID == uuid.Nil || actorID == uuid.Nil {
// 		return errs.InvalidArgument("message ID and actor ID cannot be nil")
// 	}

// 	return s.tx.ExecTx(ctx, func(txCtx context.Context) error {
// 		msg, err := s.repo.MessageGet(txCtx, messageID)
// 		if err != nil {
// 			return err
// 		}

// 		// 1. Authorization Guard: Check if actor is author
// 		isAuthor := msg.AuthorID() != nil && *msg.AuthorID() == actorID
// 		if !isAuthor {
// 			return errs.PermissionDenied("cannot delete messages sent by another user").Wrap(ErrCannotEditOther)
// 		}

// 		// Fetch channel to check if this message is the channel's last_message_id
// 		ch, err := s.repo.Get(txCtx, msg.ChannelID())
// 		if err != nil {
// 			return err
// 		}

// 		// 2. Persist deletion in database
// 		if err := s.repo.MessageDelete(txCtx, messageID); err != nil {
// 			return err
// 		}

// 		// 3. If deleted message was the last_message_id, recalculate and update
// 		if ch.LastMessageID() != nil && *ch.LastMessageID() == messageID {
// 			latestMsg, err := s.repo.MessageGetLatest(txCtx, msg.ChannelID())
// 			if err != nil {
// 				return err
// 			}

// 			var nextMsgID *uuid.UUID
// 			var updatedAt = time.Now().UTC()

// 			if latestMsg != nil {
// 				id := latestMsg.ID()
// 				nextMsgID = &id
// 				updatedAt = latestMsg.CreatedAt()
// 			}

// 			if err := s.repo.UpdateLastMessage(txCtx, msg.ChannelID(), nextMsgID, updatedAt); err != nil {
// 				return err
// 			}
// 		}

// 		// 4. Publish outbox event
// 		now := time.Now().UTC()
// 		_, err = s.outbox.Publish(txCtx, EventMessageDeleted, MessageDeletedPayload{
// 			MessageID: msg.ID(),
// 			ChannelID: msg.ChannelID(),
// 			ActorID:   actorID,
// 			DeletedAt: now,
// 		})
// 		return err
// 	})
// }

// // ToggleMessagePin handles pinning or unpinning a message for channel members.
// func (s *Service) ToggleMessagePin(ctx context.Context, messageID, userID uuid.UUID, isPinned bool) (*Message, error) {
// 	if messageID == uuid.Nil || userID == uuid.Nil {
// 		return nil, errs.InvalidArgument("message ID and user ID cannot be nil")
// 	}

// 	var updated *Message

// 	err := s.tx.ExecTx(ctx, func(txCtx context.Context) error {
// 		msg, err := s.repo.MessageGet(txCtx, messageID)
// 		if err != nil {
// 			return err
// 		}

// 		// 1. Authorization Guard: Check channel membership for the pin action
// 		ok, err := s.repo.IsMember(txCtx, msg.ChannelID(), userID)
// 		if err != nil {
// 			return err
// 		}
// 		if !ok {
// 			return errs.PermissionDenied("you are not a member of this channel").Wrap(ErrNotParticipant)
// 		}

// 		// 2. Mutate state & persist
// 		msg.SetPinned(isPinned)

// 		if err := s.repo.MessageSetPinned(txCtx, msg.ID(), isPinned); err != nil {
// 			return err
// 		}

// 		// 3. Publish outbox event
// 		_, err = s.outbox.Publish(txCtx, EventMessagePinned, MessagePinnedPayload{
// 			MessageID: msg.ID(),
// 			ChannelID: msg.ChannelID(),
// 			IsPinned:  msg.IsPinned(),
// 		})
// 		if err != nil {
// 			return err
// 		}

// 		updated = msg
// 		return nil
// 	})
// 	if err != nil {
// 		return nil, err
// 	}

// 	return updated, nil
// }

// // TODO: aggregate reactions into message

// // PaginationDirection defines the direction for fetching channel message history.
// type PaginationDirection string

// const (
// 	DirectionBefore PaginationDirection = "before"
// 	DirectionAfter  PaginationDirection = "after"
// 	DirectionAround PaginationDirection = "around"
// )

// type ListMessagesParams struct {
// 	ChannelID uuid.UUID
// 	UserID    uuid.UUID
// 	Direction PaginationDirection
// 	TargetID  *uuid.UUID // Required for BEFORE/AFTER when paging; optional/omitted for top of feed
// 	Limit     int32      // Requested limit (e.g., 50)
// }

// type ListMessagesResult struct {
// 	Messages      []Message `json:"messages"`
// 	HasMoreBefore bool      `json:"has_more_before"`
// 	HasMoreAfter  bool      `json:"has_more_after"`
// }

// type InitialMessagesResult struct {
// 	Messages      []Message  `json:"messages"`
// 	FirstUnreadID *uuid.UUID `json:"first_unread_id,omitempty"`
// 	HasMoreBefore bool       `json:"has_more_before"`
// 	HasMoreAfter  bool       `json:"has_more_after"`
// }

// // GetInitialChannelMessages orchestrates cold channel loads and unread positioning.
// func (s *Service) GetInitialChannelMessages(ctx context.Context, channelID, userID uuid.UUID, limit int32) (*InitialMessagesResult, error) {
// 	if channelID == uuid.Nil || userID == uuid.Nil {
// 		return nil, errs.InvalidArgument("channel ID and user ID cannot be nil")
// 	}

// 	// 1. Check for first unread message
// 	firstUnread, err := s.repo.MessageGetFirstUnread(ctx, channelID, userID)
// 	if err != nil && !errs.IsNotFound(err) {
// 		return nil, err
// 	}

// 	var (
// 		listRes  *ListMessagesResult
// 		unreadID *uuid.UUID
// 	)

// 	// 2. Branching strategy
// 	if firstUnread != nil {
// 		id := firstUnread.ID()
// 		unreadID = &id

// 		// Jump to first unread message using DirectionAround
// 		listRes, err = s.ListMessages(ctx, ListMessagesParams{
// 			ChannelID: channelID,
// 			UserID:    userID,
// 			Direction: DirectionAround,
// 			TargetID:  unreadID,
// 			Limit:     limit,
// 		})
// 		if err != nil {
// 			return nil, err
// 		}
// 	} else {
// 		// User is fully caught up or channel is empty; fetch latest messages going backward
// 		listRes, err = s.ListMessages(ctx, ListMessagesParams{
// 			ChannelID: channelID,
// 			UserID:    userID,
// 			Direction: DirectionBefore,
// 			TargetID:  nil,
// 			Limit:     limit,
// 		})
// 		if err != nil {
// 			return nil, err
// 		}
// 	}

// 	return &InitialMessagesResult{
// 		Messages:      listRes.Messages,
// 		FirstUnreadID: unreadID,
// 		HasMoreBefore: listRes.HasMoreBefore,
// 		HasMoreAfter:  listRes.HasMoreAfter,
// 	}, nil
// }

// func (s *Service) ListMessages(ctx context.Context, params ListMessagesParams) (*ListMessagesResult, error) {
// 	if params.ChannelID == uuid.Nil || params.UserID == uuid.Nil {
// 		return nil, errs.InvalidArgument("channel ID and user ID cannot be nil")
// 	}

// 	if params.Limit <= 0 {
// 		params.Limit = 50
// 	} else if params.Limit > 100 {
// 		params.Limit = 100
// 	}

// 	// 1. Authorization check
// 	ok, err := s.repo.IsMember(ctx, params.ChannelID, params.UserID)
// 	if err != nil {
// 		return nil, err
// 	}
// 	if !ok {
// 		return nil, errs.PermissionDenied("you are not a member of this channel").Wrap(ErrNotParticipant)
// 	}

// 	var (
// 		cursorCreatedAt *time.Time
// 		cursorID        *uuid.UUID
// 	)

// 	// 2. Resolve cursor message if TargetID is provided
// 	if params.TargetID != nil && *params.TargetID != uuid.Nil {
// 		targetMsg, err := s.repo.MessageGet(ctx, *params.TargetID)
// 		if err != nil {
// 			return nil, err
// 		}
// 		if targetMsg.ChannelID() != params.ChannelID {
// 			return nil, errs.InvalidArgument("target message does not belong to specified channel")
// 		}
// 		createdAt := targetMsg.CreatedAt()
// 		cursorCreatedAt = &createdAt
// 		cursorID = params.TargetID
// 	}

// 	result := &ListMessagesResult{
// 		Messages: make([]Message, 0),
// 	}

// 	switch params.Direction {
// 	case DirectionBefore:
// 		// Request limit + 1 to check for older messages
// 		msgs, err := s.repo.MessageListByChannelBefore(ctx, params.ChannelID, cursorCreatedAt, cursorID, params.Limit+1)
// 		if err != nil {
// 			return nil, err
// 		}

// 		if int32(len(msgs)) > params.Limit {
// 			result.HasMoreBefore = true
// 			msgs = msgs[:params.Limit] // Trim the extra overflow item
// 		}

// 		// Edge Case D Fix: ListMessagesBefore fetches DESC (newest to oldest).
// 		// Reverse in-place so returned slice is strictly ASC (chronological).
// 		reverseMessages(msgs)
// 		result.Messages = msgs

// 		// Edge Case C Fix: Avoid false positive for HasMoreAfter when cursor is already at boundary.
// 		if cursorCreatedAt != nil && cursorID != nil {
// 			hasAfter, err := s.repo.HasMessagesAfter(ctx, params.ChannelID, *cursorCreatedAt, *cursorID)
// 			if err != nil {
// 				return nil, err
// 			}
// 			result.HasMoreAfter = hasAfter
// 		} else {
// 			result.HasMoreAfter = false
// 		}

// 	case DirectionAfter:
// 		// Request limit + 1 to check for newer messages
// 		msgs, err := s.repo.MessageListByChannelAfter(ctx, params.ChannelID, cursorCreatedAt, cursorID, params.Limit+1)
// 		if err != nil {
// 			return nil, err
// 		}

// 		if int32(len(msgs)) > params.Limit {
// 			result.HasMoreAfter = true
// 			msgs = msgs[:params.Limit] // Trim the extra overflow item
// 		}

// 		// ListMessagesAfter query returns ASC (chronological), so order is preserved as-is.
// 		result.Messages = msgs

// 		// Edge Case C Fix: Avoid false positive for HasMoreBefore when cursor is already at boundary.
// 		if cursorCreatedAt != nil && cursorID != nil {
// 			hasBefore, err := s.repo.HasMessagesBefore(ctx, params.ChannelID, *cursorCreatedAt, *cursorID)
// 			if err != nil {
// 				return nil, err
// 			}
// 			result.HasMoreBefore = hasBefore
// 		} else {
// 			result.HasMoreBefore = false
// 		}

// 	case DirectionAround:
// 		if cursorCreatedAt == nil || cursorID == nil {
// 			return nil, errs.InvalidArgument("target message ID is required for DirectionAround")
// 		}

// 		halfLimit := params.Limit / 2

// 		// Fetch older + target + newer using halfLimit
// 		msgs, err := s.repo.MessageListByChannelAround(ctx, params.ChannelID, *cursorCreatedAt, *cursorID, halfLimit)
// 		if err != nil {
// 			return nil, err
// 		}

// 		// Process and slice window boundaries around target message
// 		result.Messages, result.HasMoreBefore, result.HasMoreAfter = processAroundResults(msgs, *cursorID, halfLimit)

// 	default:
// 		return nil, errs.InvalidArgument(fmt.Sprintf("unsupported list direction: %s", params.Direction))
// 	}

// 	return result, nil
// }

// // processAroundResults ensures chronological ordering, locates the target message, and evaluates overflow boundaries.
// func processAroundResults(msgs []Message, targetID uuid.UUID, halfLimit int32) ([]Message, bool, bool) {
// 	if len(msgs) == 0 {
// 		return msgs, false, false
// 	}

// 	// Edge Case D Guard: Guarantee chronological order (oldest -> newest) before slicing indexes
// 	if len(msgs) > 1 && msgs[0].CreatedAt().After(msgs[len(msgs)-1].CreatedAt()) {
// 		reverseMessages(msgs)
// 	} else {
// 		// Fallback stable sort if timestamp order is mixed or non-deterministic
// 		sort.SliceStable(msgs, func(i, j int) bool {
// 			if msgs[i].CreatedAt().Equal(msgs[j].CreatedAt()) {
// 				return msgs[i].ID().String() < msgs[j].ID().String()
// 			}
// 			return msgs[i].CreatedAt().Before(msgs[j].CreatedAt())
// 		})
// 	}

// 	targetIdx := -1
// 	for i, m := range msgs {
// 		if m.ID() == targetID {
// 			targetIdx = i
// 			break
// 		}
// 	}

// 	if targetIdx == -1 {
// 		return msgs, false, false
// 	}

// 	// Count elements older than target (before targetIdx)
// 	olderCount := int32(targetIdx)
// 	hasMoreBefore := olderCount > halfLimit

// 	// Count elements newer than target (after targetIdx)
// 	newerCount := int32(len(msgs) - 1 - targetIdx)
// 	hasMoreAfter := newerCount > halfLimit

// 	startIdx := 0
// 	if hasMoreBefore {
// 		startIdx = int(olderCount - halfLimit)
// 	}

// 	endIdx := len(msgs)
// 	if hasMoreAfter {
// 		endIdx = targetIdx + 1 + int(halfLimit)
// 	}

// 	return msgs[startIdx:endIdx], hasMoreBefore, hasMoreAfter
// }

// // reverseMessages inverts a slice of Message entities in place.
// func reverseMessages(msgs []Message) {
// 	for i, j := 0, len(msgs)-1; i < j; i, j = i+1, j-1 {
// 		msgs[i], msgs[j] = msgs[j], msgs[i]
// 	}
// }

// // GetFirstUnreadMessage retrieves the first unread message for a user in a channel.
// func (s *Service) GetFirstUnreadMessage(ctx context.Context, channelID, userID uuid.UUID) (*Message, error) {
// 	if channelID == uuid.Nil || userID == uuid.Nil {
// 		return nil, errs.InvalidArgument("channel ID and user ID cannot be nil")
// 	}

// 	ok, err := s.repo.IsMember(ctx, channelID, userID)
// 	if err != nil {
// 		return nil, err
// 	}
// 	if !ok {
// 		return nil, errs.PermissionDenied("you are not a member of this channel").Wrap(ErrNotParticipant)
// 	}

// 	msg, err := s.repo.MessageGetFirstUnread(ctx, channelID, userID)
// 	if err != nil {
// 		if errs.IsNotFound(err) {
// 			return nil, nil // No unread messages
// 		}
// 		return nil, err
// 	}

// 	return msg, nil
// }

// // AddReaction adds a reaction emoji to a message.
// func (s *Service) AddReaction(ctx context.Context, messageID, userID uuid.UUID, rawEmoji string) error {
// 	emojiVO, err := NewEmoji(rawEmoji)
// 	if err != nil {
// 		return errs.InvalidArgument(err.Error()).Wrap(err)
// 	}

// 	return s.tx.ExecTx(ctx, func(txCtx context.Context) error {
// 		msg, err := s.repo.MessageGet(txCtx, messageID)
// 		if err != nil {
// 			if errs.IsNotFound(err) {
// 				return errs.NotFound("message not found").Wrap(err)
// 			}
// 			return err
// 		}

// 		if _, err := s.repo.ReactionAdd(txCtx, messageID, userID, emojiVO.String()); err != nil {
// 			return err
// 		}

// 		_, err = s.outbox.Publish(txCtx, EventReactionAdded, ReactionPayload{
// 			MessageID: messageID,
// 			ChannelID: msg.ChannelID(),
// 			UserID:    userID,
// 			Emoji:     emojiVO.String(),
// 		})
// 		return err
// 	})
// }

// // RemoveReaction removes a user reaction from a message.
// func (s *Service) RemoveReaction(ctx context.Context, messageID, userID uuid.UUID, rawEmoji string) error {
// 	emojiVO, err := NewEmoji(rawEmoji)
// 	if err != nil {
// 		return errs.InvalidArgument(err.Error()).Wrap(err)
// 	}

// 	return s.tx.ExecTx(ctx, func(txCtx context.Context) error {
// 		msg, err := s.repo.MessageGet(txCtx, messageID)
// 		if err != nil {
// 			if errs.IsNotFound(err) {
// 				return errs.NotFound("message not found").Wrap(err)
// 			}
// 			return err
// 		}

// 		if err := s.repo.ReactionRemove(txCtx, messageID, userID, emojiVO); err != nil {
// 			return err
// 		}

// 		_, err = s.outbox.Publish(txCtx, EventReactionRemoved, ReactionPayload{
// 			MessageID: messageID,
// 			ChannelID: msg.ChannelID(),
// 			UserID:    userID,
// 			Emoji:     emojiVO.String(),
// 		})
// 		return err
// 	})
// }

// // MarkAsRead updates a member's unread position and clears their mention count.
// func (s *Service) MarkAsRead(ctx context.Context, channelID, userID, messageID uuid.UUID) error {
// 	ok, err := s.repo.IsMember(ctx, channelID, userID)
// 	if err != nil {
// 		return err
// 	}
// 	if !ok {
// 		return errs.PermissionDenied("you are not a member of this channel").Wrap(ErrNotParticipant)
// 	}

// 	now := time.Now().UTC()
// 	return s.repo.MemberUpdateReadState(ctx, channelID, userID, &messageID, now)
// }

// // SendTypingSignal records an active typing state in Redis.
// func (s *Service) SendTypingSignal(ctx context.Context, channelID, userID uuid.UUID) error {
// 	ok, err := s.repo.IsMember(ctx, channelID, userID)
// 	if err != nil {
// 		return err
// 	}
// 	if !ok {
// 		return errs.PermissionDenied("you are not a member of this channel").Wrap(ErrNotParticipant)
// 	}

// 	if err := s.typingStore.SetTyping(ctx, channelID, userID); err != nil {
// 		return errs.Internal("failed to set typing indicator").Wrap(err)
// 	}

// 	return nil
// }

func sortUUIDs(a, b uuid.UUID) (uuid.UUID, uuid.UUID) {
	if a.String() < b.String() {
		return a, b
	}
	return b, a
}
