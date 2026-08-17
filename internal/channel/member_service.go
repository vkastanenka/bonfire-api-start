package channel

import (
	"bonfire-api/internal/errs"
	"bonfire-api/internal/fields"
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type MemberRepository interface {
	CreateBatch(ctx context.Context, members []*Member) ([]*Member, error)
	Delete(ctx context.Context, channelID fields.ID, userID fields.ID) error
	Get(ctx context.Context, channelID fields.ID, userID fields.ID) (*Member, error)
	GetBatchByChannelIDs(ctx context.Context, channelIDs []fields.ID) (map[fields.ID][]*Member, error)
	IncrementBatchMentionCount(ctx context.Context, channelID fields.ID, userIDs []fields.ID, updatedAt fields.Timestamp) error
	ListVisibleByUserID(ctx context.Context, userID fields.ID, limit int32) ([]*Member, error)
	UpdateIsVisible(ctx context.Context, channelID fields.ID, userID fields.ID, isVisible bool, updatedAt fields.Timestamp) (*Member, error)
	UpdateLastReadMessage(ctx context.Context, channelID fields.ID, userID fields.ID, lastReadMessageID fields.ID, lastReadMessageAt fields.Timestamp, updatedAt fields.Timestamp, mentionCount *int32) (*Member, error)
	UpdateMutedUntil(ctx context.Context, channelID fields.ID, userID fields.ID, mutedUntil fields.Timestamp, updatedAt fields.Timestamp) (*Member, error)
	UpdatePinnedAt(ctx context.Context, channelID fields.ID, userID fields.ID, pinnedAt fields.Timestamp, updatedAt fields.Timestamp) (*Member, error)
}

type MemberCache interface {
	Delete(ctx context.Context, key fields.ID) error
	DeleteBatch(ctx context.Context, keys []fields.ID) error
	Get(ctx context.Context, id fields.ID) (*Member, error)
	GetBatch(ctx context.Context, ids []fields.ID) (map[fields.ID]*Member, []fields.ID, error)
	Set(ctx context.Context, ch *Member) error
	SetBatch(ctx context.Context, members []*Member) error
}

type MemberService struct {
	repo         MemberRepository
	cache        MemberCache
	channelRepo  ChannelRepository
	channelCache ChannelCache
	outboxRepo   OutboxRepository
	relationRepo RelationRepository
	tx           TX
}

func NewMemberService(
	repo MemberRepository,
	cache MemberCache,
	channelRepo ChannelRepository,
	channelCache ChannelCache,
	outboxRepo OutboxRepository,
	relationRepo RelationRepository,
	tx TX,
) *MemberService {
	return &MemberService{
		repo:         repo,
		cache:        cache,
		channelRepo:  channelRepo,
		channelCache: channelCache,
		outboxRepo:   outboxRepo,
		relationRepo: relationRepo,
		tx:           tx,
	}
}

func (s *MemberService) AddMembers(ctx context.Context, rawActorID, rawChannelID uuid.UUID, rawMemberIDs []uuid.UUID) error {
	if len(rawMemberIDs) == 0 {
		return errs.InvalidArgument("Member list cannot be empty.")
	}

	actorID, err := fields.ParseRequiredID("actor_id", rawActorID)
	if err != nil {
		return err
	}

	channelID, err := fields.ParseRequiredID("channel_id", rawChannelID)
	if err != nil {
		return err
	}

	memberIDs, err := fields.ParseIDs("member_id", rawMemberIDs)
	if err != nil {
		return err
	}

	// 1. Deduplicate new member IDs and filter out actorID
	newPeerIDs := make([]fields.ID, 0, len(memberIDs))
	seen := map[fields.ID]struct{}{
		actorID: {},
	}

	for _, id := range memberIDs {
		if _, exists := seen[id]; !exists {
			seen[id] = struct{}{}
			newPeerIDs = append(newPeerIDs, id)
		}
	}

	// If all provided IDs were duplicates of actorID or each other, return early
	if len(newPeerIDs) == 0 {
		return errs.InvalidArgument("No new members to add.")
	}

	// 2. Verify no block exists between actor and prospective new members
	hasBlock, err := s.relationRepo.HasIncomingBlock(ctx, actorID, newPeerIDs)
	if err != nil {
		return err
	}
	if hasBlock {
		return errs.InvalidArgument("Cannot add users who have blocked you.")
	}

	var newMembers []*Member
	now := fields.NewTimestamp(time.Now())

	// 3. Execute Transaction
	err = s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		// Lock channel for update to avoid race conditions on group size
		ch, err := s.channelRepo.GetForUpdate(txCtx, channelID)
		if err != nil {
			return err
		}
		if ch == nil {
			return errs.NotFound("Channel not found.")
		}

		// Fetch existing channel members within tx
		existingMembersMap, err := s.repo.GetBatchByChannelIDs(txCtx, []fields.ID{channelID})
		if err != nil {
			return err
		}

		existingMembers := existingMembersMap[channelID]

		// Ensure actor is currently a member of the target channel
		isActorMember := false
		existingMemberSet := make(map[fields.ID]struct{}, len(existingMembers))
		for _, m := range existingMembers {
			existingMemberSet[m.UserID()] = struct{}{}
			if m.UserID() == actorID {
				isActorMember = true
			}
		}

		if !isActorMember {
			return errs.PermissionDenied("You are not a member of this channel.")
		}

		// Filter out candidates who are already members
		toAddIDs := make([]fields.ID, 0, len(newPeerIDs))
		for _, id := range newPeerIDs {
			if _, exists := existingMemberSet[id]; !exists {
				toAddIDs = append(toAddIDs, id)
			}
		}

		if len(toAddIDs) == 0 {
			return errs.InvalidArgument("All specified users are already members of this channel.")
		}

		// Validate capacity limit: existing count + candidates to add <= max allowed peers + 1
		if len(existingMembers)+len(toAddIDs) > ChannelMaxPeers+1 {
			return errs.InvalidArgument(fmt.Sprintf("Adding these members exceeds the maximum limit of %d members.", ChannelMaxPeers+1))
		}

		targetChannelID := channelID
		finalMemberIDs := toAddIDs

		// If this is a 1:1 Direct DM (Type 1), spawn a new Group DM channel instead
		if ChannelTypeValue(ch.Type().Value) == ChannelTypeDirect {
			rawID, err := uuid.NewV7()
			if err != nil {
				return errs.Internal("Failed to generate channel ID.").Wrap(err)
			}

			newGroupChannelID := fields.NewID(rawID)

			// Construct new Group DM channel
			newGroupChannel := ParseChannel(
				newGroupChannelID,
				NewChannelType(ChannelTypeGroup),
				ChannelName{},
				fields.URL{},
				fields.ID{},
				fields.Timestamp{},
				now,
				now,
			)

			newCh, err := s.channelRepo.Create(txCtx, newGroupChannel)
			if err != nil {
				return err
			}

			targetChannelID = newCh.ID()

			// When converting to a new group channel, existing members of the DM
			// also need to be seeded into the new channel alongside the new candidates.
			finalMemberIDs = make([]fields.ID, 0, len(existingMembers)+len(toAddIDs))
			for _, m := range existingMembers {
				finalMemberIDs = append(finalMemberIDs, m.UserID())
			}
			finalMemberIDs = append(finalMemberIDs, toAddIDs...)
		}

		// Construct Domain DTOs using targetChannelID
		parsedMembers := make([]*Member, 0, len(finalMemberIDs))
		for _, id := range finalMemberIDs {
			m := ParseMember(
				targetChannelID,
				id,
				fields.ID{},
				fields.Timestamp{},
				fields.Timestamp{},
				fields.Timestamp{},
				1,
				true,
				now,
				now,
			)
			parsedMembers = append(parsedMembers, m)
		}

		// Batch create membership records
		createdMembers, err := s.repo.CreateBatch(txCtx, parsedMembers)
		if err != nil {
			return err
		}

		// Publish outbox event (If upgraded to group, notify with the new group channel ID)
		_, err = s.outboxRepo.Publish(txCtx, EventMembersAdded, MembersAddedPayload{})
		if err != nil {
			return err
		}

		newMembers = createdMembers
		return nil
	})
	if err != nil {
		return err
	}

	// 4. Update Cache post-commit
	s.cache.SetBatch(ctx, newMembers)

	return nil
}

func (s *MemberService) UpdateLastReadMessage(
	ctx context.Context,
	rawActorID,
	rawChannelID,
	rawLastReadMessageID uuid.UUID,
	rawLastReadAt time.Time,
) (*Member, error) {
	actorID, err := fields.ParseRequiredID("actor_id", rawActorID)
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
			actorID,
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
	rawActorID,
	rawChannelID uuid.UUID,
	rawPinnedAt *time.Time,
) (*Member, error) {
	actorID, err := fields.ParseRequiredID("actor_id", rawActorID)
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
			actorID,
			pinnedAt,
			now,
		)
		if err != nil {
			return err
		}

		_, err = s.outboxRepo.Publish(txCtx, EventMemberPinned, MemberUpdatePinnedAtPayload{})
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

// Update is visible

// Update muted until

// Delete
