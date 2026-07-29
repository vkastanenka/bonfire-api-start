package channel

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"bonfire-api/internal/db"
	"bonfire-api/internal/errs"
	"bonfire-api/internal/outbox"

	"github.com/google/uuid"
)

const MaxGroupMembers = 10

var (
	ErrNotParticipant  = errors.New("user is not a participant of this channel")
	ErrCannotEditOther = errors.New("cannot edit or delete a message authored by someone else")
	ErrNotOwner        = errors.New("only the channel owner can perform this action")
)

type OutboxRepository interface {
	Publish(ctx context.Context, variant string, payload any) (*outbox.Event, error)
}

type Repository interface {
	// ============================================================================
	// CHANNELS
	// ============================================================================
	Create(ctx context.Context, ch *Channel) error
	Get(ctx context.Context, id uuid.UUID) (*Channel, error)
	Update(ctx context.Context, ch *Channel) error
	UpdateLastMessage(ctx context.Context, channelID, messageID uuid.UUID, updatedAt time.Time) error
	ClearLastMessage(ctx context.Context, channelID uuid.UUID) error
	Delete(ctx context.Context, id uuid.UUID) error
	FindDM(ctx context.Context, user1ID, user2ID uuid.UUID) (*Channel, error)
	ListByUser(ctx context.Context, userID uuid.UUID) ([]db.ChannelListByUserRow, error)

	// ============================================================================
	// MEMBERS
	// ============================================================================
	AddMembers(ctx context.Context, members []*Member) error
	GetMember(ctx context.Context, channelID, userID uuid.UUID) (*Member, error)
	ListMembers(ctx context.Context, channelID uuid.UUID) ([]db.ChannelMemberListByChannelRow, error)
	UpdateMemberReadState(ctx context.Context, channelID, userID uuid.UUID, messageID *uuid.UUID, lastReadAt time.Time) error
	IncrementMentionCount(ctx context.Context, channelID, userID uuid.UUID) error
	IncrementMentionCountBatch(ctx context.Context, channelID uuid.UUID, userIDs []uuid.UUID) error
	RemoveMember(ctx context.Context, channelID, userID uuid.UUID) error
	IsMember(ctx context.Context, channelID, userID uuid.UUID) (bool, error)
	GetUnreadCount(ctx context.Context, channelID, userID uuid.UUID) (int32, error)

	// ============================================================================
	// MESSAGES
	// ============================================================================
	CreateMessage(ctx context.Context, msg *Message) error
	GetMessage(ctx context.Context, id uuid.UUID) (*Message, error)
	GetLatestMessage(ctx context.Context, channelID uuid.UUID) (*Message, error)
	ListMessagesBefore(ctx context.Context, channelID uuid.UUID, cursorCreatedAt *time.Time, cursorID *uuid.UUID, limit int32) ([]Message, error)
	ListMessagesAfter(ctx context.Context, channelID uuid.UUID, cursorCreatedAt *time.Time, cursorID *uuid.UUID, limit int32) ([]Message, error)
	ListMessagesAround(ctx context.Context, channelID uuid.UUID, cursorCreatedAt time.Time, cursorID uuid.UUID, halfLimit int32) ([]Message, error)
	ListPinnedMessages(ctx context.Context, channelID uuid.UUID) ([]Message, error)
	ListMessageReplies(ctx context.Context, replyToMessageID uuid.UUID) ([]Message, error)
	GetFirstUnreadMessage(ctx context.Context, channelID, userID uuid.UUID) (*Message, error)
	UpdateMessage(ctx context.Context, msg *Message) error
	SetPinnedMessage(ctx context.Context, id uuid.UUID, isPinned bool) error
	DeleteMessage(ctx context.Context, id uuid.UUID) error
	HasMessagesAfter(ctx context.Context, channelID uuid.UUID, createdAt time.Time, id uuid.UUID) (bool, error)
	HasMessagesBefore(ctx context.Context, channelID uuid.UUID, createdAt time.Time, id uuid.UUID) (bool, error)

	// ============================================================================
	// REACTIONS
	// ============================================================================
	AddReaction(ctx context.Context, messageID, userID uuid.UUID, emoji string) (*Reaction, error)
	RemoveReaction(ctx context.Context, messageID, userID uuid.UUID, emoji Emoji) error
	ListReactionsByMessage(ctx context.Context, messageID uuid.UUID) ([]Reaction, error)
	SummarizeReactionsByMessage(ctx context.Context, messageID, currentUserID uuid.UUID) ([]ReactionSummary, error)
}

type TypingStore interface {
	SetTyping(ctx context.Context, channelID, userID uuid.UUID) error
}

type Tx interface {
	ExecTx(ctx context.Context, fn func(txCtx context.Context) error) error
}

type Service struct {
	repo        Repository
	outbox      OutboxRepository
	typingStore TypingStore
	tx          Tx
}

func NewService(repo Repository, outbox OutboxRepository, typingStore TypingStore, tx Tx) *Service {
	return &Service{
		repo:        repo,
		outbox:      outbox,
		typingStore: typingStore,
		tx:          tx,
	}
}

// CreateChannel creates a new DM or Group channel and adds participants atomically.
// If a DM already exists between two users, it returns the existing channel.
func (s *Service) CreateChannel(ctx context.Context, creatorID uuid.UUID, memberIDs []uuid.UUID) (*Channel, error) {
	if creatorID == uuid.Nil {
		return nil, errs.InvalidArgument("creator ID cannot be nil")
	}

	// 1. Consolidate unique member IDs (excluding nil UUIDs and deduplicating creator)
	memberMap := make(map[uuid.UUID]struct{}, len(memberIDs)+1)
	memberMap[creatorID] = struct{}{}

	for _, id := range memberIDs {
		if id != uuid.Nil {
			memberMap[id] = struct{}{}
		}
	}

	totalMembers := len(memberMap)

	// 2. Business Rule Guard: Maximum Member Limit
	if totalMembers > MaxGroupMembers {
		return nil, errs.InvalidArgument(fmt.Sprintf("group channels cannot exceed %d total members", MaxGroupMembers))
	}

	// 3. Handle DM Case (Exactly 2 participants)
	if totalMembers == 2 {
		var otherID uuid.UUID
		for id := range memberMap {
			if id != creatorID {
				otherID = id
				break
			}
		}

		existingDM, err := s.repo.FindDM(ctx, creatorID, otherID)
		if err == nil && existingDM != nil {
			return existingDM, nil
		}
		if err != nil && !errs.IsNotFound(err) {
			return nil, err
		}
	}

	// 4. Determine Channel Type & Owner
	var (
		chType  Type
		ownerID *uuid.UUID
	)

	if totalMembers == 2 {
		chType = TypeDirect
		ownerID = nil
	} else {
		chType = TypeGroup
		ownerID = &creatorID
	}

	// 5. Instantiate Domain Entities Outside the Transaction
	ch, err := New(chType, ownerID, nil, nil)
	if err != nil {
		return nil, errs.InvalidArgument(err.Error()).Wrap(err)
	}

	members := make([]*Member, 0, totalMembers)
	for memberID := range memberMap {
		m, err := NewMember(ch.ID(), memberID)
		if err != nil {
			return nil, errs.InvalidArgument(err.Error()).Wrap(err)
		}
		members = append(members, m)
	}

	// 6. Atomic Persistence Block (2 DB round-trips total)
	err = s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		if err := s.repo.Create(txCtx, ch); err != nil {
			return err
		}

		return s.repo.AddMembers(txCtx, members)
	})
	if err != nil {
		return nil, err
	}

	return ch, nil
}

// GetChannel fetches channel details after verifying the actor is a member.
func (s *Service) GetChannel(ctx context.Context, channelID, actorID uuid.UUID) (*Channel, error) {
	if channelID == uuid.Nil || actorID == uuid.Nil {
		return nil, errs.InvalidArgument("channel ID and actor ID cannot be nil")
	}

	ok, err := s.repo.IsMember(ctx, channelID, actorID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, errs.PermissionDenied("you are not a member of this channel").Wrap(ErrNotParticipant)
	}

	return s.repo.Get(ctx, channelID)
}

// PostMessage validates user membership, constructs content value object, persists, and publishes the event.
func (s *Service) PostMessage(ctx context.Context, channelID, authorID uuid.UUID, rawContent string, replyToID *uuid.UUID) (*Message, error) {
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
		parentMsg, err := s.repo.GetMessage(ctx, *replyToID)
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
		if err := s.repo.CreateMessage(txCtx, msg); err != nil {
			return err
		}

		if err := s.repo.UpdateLastMessage(txCtx, msg.ChannelID(), msg.ID(), msg.CreatedAt()); err != nil {
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

// EditMessageContent allows the author to update message content.
func (s *Service) EditMessageContent(ctx context.Context, messageID, authorID uuid.UUID, newRawContent string) (*Message, error) {
	if messageID == uuid.Nil || authorID == uuid.Nil {
		return nil, errs.InvalidArgument("message ID and author ID cannot be nil")
	}

	contentVO, err := NewContent(newRawContent)
	if err != nil {
		return nil, errs.InvalidArgument(err.Error()).Wrap(err)
	}

	var updated *Message

	err = s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		msg, err := s.repo.GetMessage(txCtx, messageID)
		if err != nil {
			return err
		}

		if msg.AuthorID() == nil || *msg.AuthorID() != authorID {
			return errs.PermissionDenied("cannot edit messages sent by another user").Wrap(ErrCannotEditOther)
		}

		msg.EditContent(contentVO)

		if err := s.repo.UpdateMessage(txCtx, msg); err != nil {
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

// SetMessagePinned handles pinning or unpinning a message for channel members.
func (s *Service) SetMessagePinned(ctx context.Context, messageID, userID uuid.UUID, isPinned bool) (*Message, error) {
	if messageID == uuid.Nil || userID == uuid.Nil {
		return nil, errs.InvalidArgument("message ID and user ID cannot be nil")
	}

	var updated *Message

	err := s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		msg, err := s.repo.GetMessage(txCtx, messageID)
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

		if err := s.repo.SetPinnedMessage(txCtx, msg.ID(), isPinned); err != nil {
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

// DeleteMessage deletes a message if requested by its author or an authorized user.
func (s *Service) DeleteMessage(ctx context.Context, messageID, actorID uuid.UUID) error {
	if messageID == uuid.Nil || actorID == uuid.Nil {
		return errs.InvalidArgument("message ID and actor ID cannot be nil")
	}

	return s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		msg, err := s.repo.GetMessage(txCtx, messageID)
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
		if err := s.repo.DeleteMessage(txCtx, messageID); err != nil {
			return err
		}

		// 3. If deleted message was the last_message_id, recalculate and update
		if ch.LastMessageID() != nil && *ch.LastMessageID() == messageID {
			// Get the most recent remaining message in the channel (returns nil if channel is now empty)
			latestMsg, err := s.repo.GetLatestMessage(txCtx, msg.ChannelID())
			if err != nil {
				return err
			}

			if latestMsg != nil {
				if err := s.repo.UpdateLastMessage(txCtx, msg.ChannelID(), latestMsg.ID(), latestMsg.CreatedAt()); err != nil {
					return err
				}
			} else {
				// Optional: Clear lastMessageID if no messages remain in the channel
				if err := s.repo.ClearLastMessage(txCtx, msg.ChannelID()); err != nil {
					return err
				}
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
	firstUnread, err := s.repo.GetFirstUnreadMessage(ctx, channelID, userID)
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
		targetMsg, err := s.repo.GetMessage(ctx, *params.TargetID)
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
		msgs, err := s.repo.ListMessagesBefore(ctx, params.ChannelID, cursorCreatedAt, cursorID, params.Limit+1)
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
		msgs, err := s.repo.ListMessagesAfter(ctx, params.ChannelID, cursorCreatedAt, cursorID, params.Limit+1)
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
		msgs, err := s.repo.ListMessagesAround(ctx, params.ChannelID, *cursorCreatedAt, *cursorID, halfLimit)
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

	msg, err := s.repo.GetFirstUnreadMessage(ctx, channelID, userID)
	if err != nil {
		if errs.IsNotFound(err) {
			return nil, nil // No unread messages
		}
		return nil, err
	}

	return msg, nil
}

var (
	ErrCannotModifyDM       = errors.New("cannot modify properties of a direct message channel")
	ErrInvalidIconURL       = errors.New("icon URL must be between 3 and 2048 characters")
	ErrCannotTransferToSelf = errors.New("cannot transfer ownership to current owner")
)

type UpdateChannelParams struct {
	ChannelID uuid.UUID
	ActorID   uuid.UUID
	Name      *string
	IconURL   *string
}

// Update encapsulates metadata changes requested by a user
func (s *Service) Update(ctx context.Context, input UpdateChannelParams) (*Channel, error) {
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

	// 2. Authorization: Only the owner can update group channel metadata
	if !ch.IsOwner(input.ActorID) {
		return nil, ErrNotGroupOwner
	}

	var updated bool

	// 3. Update Name (with true value equality check)
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

	// 4. Update Icon URL (with true value equality check)
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

	// 5. Early return if state did not change
	if !updated {
		return ch, nil
	}

	// 6. Persist entity changes and emit Outbox event in a single atomic transaction
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

// ListMembers returns all member records for a channel after confirming the requesting user is a participant.
func (s *Service) ListMembers(ctx context.Context, channelID, actorID uuid.UUID) ([]db.ChannelMemberListByChannelRow, error) {
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
	members, err := s.repo.ListMembers(ctx, channelID)
	if err != nil {
		return nil, err
	}

	return members, nil
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
		if err := s.repo.UpdateMemberReadState(txCtx, channelID, userID, messageID, readAt); err != nil {
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

// AddReaction adds a reaction emoji to a message.
func (s *Service) AddReaction(ctx context.Context, messageID, userID uuid.UUID, rawEmoji string) error {
	emojiVO, err := NewEmoji(rawEmoji)
	if err != nil {
		return errs.InvalidArgument(err.Error()).Wrap(err)
	}

	return s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		msg, err := s.repo.GetMessage(txCtx, messageID)
		if err != nil {
			if errs.IsNotFound(err) {
				return errs.NotFound("message not found").Wrap(err)
			}
			return err
		}

		if _, err := s.repo.AddReaction(txCtx, messageID, userID, emojiVO.String()); err != nil {
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
		msg, err := s.repo.GetMessage(txCtx, messageID)
		if err != nil {
			if errs.IsNotFound(err) {
				return errs.NotFound("message not found").Wrap(err)
			}
			return err
		}

		if err := s.repo.RemoveReaction(txCtx, messageID, userID, emojiVO); err != nil {
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
	return s.repo.UpdateMemberReadState(ctx, channelID, userID, &messageID, now)
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
