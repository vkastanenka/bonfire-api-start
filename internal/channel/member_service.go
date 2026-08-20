package channel

import (
	"bonfire-api/internal/errs"
	"bonfire-api/internal/fields"
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
)

type MemberService struct {
	repo         MemberRepository
	cache        MemberCache
	channelRepo  ChannelRepository
	channelCache ChannelCache
	messageRepo  MessageRepository
	messageCache MessageCache
	userRepo     UserRepository
	userCache    UserCache
	outboxRepo   OutboxRepository
	relationRepo RelationRepository
	tx           TX
}

func NewMemberService(
	repo MemberRepository,
	cache MemberCache,
	channelRepo ChannelRepository,
	channelCache ChannelCache,
	messageRepo MessageRepository,
	messageCache MessageCache,
	userRepo UserRepository,
	userCache UserCache,
	outboxRepo OutboxRepository,
	relationRepo RelationRepository,
	tx TX,
) *MemberService {
	return &MemberService{
		repo:         repo,
		cache:        cache,
		channelRepo:  channelRepo,
		channelCache: channelCache,
		messageRepo:  messageRepo,
		messageCache: messageCache,
		userRepo:     userRepo,
		userCache:    userCache,
		outboxRepo:   outboxRepo,
		relationRepo: relationRepo,
		tx:           tx,
	}
}

func (s *MemberService) AddMembers(
	ctx context.Context,
	rawUserID,
	rawChannelID uuid.UUID,
	rawMemberIDs []uuid.UUID,
) error {
	err := ValidateMinMembers(rawMemberIDs)
	if err != nil {
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

	newPeerIDs := fields.RemoveID(fields.DedupeIDs(memberIDs), userID)
	if len(newPeerIDs) == 0 {
		return errs.InvalidArgument("No new members to add.").
			Reason("NO_NEW_MEMBERS")
	}

	hasBlock, err := s.relationRepo.HasIncomingBlock(ctx, userID, newPeerIDs)
	if err != nil {
		return err
	}
	if hasBlock {
		return errs.InvalidArgument("Cannot add users who have blocked you.").
			Reason("INCOMING_BLOCK_DETECTED")
	}

	now := fields.Now()

	var (
		createdChannel *Channel
		newMembers     []*Member
		systemMessages []*Message
		allMemberIDs   []fields.ID
	)

	err = s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		chLock, err := s.channelRepo.GetForUpdate(txCtx, channelID)
		if err != nil {
			return err
		}

		existingMembersMap, err := s.repo.GetBatchByChannelIDs(txCtx, []fields.ID{channelID})
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

		targetChannelID := channelID
		peerIDs := newMemberIDs

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

			ch, err := s.channelRepo.Create(txCtx, newGroupChannel)
			if err != nil {
				return err
			}

			createdChannel = ch
			targetChannelID = ch.ID()

			peerIDs = make([]fields.ID, 0, (len(existingMembers)-1)+len(newMemberIDs))
			for _, m := range existingMembers {
				if !m.UserID().Equals(userID) {
					peerIDs = append(peerIDs, m.UserID())
				}
			}
			peerIDs = append(peerIDs, newMemberIDs...)
		}

		creatorMember := ParseMember(
			targetChannelID,
			userID,
			fields.ID{},
			fields.Timestamp{},
			fields.Timestamp{},
			fields.Timestamp{},
			0,
			true,
			now,
			now,
		)

		peerMembers := ParseMembers(
			targetChannelID,
			peerIDs,
			fields.ID{},
			fields.Timestamp{},
			fields.Timestamp{},
			fields.Timestamp{},
			1,
			true,
			now,
			now,
		)

		parsedMembers := make([]*Member, 0, len(peerIDs)+1)
		parsedMembers = append(parsedMembers, creatorMember)
		parsedMembers = append(parsedMembers, peerMembers...)

		createdMembers, err := s.repo.CreateBatch(txCtx, parsedMembers)
		if err != nil {
			return err
		}

		if chLock.chType.IsGroup() {
			systemMessages = make([]*Message, 0, len(newMemberIDs))
			msgTime := now

			for _, addedUserID := range newMemberIDs {
				msgID, err := fields.NewID()
				if err != nil {
					return err
				}

				msg := ParseMessageMemberAdd(
					msgID,
					targetChannelID,
					userID,
					addedUserID,
					msgTime,
				)
				systemMessages = append(systemMessages, msg)

				msgTime = msgTime.Add(time.Microsecond)
			}

			if len(systemMessages) > 0 {
				systemMessages, err = s.messageRepo.CreateBatch(txCtx, systemMessages)
				if err != nil {
					return err
				}
			}
		}

		_, err = s.outboxRepo.Publish(txCtx, EventMembersAdded, MembersAddedPayload{})
		if err != nil {
			return err
		}

		newMembers = createdMembers
		allMemberIDs = make([]fields.ID, len(parsedMembers))
		for i, m := range parsedMembers {
			allMemberIDs[i] = m.UserID()
		}

		return nil
	})
	if err != nil {
		return err
	}

	// Cache Layer Implementation (Pipeline + Atomic Ordering)

	// Handle cache
	cacheCtx := context.WithoutCancel(ctx)

	if createdChannel != nil {
		if err := s.channelCache.Set(cacheCtx, createdChannel); err != nil {
			slog.WarnContext(cacheCtx, "failed to cache new channel entity",
				"channel_id", createdChannel.ID().String(),
				"error", err,
			)
		}

		if err := s.channelCache.SetMemberIDs(cacheCtx, createdChannel.ID(), allMemberIDs); err != nil {
			slog.WarnContext(cacheCtx, "failed to cache channel member ids",
				"channel_id", createdChannel.ID().String(),
				"count", len(allMemberIDs),
				"error", err,
			)
		}

		if err := s.channelCache.SetLoaded(cacheCtx, createdChannel.ID()); err != nil {
			slog.WarnContext(cacheCtx, "failed to set channel loaded flag",
				"channel_id", createdChannel.ID().String(),
				"error", err,
			)
		}

		// if err := s.userCache.DeleteChannelIDsBatch(cacheCtx, allMemberIDs); err != nil {
		// 	slog.WarnContext(cacheCtx, "failed to invalidate user channel ids batch",
		// 		"count", len(allMemberIDs),
		// 		"error", err,
		// 	)
		// }
	} else {
		if err := s.channelCache.DeleteMemberIDs(cacheCtx, channelID); err != nil {
			slog.WarnContext(cacheCtx, "failed to invalidate channel member ids",
				"channel_id", channelID.String(),
				"error", err,
			)
		}

		// if err := s.userCache.DeleteChannelIDsBatch(cacheCtx, allMemberIDs); err != nil {
		// 	slog.WarnContext(cacheCtx, "failed to invalidate user channel ids batch",
		// 		"count", len(allMemberIDs),
		// 		"error", err,
		// 	)
		// }
	}

	if err := s.cache.SetBatch(cacheCtx, newMembers); err != nil {
		slog.WarnContext(cacheCtx, "failed to batch cache members",
			"count", len(newMembers),
			"error", err,
		)
	}

	if len(systemMessages) > 0 {
		if err := s.messageCache.SetBatch(cacheCtx, systemMessages); err != nil {
			slog.WarnContext(cacheCtx, "failed to batch cache system messages",
				"channel_id", channelID.String(),
				"count", len(systemMessages),
				"error", err,
			)
		}
	}

	return nil
}

func (s *MemberService) CloseDirect(
	ctx context.Context,
	rawUserID,
	rawChannelID uuid.UUID,
) error {
	userID, err := fields.ParseRequiredID("user_id", rawUserID)
	if err != nil {
		return err
	}

	channelID, err := fields.ParseRequiredID("channel_id", rawChannelID)
	if err != nil {
		return err
	}

	ch, err := s.channelRepo.Get(ctx, channelID)
	if err != nil {
		return err
	}

	if !ch.Type().IsDirect() {
		return errs.InvalidArgument("Only direct channels can be closed or hidden.").
			Reason("INVALID_CHANNEL_TYPE")
	}

	now := fields.Now()

	var member *Member

	err = s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		member, err = s.repo.UpdateIsVisible(
			txCtx,
			channelID,
			userID,
			false,
			now,
		)
		if err != nil {
			return err
		}
		if member == nil {
			return errs.NotFound("Member not found in channel.")
		}

		_, err = s.outboxRepo.Publish(
			txCtx,
			EventMemberUpdateUpdateVisibility,
			MemberUpdateVisibilitytPayload{},
		)
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return err
	}

	// Handle cache
	cacheCtx := context.WithoutCancel(ctx)

	if err := s.cache.Delete(cacheCtx, member.channelID, member.userID); err != nil {
		slog.WarnContext(cacheCtx, "failed to delete cached member entity",
			"channel_id", member.channelID.String(),
			"user_id", member.userID.String(),
			"error", err,
		)
	}

	if err := s.userCache.RemoveChannelID(cacheCtx, userID, channelID); err != nil {
		slog.WarnContext(cacheCtx, "failed to remove channel id from user cache",
			"channel_id", channelID.String(),
			"user_id", userID.String(),
			"error", err,
		)
	}

	return nil
}

func (s *MemberService) UpdateLastReadMessage(
	ctx context.Context,
	rawUserID,
	rawChannelID,
	rawLastReadMessageID uuid.UUID,
	rawLastReadAt time.Time, // TODO: Remove, handle in service
) (*Member, error) {
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
	lastReadMessageAt := fields.NewTimestamp(rawLastReadAt)
	now := fields.NewTimestamp(time.Now())

	err = s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		member, err := s.repo.UpdateLastReadMessage(
			txCtx,
			channelID,
			userID,
			lastReadMessageID,
			lastReadMessageAt,
			now,
			mentionCount,
		)
		if err != nil {
			return err
		}
		if member == nil {
			return errs.NotFound("Member not found in channel.")
		}

		_, err = s.outboxRepo.Publish(txCtx, EventMemberUpdateLastReadMessage, MemberUpdateLastReadMessagePayload{})
		if err != nil {
			return err
		}

		updatedMember = member
		return nil
	})
	if err != nil {
		return nil, err
	}

	return updatedMember, nil
}

func (s *MemberService) UpdatePinnedAt(
	ctx context.Context,
	rawUserID,
	rawChannelID uuid.UUID,
	rawPinnedAt *time.Time,
) (*Member, error) {
	userID, err := fields.ParseRequiredID("user_id", rawUserID)
	if err != nil {
		return nil, err
	}

	channelID, err := fields.ParseRequiredID("channel_id", rawChannelID)
	if err != nil {
		return nil, err
	}

	var pinnedAt fields.Timestamp
	if rawPinnedAt != nil {
		pinnedAt = fields.NewTimestamp(*rawPinnedAt)
	}

	now := fields.NewTimestamp(time.Now())
	var updatedMember *Member

	err = s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		member, err := s.repo.UpdatePinnedAt(
			txCtx,
			channelID,
			userID,
			pinnedAt,
			now,
		)
		if err != nil {
			return err
		}

		_, err = s.outboxRepo.Publish(
			txCtx,
			EventMemberUpdatePinnedAt,
			MemberUpdatePinnedAtPayload{},
		)
		if err != nil {
			return err
		}

		updatedMember = member
		return nil
	})
	if err != nil {
		return nil, err
	}

	return updatedMember, nil
}

func (s *MemberService) UpdateGroupMutedUntil(
	ctx context.Context,
	rawUserID,
	rawChannelID uuid.UUID,
	rawMutedUntil *time.Time,
) (*Member, error) {
	userID, err := fields.ParseRequiredID("user_id", rawUserID)
	if err != nil {
		return nil, err
	}

	channelID, err := fields.ParseRequiredID("channel_id", rawChannelID)
	if err != nil {
		return nil, err
	}

	var mutedUntil fields.Timestamp
	if rawMutedUntil != nil {
		mutedUntil = fields.NewTimestamp(*rawMutedUntil)
	}

	now := fields.NewTimestamp(time.Now())
	var updatedMember *Member

	// TODO: Ensure channel type is type 2

	err = s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		member, err := s.repo.UpdateMutedUntil(
			txCtx,
			channelID,
			userID,
			mutedUntil,
			now,
		)
		if err != nil {
			return err
		}
		if member == nil {
			return errs.NotFound("Member not found in channel.")
		}

		_, err = s.outboxRepo.Publish(
			txCtx,
			EventMemberUpdateMutedUntil,
			MemberUpdateMutedUntilPayload{},
		)
		if err != nil {
			return err
		}

		updatedMember = member
		return nil
	})
	if err != nil {
		return nil, err
	}

	return updatedMember, nil
}

func (s *MemberService) LeaveGroup(
	ctx context.Context,
	rawUserID,
	rawChannelID uuid.UUID,
) error {
	userID, err := fields.ParseRequiredID("user_id", rawUserID)
	if err != nil {
		return err
	}

	channelID, err := fields.ParseRequiredID("channel_id", rawChannelID)
	if err != nil {
		return err
	}

	err = s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		ch, err := s.channelRepo.GetForUpdate(txCtx, channelID)
		if err != nil {
			return err
		}

		if ch.Type() == NewChannelType(ChannelTypeDirect) {
			return errs.InvalidArgument("Cannot leave a direct message channel.")
		}

		err = s.repo.Delete(txCtx, channelID, userID)
		if err != nil {
			return err
		}

		remainingCount, err := s.repo.CountByChannel(txCtx, channelID)
		if err != nil {
			return err
		}

		if remainingCount == 0 {
			err = s.channelRepo.Delete(txCtx, channelID)
			if err != nil {
				return err
			}
			// Optional: Publish channel deleted event
		} else {
			_, err = s.outboxRepo.Publish(
				txCtx,
				EventMemberDelete,
				MemberDeletePayload{},
			)
			if err != nil {
				return err
			}
		}

		// TODO: Send system message that user left

		return nil
	})
	if err != nil {
		return err
	}

	return nil
}
