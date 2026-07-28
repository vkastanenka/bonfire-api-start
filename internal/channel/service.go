package channel

import (
	"context"
	"errors"
	"fmt"
	"time"

	"bonfire-api/internal/channel"
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
	Delete(ctx context.Context, id uuid.UUID) error
	FindDM(ctx context.Context, user1ID, user2ID uuid.UUID) (*Channel, error)
	ListByUser(ctx context.Context, userID uuid.UUID) ([]db.ChannelListByUserRow, error)

	// ============================================================================
	// MEMBERS
	// ============================================================================
	AddMember(ctx context.Context, m *Member) error
	AddMembers(ctx context.Context, members []*channel.Member) error
	GetMember(ctx context.Context, channelID, userID uuid.UUID) (*Member, error)
	ListMembers(ctx context.Context, channelID uuid.UUID) ([]db.ChannelMemberListByChannelRow, error)
	UpdateMemberReadState(ctx context.Context, channelID, userID, messageID uuid.UUID) error
	IncrementMentionCount(ctx context.Context, channelID, userID uuid.UUID) error
	IncrementMentionCountBatch(ctx context.Context, channelID uuid.UUID, userIDs []uuid.UUID) error
	RemoveMember(ctx context.Context, channelID, userID uuid.UUID) error
	IsMember(ctx context.Context, channelID, userID uuid.UUID) (bool, error)

	// ============================================================================
	// MESSAGES
	// ============================================================================
	CreateMessage(ctx context.Context, msg *Message) error
	GetMessage(ctx context.Context, id uuid.UUID) (*Message, error)
	ListMessages(ctx context.Context, channelID uuid.UUID, cursorCreatedAt *time.Time, cursorID *uuid.UUID, limit int) ([]Message, error)
	ListPinnedMessages(ctx context.Context, channelID uuid.UUID) ([]Message, error)
	ListMessageReplies(ctx context.Context, replyToMessageID uuid.UUID) ([]Message, error)
	UpdateMessage(ctx context.Context, msg *Message) error
	SetPinnedMessage(ctx context.Context, id uuid.UUID, isPinned bool) error
	DeleteMessage(ctx context.Context, id uuid.UUID) error

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
func (s *Service) CreateChannel(ctx context.Context, creatorID uuid.UUID, memberIDs []uuid.UUID) (*channel.Channel, error) {
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

		_, err := s.outbox.Publish(txCtx, EventMessageCreated, MessageCreatedPayload{
			MessageID: msg.ID(),
			ChannelID: msg.ChannelID(),
			AuthorID:  msg.AuthorID(),
			Content:   msg.Content().String(), // Ensure this matches your payload type expectation
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
	contentVO, err := NewContent(newRawContent)
	if err != nil {
		return nil, errs.InvalidArgument(err.Error()).Wrap(err)
	}

	var updated *Message

	err = s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		msg, err := s.repo.GetMessage(txCtx, messageID)
		if err != nil {
			if errs.IsNotFound(err) {
				return errs.NotFound("message not found").Wrap(err)
			}
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
			Content:   msg.Content(),
		})

		updated = msg
		return err
	})

	if err != nil {
		return nil, err
	}

	return updated, nil
}

// DeleteMessage deletes a message if requested by its author.
func (s *Service) DeleteMessage(ctx context.Context, messageID, actorID uuid.UUID) error {
	return s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		msg, err := s.repo.GetMessage(txCtx, messageID)
		if err != nil {
			if errs.IsNotFound(err) {
				return errs.NotFound("message not found").Wrap(err)
			}
			return err
		}

		if msg.AuthorID() == nil || *msg.AuthorID() != actorID {
			return errs.PermissionDenied("cannot delete messages sent by another user").Wrap(ErrCannotEditOther)
		}

		if err := s.repo.DeleteMessage(txCtx, messageID); err != nil {
			return err
		}

		_, err = s.outbox.Publish(txCtx, EventMessageDeleted, MessageDeletedPayload{
			MessageID: msg.ID(),
			ChannelID: msg.ChannelID(),
		})
		return err
	})
}

// ListMessages retrieves paginated channel history.
func (s *Service) ListMessages(ctx context.Context, channelID, userID uuid.UUID, before *uuid.UUID, limit int) ([]Message, error) {
	ok, err := s.repo.IsParticipant(ctx, channelID, userID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, errs.PermissionDenied("you are not a member of this channel").Wrap(ErrNotParticipant)
	}

	if limit <= 0 || limit > 100 {
		limit = 50
	}

	messages, err := s.repo.ListMessages(ctx, channelID, before, limit)
	if err != nil {
		if errs.IsNotFound(err) {
			return []Message{}, nil
		}
		return nil, err
	}

	return messages, nil
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

		if err := s.repo.AddReaction(txCtx, messageID, userID, emojiVO.String()); err != nil {
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

		if err := s.repo.RemoveReaction(txCtx, messageID, userID, emojiVO.String()); err != nil {
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
	ok, err := s.repo.IsParticipant(ctx, channelID, userID)
	if err != nil {
		return err
	}
	if !ok {
		return errs.PermissionDenied("you are not a member of this channel").Wrap(ErrNotParticipant)
	}

	return s.repo.UpdateMemberReadState(ctx, channelID, userID, messageID)
}

// SendTypingSignal records an active typing state in Redis.
func (s *Service) SendTypingSignal(ctx context.Context, channelID, userID uuid.UUID) error {
	ok, err := s.repo.IsParticipant(ctx, channelID, userID)
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
