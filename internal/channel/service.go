package channel

import (
	"context"
	"errors"

	"bonfire-api/internal/errs"
	"bonfire-api/internal/outbox"

	"github.com/google/uuid"
)

var (
	ErrNotParticipant  = errors.New("user is not a participant of this channel")
	ErrCannotEditOther = errors.New("cannot edit or delete a message authored by someone else")
	ErrNotOwner        = errors.New("only the channel owner can perform this action")
)

type OutboxRepository interface {
	Publish(ctx context.Context, variant string, payload any) (*outbox.Event, error)
}

type Repository interface {
	// Channel operations
	CreateChannel(ctx context.Context, ch *Channel) error
	GetChannel(ctx context.Context, id uuid.UUID) (*Channel, error)
	UpdateChannel(ctx context.Context, ch *Channel) error
	FindDirectMessage(ctx context.Context, user1ID, user2ID uuid.UUID) (*Channel, error)
	IsParticipant(ctx context.Context, channelID, userID uuid.UUID) (bool, error)

	// Member operations
	AddMember(ctx context.Context, member *Member) error
	GetMember(ctx context.Context, channelID, userID uuid.UUID) (*Member, error)
	UpdateMemberReadState(ctx context.Context, channelID, userID, messageID uuid.UUID) error

	// Message operations
	CreateMessage(ctx context.Context, msg *Message) error
	GetMessage(ctx context.Context, id uuid.UUID) (*Message, error)
	UpdateMessage(ctx context.Context, msg *Message) error
	DeleteMessage(ctx context.Context, id uuid.UUID) error
	ListMessages(ctx context.Context, channelID uuid.UUID, before *uuid.UUID, limit int) ([]Message, error)
	ListPinnedMessages(ctx context.Context, channelID uuid.UUID) ([]Message, error)

	// Reaction operations
	AddReaction(ctx context.Context, messageID, userID uuid.UUID, emoji string) error
	RemoveReaction(ctx context.Context, messageID, userID uuid.UUID, emoji string) error
	GetReactionSummaries(ctx context.Context, messageID uuid.UUID) ([]ReactionSummary, error)
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
func (s *Service) CreateChannel(ctx context.Context, chType Type, creatorID uuid.UUID, recipientIDs []uuid.UUID, name *string, iconURL *string) (*Channel, error) {
	var parsedName *Name
	if name != nil {
		n, err := NewName(*name)
		if err != nil {
			return nil, errs.InvalidArgument(err.Error()).Wrap(err)
		}
		parsedName = &n
	}

	var ownerID *uuid.UUID
	if chType == TypeGroup {
		ownerID = &creatorID
	}

	ch, err := New(chType, ownerID, parsedName, iconURL)
	if err != nil {
		return nil, errs.InvalidArgument(err.Error()).Wrap(err)
	}

	// Consolidate unique member IDs
	memberMap := make(map[uuid.UUID]struct{})
	memberMap[creatorID] = struct{}{}
	for _, id := range recipientIDs {
		if id != uuid.Nil {
			memberMap[id] = struct{}{}
		}
	}

	err = s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		// 1. Create the base channel row
		if err := s.repo.CreateChannel(txCtx, ch); err != nil {
			return err
		}

		// 2. Add members individually
		for memberID := range memberMap {
			member, err := NewMember(ch.ID(), memberID)
			if err != nil {
				return err
			}

			if err := s.repo.AddMember(txCtx, member); err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return ch, nil
}

// PostMessage validates user membership, constructs content value object, persists, and publishes the event.
func (s *Service) PostMessage(ctx context.Context, channelID, authorID uuid.UUID, rawContent string, replyToID *uuid.UUID) (*Message, error) {
	ok, err := s.repo.IsParticipant(ctx, channelID, authorID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, errs.PermissionDenied("you are not a member of this channel").Wrap(ErrNotParticipant)
	}

	contentVO, err := NewContent(rawContent)
	if err != nil {
		return nil, errs.InvalidArgument(err.Error()).Wrap(err)
	}

	msg, err := NewMessage(channelID, &authorID, replyToID, contentVO)
	if err != nil {
		return nil, errs.InvalidArgument(err.Error()).Wrap(err)
	}

	err = s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		if err := s.repo.CreateMessage(txCtx, msg); err != nil {
			return err
		}

		_, err := s.outbox.Publish(txCtx, EventMessageCreated, MessageCreatedPayload{
			MessageID: msg.ID(),
			ChannelID: msg.ChannelID(),
			AuthorID:  msg.AuthorID(),
			Content:   msg.Content(),
			ReplyToID: msg.ReplyToMessageID(),
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
