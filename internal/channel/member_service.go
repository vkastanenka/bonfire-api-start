package channel

import (
	"bonfire-api/internal/errs"
	"bonfire-api/internal/fields"
	"context"
	"time"

	"github.com/google/uuid"
)

type MemberService struct {
	repo         MemberRepository
	channelRepo  ChannelRepository
	messageRepo  MessageRepository
	userRepo     UserRepository
	userCache    UserCache
	outboxRepo   OutboxRepository
	relationRepo RelationRepository
	tx           TX
}

func NewMemberService(
	repo MemberRepository,
	channelRepo ChannelRepository,
	messageRepo MessageRepository,
	userRepo UserRepository,
	userCache UserCache,
	outboxRepo OutboxRepository,
	relationRepo RelationRepository,
	tx TX,
) *MemberService {
	return &MemberService{
		repo:         repo,
		channelRepo:  channelRepo,
		messageRepo:  messageRepo,
		userRepo:     userRepo,
		userCache:    userCache,
		outboxRepo:   outboxRepo,
		relationRepo: relationRepo,
		tx:           tx,
	}
}

// AddMembers adds members to a channel and creates a group channel if adding to a direct channel.
func (s *MemberService) AddMembers(
	ctx context.Context,
	rawUserID,
	rawChannelID uuid.UUID,
	rawMemberIDs []uuid.UUID,
) error {
	// Validate
	if err := ValidateMinMembers(rawMemberIDs); err != nil {
		return err
	}

	userID, err := fields.ParseRequiredID("user_id", rawUserID)
	if err != nil {
		return err
	}

	channelID, err := fields.ParseRequiredID("channel_id", rawChannelID)
	if err != nil {
		return err
	}

	memberIDs, err := fields.ParseIDs("member_ids", rawMemberIDs)
	if err != nil {
		return err
	}

	// Dedupe and remove user id
	newPeerIDs := fields.RemoveID(fields.DedupeIDs(memberIDs), userID)
	if len(newPeerIDs) == 0 {
		return errs.InvalidArgument("No new members to add.").
			Reason("NO_NEW_MEMBERS")
	}

	// Verify blocks
	hasBlock, err := s.relationRepo.HasIncomingBlock(ctx, userID, newPeerIDs)
	if err != nil {
		return err
	}
	if hasBlock {
		return errs.InvalidArgument("Cannot add users who have blocked you.").
			Reason("INCOMING_BLOCK_DETECTED")
	}

	// Validate membership and filter new members
	existingMembersMap, err := s.repo.GetBatchByChannelIDs(ctx, []fields.ID{channelID})
	if err != nil {
		return err
	}

	existingMembers, ok := existingMembersMap[channelID]
	if !ok || len(existingMembers) == 0 {
		return ErrMembersNotFound()
	}

	if _, err := ValidateMembership(userID, existingMembers); err != nil {
		return err
	}

	newMemberIDs, err := FilterNewMemberIDs(userID, existingMembers, newPeerIDs)
	if err != nil {
		return err
	}

	now := fields.Now()

	return s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		// Lock channel
		chLock, err := s.channelRepo.GetForUpdate(txCtx, channelID)
		if err != nil {
			return err
		}

		var (
			targetChannelID = channelID
			membersToInsert []*Member
		)

		// If direct, create new group channel
		if chLock.chType.IsDirect() {
			newGroupChannelID, err := fields.NewID()
			if err != nil {
				return err
			}

			newGroupChannel := ParseChannel(
				newGroupChannelID,
				NewChannelTypeGroup(),
				ChannelName{},
				fields.URL{},
				fields.ID{},
				fields.Timestamp{},
				now,
				now,
			)

			createdChannel, err := s.channelRepo.Create(txCtx, newGroupChannel)
			if err != nil {
				return err
			}
			targetChannelID = createdChannel.ID()

			// Actor member has no mentions
			creatorMember := ParseMember(
				targetChannelID, userID, fields.ID{}, fields.Timestamp{},
				fields.Timestamp{}, fields.Timestamp{}, 0, true, now, now,
			)

			// Peer members have 1 mention
			allPeerIDs := make([]fields.ID, 0, (len(existingMembers)-1)+len(newMemberIDs))
			for _, m := range existingMembers {
				if !m.UserID().Equals(userID) {
					allPeerIDs = append(allPeerIDs, m.UserID())
				}
			}
			allPeerIDs = append(allPeerIDs, newMemberIDs...)

			peerMembers := ParseMembers(
				targetChannelID, allPeerIDs, fields.ID{}, fields.Timestamp{},
				fields.Timestamp{}, fields.Timestamp{}, 1, true, now, now,
			)

			membersToInsert = make([]*Member, 0, len(allPeerIDs)+1)
			membersToInsert = append(membersToInsert, creatorMember)
			membersToInsert = append(membersToInsert, peerMembers...)

		} else {
			// Existing group creates new members
			membersToInsert = ParseMembers(
				targetChannelID, newMemberIDs, fields.ID{}, fields.Timestamp{},
				fields.Timestamp{}, fields.Timestamp{}, 1, true, now, now,
			)

			// System messages only for existing groups
			systemMessages := make([]*Message, 0, len(newMemberIDs))
			msgTime := now

			for _, addedUserID := range newMemberIDs {
				msgID, err := fields.NewID()
				if err != nil {
					return err
				}

				msg := ParseMessageMemberAdd(
					msgID, targetChannelID, userID, addedUserID, msgTime,
				)
				systemMessages = append(systemMessages, msg)
				msgTime = msgTime.Add(time.Microsecond)
			}

			if len(systemMessages) > 0 {
				if _, err = s.messageRepo.CreateBatch(txCtx, systemMessages); err != nil {
					return err
				}
			}
		}

		// Create members
		if _, err := s.repo.CreateBatch(txCtx, membersToInsert); err != nil {
			return err
		}

		// Publish Outbox Event
		_, err = s.outboxRepo.Publish(txCtx, EventMembersAdded, MembersAddedPayload{})
		return err
	})
}

// CloseDirect updates the visibility of a channel membership to false.
func (s *MemberService) CloseDirect(
	ctx context.Context,
	rawUserID,
	rawChannelID uuid.UUID,
) error {
	// Validate
	userID, err := fields.ParseRequiredID("user_id", rawUserID)
	if err != nil {
		return err
	}

	channelID, err := fields.ParseRequiredID("channel_id", rawChannelID)
	if err != nil {
		return err
	}

	// Validate channel type
	ch, err := s.channelRepo.Get(ctx, channelID)
	if err != nil {
		return err
	}

	if !ch.Type().IsDirect() {
		return errs.InvalidArgument("Only direct channels can be closed or hidden.").
			Reason("INVALID_CHANNEL_TYPE")
	}

	now := fields.Now()

	return s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		// Update visibility
		_, err := s.repo.UpdateIsVisible(
			txCtx,
			channelID,
			userID,
			false,
			now,
		)
		if err != nil {
			return err
		}

		// Publish event
		_, err = s.outboxRepo.Publish(
			txCtx,
			EventMemberUpdateUpdateVisibility,
			MemberUpdateVisibilityPayload{},
		)
		if err != nil {
			return err
		}

		return nil
	})
}

// UpdateLastReadMessage updates a members last read message id and time.
func (s *MemberService) UpdateLastReadMessage(
	ctx context.Context,
	rawUserID,
	rawChannelID,
	rawLastReadMessageID uuid.UUID,
) (*Member, error) {
	// Validate
	userID, err := fields.ParseRequiredID("user_id", rawUserID)
	if err != nil {
		return nil, err
	}

	channelID, err := fields.ParseRequiredID("channel_id", rawChannelID)
	if err != nil {
		return nil, err
	}

	lastReadMessageID, err := fields.ParseRequiredID("last_read_message_id", rawLastReadMessageID)
	if err != nil {
		return nil, err
	}

	// Get channel for mention count handling
	ch, err := s.channelRepo.Get(ctx, channelID)
	if err != nil {
		return nil, err
	}

	var mentionCount *int32

	if ch.LastMessageID().Equals(lastReadMessageID) {
		zero := int32(0)
		mentionCount = &zero
	}

	var updatedMember *Member

	now := fields.Now()

	err = s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		// Update member
		updatedMember, err = s.repo.UpdateLastReadMessage(
			txCtx,
			channelID,
			userID,
			lastReadMessageID,
			now,
			now,
			mentionCount,
		)
		if err != nil {
			return err
		}

		// Publish event
		_, err = s.outboxRepo.Publish(
			txCtx,
			EventMemberUpdateLastReadMessage,
			MemberUpdateLastReadMessagePayload{},
		)
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return updatedMember, nil
}

// UpdatePinnedAt updates a members pinned at timestamp.
func (s *MemberService) UpdatePinnedAt(
	ctx context.Context,
	rawUserID,
	rawChannelID uuid.UUID,
) (*Member, error) {
	// Validate
	userID, err := fields.ParseRequiredID("user_id", rawUserID)
	if err != nil {
		return nil, err
	}

	channelID, err := fields.ParseRequiredID("channel_id", rawChannelID)
	if err != nil {
		return nil, err
	}

	var updatedMember *Member

	now := fields.Now()

	err = s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		// Update member
		updatedMember, err = s.repo.UpdatePinnedAt(
			txCtx,
			channelID,
			userID,
			now,
			now,
		)
		if err != nil {
			return err
		}

		// Publish event
		_, err = s.outboxRepo.Publish(
			txCtx,
			EventMemberUpdatePinnedAt,
			MemberUpdatePinnedAtPayload{},
		)
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return updatedMember, nil
}

// UpdateMuted until updates a members muted until timestamp.
func (s *MemberService) UpdateMutedUntil(
	ctx context.Context,
	rawUserID,
	rawChannelID uuid.UUID,
) (*Member, error) {
	// Validate
	userID, err := fields.ParseRequiredID("user_id", rawUserID)
	if err != nil {
		return nil, err
	}

	channelID, err := fields.ParseRequiredID("channel_id", rawChannelID)
	if err != nil {
		return nil, err
	}

	var updatedMember *Member

	now := fields.Now()

	err = s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		// Update member
		updatedMember, err = s.repo.UpdateMutedUntil(
			txCtx,
			channelID,
			userID,
			now,
			now,
		)
		if err != nil {
			return err
		}

		// Publish event
		_, err = s.outboxRepo.Publish(
			txCtx,
			EventMemberUpdateMutedUntil,
			MemberUpdateMutedUntilPayload{},
		)
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return updatedMember, nil
}

// LeaveGroup deletes a member and a group channel if no remaining members.
func (s *MemberService) LeaveGroup(
	ctx context.Context,
	rawUserID,
	rawChannelID uuid.UUID,
) error {
	// Validate
	userID, err := fields.ParseRequiredID("user_id", rawUserID)
	if err != nil {
		return err
	}

	channelID, err := fields.ParseRequiredID("channel_id", rawChannelID)
	if err != nil {
		return err
	}

	now := fields.Now()

	return s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		// Lock channel
		ch, err := s.channelRepo.GetForUpdate(txCtx, channelID)
		if err != nil {
			return err
		}

		// Validate channel type
		if ch.Type().IsDirect() {
			return errs.InvalidArgument("Cannot leave a direct message channel.")
		}

		// Delete member
		err = s.repo.Delete(txCtx, channelID, userID)
		if err != nil {
			return err
		}

		// Count remaining members
		remainingCount, err := s.repo.CountByChannel(txCtx, channelID)
		if err != nil {
			return err
		}

		// Delete and return if no members left
		if remainingCount == 0 {
			err = s.channelRepo.Delete(txCtx, channelID)
			if err != nil {
				return err
			}
			return nil
		}

		// Create system message
		msgID, err := fields.NewID()
		if err != nil {
			return err
		}

		sysMsg := ParseMessageMemberRemove(
			msgID,
			ch.id,
			userID,
			now,
		)

		_, err = s.messageRepo.Create(txCtx, sysMsg)
		if err != nil {
			return err
		}

		// Publish event
		_, err = s.outboxRepo.Publish(
			txCtx,
			EventMemberDelete,
			MemberDeletePayload{},
		)
		if err != nil {
			return err
		}

		return nil
	})
}
