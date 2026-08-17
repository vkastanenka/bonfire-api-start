package channel

import (
	"context"
	"errors"
	"time"

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
	GetForMemberUpdate(ctx context.Context, channelID uuid.UUID, memberID uuid.UUID) (*Channel, error)
	HasMessagesAfter(ctx context.Context, channelID uuid.UUID, createdAt time.Time, id uuid.UUID) (bool, error)
	HasMessagesBefore(ctx context.Context, channelID uuid.UUID, createdAt time.Time, id uuid.UUID) (bool, error)
	IsMember(ctx context.Context, channelID uuid.UUID, userID uuid.UUID) (bool, error)
	MemberAddBatch(ctx context.Context, members []*Member) error
	MemberCloseDM(ctx context.Context, channelID uuid.UUID, userID uuid.UUID, updatedAt time.Time) error
	MemberCount(ctx context.Context, channelID uuid.UUID) (int32, error)
	MemberGet(ctx context.Context, channelID uuid.UUID, userID uuid.UUID) (*Member, error)
	MemberGetUnreadCount(ctx context.Context, channelID uuid.UUID, userID uuid.UUID) (int32, error)
	MemberIncrementMentionCountBatch(ctx context.Context, channelID uuid.UUID, userIDs []uuid.UUID) error
	MemberListByChannel(ctx context.Context, channelID uuid.UUID) ([]*Member, error)
	MemberListItemsByChannel(ctx context.Context, channelID uuid.UUID, cursorCreatedAt *time.Time, cursorUserID *uuid.UUID, limit int32) ([]*MemberListItem, error)
	MemberOpenDM(ctx context.Context, channelID uuid.UUID, userID uuid.UUID, updatedAt time.Time) error
	MemberRemove(ctx context.Context, channelID uuid.UUID, userID uuid.UUID) error
	MemberResetMentionCount(ctx context.Context, channelID uuid.UUID, userID uuid.UUID) error
	MemberTogglePinned(ctx context.Context, channelID uuid.UUID, userID uuid.UUID, pinnedAt time.Time) error
	MemberUpdateRead(ctx context.Context, channelID uuid.UUID, userID uuid.UUID, messageID *uuid.UUID, lastReadAt time.Time) error
	MessageCreate(ctx context.Context, msg *Message) (*Message, error)
	MessageDelete(ctx context.Context, id uuid.UUID) error
	MessageGet(ctx context.Context, messageID uuid.UUID) (*Message, error)
	MessageGetAggregate(ctx context.Context, id uuid.UUID, userID *uuid.UUID) (*MessageAggregate, error)
	MessageGetFirstUnread(ctx context.Context, channelID uuid.UUID, userID uuid.UUID) (*Message, error)
	MessageGetLatest(ctx context.Context, channelID uuid.UUID) (*Message, error)
	MessageListAggregateAfter(ctx context.Context, channelID uuid.UUID, cursorCreatedAt *time.Time, cursorID *uuid.UUID, userID *uuid.UUID, limit int32) ([]*MessageAggregate, error)
	MessageListAggregateAround(ctx context.Context, channelID uuid.UUID, targetID uuid.UUID, userID *uuid.UUID, olderLimit int32, newerLimit int32) ([]*MessageAggregate, error)
	MessageListAggregateBefore(ctx context.Context, channelID uuid.UUID, cursorCreatedAt *time.Time, cursorID *uuid.UUID, userID *uuid.UUID, limit int32) ([]*MessageAggregate, error)
	MessageListPinnedAggregate(ctx context.Context, channelID uuid.UUID, cursorCreatedAt *time.Time, cursorID *uuid.UUID, userID *uuid.UUID, limit int32) ([]*MessageAggregate, error)
	MessageTogglePinned(ctx context.Context, msg *Message) (*Message, error)
	MessageUpdateContent(ctx context.Context, msg *Message) (*Message, error)
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

type TX interface {
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

var (
	ErrCannotModifyDM       = errors.New("cannot modify properties of a direct message channel")
	ErrInvalidIconURL       = errors.New("icon URL must be between 3 and 2048 characters")
	ErrCannotTransferToSelf = errors.New("cannot transfer ownership to current owner")
)

// ============================================================================
// MESSAGES
// ============================================================================

// DeleteMessage deletes a message if requested by its author or an authorized user.
func (s *Service) DeleteMessage(ctx context.Context, rawMsgID, rawActorID uuid.UUID) error {
	msgID, err := NewID(rawMsgID)
	if err != nil {
		return errs.InvalidArgument("invalid message ID").Wrap(err)
	}

	actorID, err := NewID(rawActorID)
	if err != nil {
		return errs.InvalidArgument("invalid actor ID").Wrap(err)
	}

	return s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		msg, err := s.repo.MessageGet(txCtx, msgID.UUID())
		if err != nil {
			return err
		}

		// Authorization Guard: Check if actor is author
		isAuthor := msg.AuthorID() != nil && *msg.AuthorID() == UserID(actorID.UUID())
		if !isAuthor {
			return errs.PermissionDenied("cannot delete messages sent by another user").Wrap(ErrCannotEditOther)
		}

		// Fetch channel to check if this message is the channel's last_message_id
		ch, err := s.repo.Get(txCtx, msg.ChannelID().UUID())
		if err != nil {
			return err
		}

		// Persist deletion in database
		if err := s.repo.MessageDelete(txCtx, msgID.UUID()); err != nil {
			return err
		}

		// If deleted message was the last_message_id, recalculate and update
		if ch.LastMessageID() != nil && *ch.LastMessageID() == MessageID(msgID.UUID()) {
			latestMsg, err := s.repo.MessageGetLatest(txCtx, msg.ChannelID().UUID())
			if err != nil {
				return err
			}

			ch.SetLastMessage(latestMsg.ID())

			_, err = s.repo.UpdateLastMessage(txCtx, ch)
			if err != nil {
				return err
			}
		}

		// 4. Publish outbox event
		now := time.Now().UTC()
		_, err = s.outbox.Publish(txCtx, EventMessageDeleted, MessageDeletedPayload{
			MessageID: msg.ID().UUID(),
			ChannelID: msg.ChannelID().UUID(),
			ActorID:   actorID.UUID(),
			DeletedAt: now,
		})
		return err
	})
}

// EditMessage allows the author to update message content.
func (s *Service) EditMessageContent(ctx context.Context, rawMsgID, rawAuthorID uuid.UUID, newRawContent string) (*Message, error) {
	msgID, err := NewID(rawMsgID)
	if err != nil {
		return nil, errs.InvalidArgument("invalid message ID").Wrap(err)
	}

	authorID, err := NewID(rawAuthorID)
	if err != nil {
		return nil, errs.InvalidArgument("invalid author ID").Wrap(err)
	}

	contentVO, err := NewContent(newRawContent)
	if err != nil {
		return nil, errs.InvalidArgument(err.Error()).Wrap(err)
	}

	var updated *Message

	err = s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		msg1, err := s.repo.MessageGet(txCtx, msgID.UUID())
		if err != nil {
			return err
		}

		if msg1.AuthorID() == nil || *msg1.AuthorID() != UserID(authorID.UUID()) {
			return errs.PermissionDenied("cannot edit messages sent by another user").Wrap(ErrCannotEditOther)
		}

		msg1.EditContent(ptr.To(contentVO))

		msg2, err := s.repo.MessageUpdateContent(txCtx, msg1)
		if err != nil {
			return err
		}

		_, err = s.outbox.Publish(txCtx, EventMessageUpdated, MessageUpdatedPayload{
			MessageID: msg2.ID().UUID(),
			ChannelID: msg2.ChannelID().UUID(),
			AuthorID:  ptr.To(msg2.AuthorID().UUID()),
			Content:   msg2.Content().String(),
			EditedAt:  msg2.EditedAt(),
		})
		if err != nil {
			return err
		}

		updated = msg2
		return nil
	})
	if err != nil {
		return nil, err
	}

	return updated, nil
}

type ListPinnedMessagesParams struct {
	ChannelID uuid.UUID
	UserID    uuid.UUID
	BeforeID  *uuid.UUID
	Limit     int32
}

type ListPinnedMessagesResult struct {
	Messages []*MessageAggregate `json:"messages"`
	HasMore  bool                `json:"has_more"`
}

// ListPinnedMessages retrieves pinned messages for a channel ordered by newest first.
// If BeforeID is supplied, it fetches pins created prior to that message's timestamp.
func (s *Service) ListPinnedMessages(ctx context.Context, params ListPinnedMessagesParams) (*ListPinnedMessagesResult, error) {
	// 1. Input validation
	if params.ChannelID == uuid.Nil || params.UserID == uuid.Nil {
		return nil, errs.InvalidArgument("channel ID and user ID cannot be nil")
	}

	// Clamp pagination limit
	if params.Limit <= 0 {
		params.Limit = 50
	} else if params.Limit > 100 {
		params.Limit = 100
	}

	// 2. Authorization check
	ok, err := s.repo.IsMember(ctx, params.ChannelID, params.UserID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, errs.PermissionDenied("you are not a member of this channel").Wrap(ErrNotParticipant)
	}

	// 3. Resolve cursor message details if BeforeID is provided
	var cursorMsg *Message
	if params.BeforeID != nil && *params.BeforeID != uuid.Nil {
		var err error
		cursorMsg, err = s.repo.MessageGet(ctx, *params.BeforeID)
		if err != nil {
			return nil, err
		}
		if cursorMsg.ChannelID().UUID() != params.ChannelID {
			return nil, errs.InvalidArgument("target cursor message does not belong to specified channel")
		}
	}

	var (
		cursorCreatedAt *time.Time
		cursorID        *uuid.UUID
	)
	if cursorMsg != nil {
		cursorCreatedAt = cursorMsg.CreatedAtPtr()
		cursorID = cursorMsg.IDPtr()
	}

	// 4. Request Limit + 1 to evaluate the HasMore overflow flag
	msgs, err := s.repo.MessageListPinnedAggregate(
		ctx,
		params.ChannelID,
		cursorCreatedAt,
		cursorID,
		&params.UserID,
		params.Limit+1,
	)
	if err != nil {
		return nil, err
	}

	result := &ListPinnedMessagesResult{
		Messages: make([]*MessageAggregate, 0),
		HasMore:  false,
	}

	// 5. Trim overflow item and set HasMore flag
	if int32(len(msgs)) > params.Limit {
		result.HasMore = true
		msgs = msgs[:params.Limit]
	}

	if msgs != nil {
		result.Messages = msgs
	}

	return result, nil
}

// ToggleMessagePin handles pinning or unpinning a message for channel members.
func (s *Service) ToggleMessagePin(ctx context.Context, rawMsgID, rawActorID uuid.UUID) (*Message, error) {
	msgID, err := NewID(rawMsgID)
	if err != nil {
		return nil, errs.InvalidArgument("invalid message ID").Wrap(err)
	}

	actorID, err := NewID(rawActorID)
	if err != nil {
		return nil, errs.InvalidArgument("invalid actor ID").Wrap(err)
	}

	var updated *Message

	err = s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		msg, err := s.repo.MessageGet(txCtx, msgID.UUID())
		if err != nil {
			return err
		}

		// 1. Authorization Guard: Check channel membership for the pin action
		ok, err := s.repo.IsMember(txCtx, msg.ChannelID().UUID(), actorID.UUID())
		if err != nil {
			return err
		}
		if !ok {
			return errs.PermissionDenied("you are not a member of this channel").Wrap(ErrNotParticipant)
		}

		// 2. Mutate domain state & persist
		msg.SetPinned(!msg.IsPinned())

		msg1, err := s.repo.MessageTogglePinned(txCtx, msg)
		if err != nil {
			return err
		}

		// 3. Publish outbox event
		_, err = s.outbox.Publish(txCtx, EventMessagePinned, MessagePinnedPayload{
			MessageID: msg1.ID().UUID(),
			ChannelID: msg1.ChannelID().UUID(),
			IsPinned:  msg1.IsPinned(),
		})
		if err != nil {
			return err
		}

		updated = msg1
		return nil
	})
	if err != nil {
		return nil, err
	}

	return updated, nil
}

// ============================================================================
// REACTIONS
// ============================================================================

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
			ChannelID: msg.ChannelID().UUID(),
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
			ChannelID: msg.ChannelID().UUID(),
			UserID:    userID,
			Emoji:     emojiVO.String(),
		})
		return err
	})
}

// processAroundResults slices the returned window around targetID and flags overflow flags.
func processAroundResults(
	msgs []*MessageAggregate,
	targetID uuid.UUID,
	halfLimit int32,
) ([]*MessageAggregate, bool, bool) {
	if len(msgs) == 0 {
		return msgs, false, false
	}

	targetIdx := -1
	for i, m := range msgs {
		if m.Message() != nil && m.Message().ID().UUID() == targetID {
			targetIdx = i
			break
		}
	}

	if targetIdx == -1 {
		return msgs, false, false
	}

	olderCount := int32(targetIdx)
	hasMoreBefore := olderCount > halfLimit

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

func sortUUIDs(a, b uuid.UUID) (uuid.UUID, uuid.UUID) {
	if a.String() < b.String() {
		return a, b
	}
	return b, a
}
