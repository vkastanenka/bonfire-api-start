package channel

import (
	"bonfire-api/internal/errs"
	"bonfire-api/internal/fields"
	"bonfire-api/internal/pkg/ptr"
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type ChannelRepository interface {
	Create(ctx context.Context, ch *Channel) (*Channel, error)
	Delete(ctx context.Context, id fields.ID) error
	Get(ctx context.Context, id fields.ID) (*Channel, error)
	GetBatch(ctx context.Context, ids []fields.ID) ([]*Channel, error)
	UpdateGroup(ctx context.Context, id fields.ID, name ChannelName, iconURL fields.URL, updatedAt fields.Timestamp) (*Channel, error)
	UpdateLastMessage(ctx context.Context, id fields.ID, lastMessageID fields.ID, lastMessageAt fields.Timestamp, updatedAt fields.Timestamp) (*Channel, error)
}

type ChannelCache interface {
	Delete(ctx context.Context, key fields.ID) error
	DeleteBatch(ctx context.Context, keys []fields.ID) error
	Get(ctx context.Context, id fields.ID) (*Channel, error)
	GetBatch(ctx context.Context, ids []fields.ID) (map[fields.ID]*Channel, []fields.ID, error)
	Set(ctx context.Context, ch *Channel) error
	SetBatch(ctx context.Context, channels []*Channel) error
}

type RelationRepository interface {
	HasIncomingBlock(ctx context.Context, actorID fields.ID, peerIDs []fields.ID) (bool, error)
}

type ChannelService struct {
	repo         ChannelRepository
	cache        ChannelCache
	memberRepo   MemberRepository
	memberCache  MemberCache
	outboxRepo   OutboxRepository
	relationRepo RelationRepository
	tx           TX
}

func NewChannelService(
	repo ChannelRepository,
	cache ChannelCache,
	memberRepo MemberRepository,
	memberCache MemberCache,
	outboxRepo OutboxRepository,
	relationRepo RelationRepository,
	tx TX,
) *ChannelService {
	return &ChannelService{
		repo:         repo,
		cache:        cache,
		memberRepo:   memberRepo,
		memberCache:  memberCache,
		outboxRepo:   outboxRepo,
		relationRepo: relationRepo,
		tx:           tx,
	}
}

func (s *ChannelService) CreateGroup(ctx context.Context, rawUserID uuid.UUID, rawMemberIDs []uuid.UUID) error {
	if len(rawMemberIDs) > ChannelMaxPeers {
		return errs.InvalidArgument(fmt.Sprintf("Member list cannot exceed %d items.", ChannelMaxPeers))
	}

	userID, err := fields.ParseRequiredID("user_id", rawUserID)
	if err != nil {
		return err
	}

	memberIDs, err := fields.ParseIDs("user_id", rawMemberIDs)
	if err != nil {
		return err
	}

	peerIDs := make([]fields.ID, 0, len(memberIDs))
	seen := map[fields.ID]struct{}{
		userID: {},
	}

	for _, id := range memberIDs {
		if _, exists := seen[id]; !exists {
			seen[id] = struct{}{}
			peerIDs = append(peerIDs, id)
		}
	}

	if len(peerIDs) > 0 {
		hasBlock, err := s.relationRepo.HasIncomingBlock(ctx, userID, peerIDs)
		if err != nil {
			return err
		}
		if hasBlock {
			return errs.InvalidArgument("Cannot create group DM with users who have blocked you.")
		}
	}

	allMemberIDs := make([]fields.ID, 0, len(peerIDs)+1)
	allMemberIDs = append(allMemberIDs, userID)
	allMemberIDs = append(allMemberIDs, peerIDs...)

	now := fields.NewTimestamp(time.Now())
	channelID := fields.NewID(uuid.New())

	parsedChannel := ParseChannel(
		channelID,
		NewChannelType(ChannelTypeGroup),
		ChannelName{},
		fields.URL{},
		fields.ID{},
		fields.Timestamp{},
		now,
		now,
	)

	parsedMembers := make([]*Member, 0, len(allMemberIDs))
	for _, id := range allMemberIDs {
		m := ParseMember(
			channelID,
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

	var newChannel *Channel
	var newChannelMembers []*Member

	err = s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		channelRow, err := s.repo.Create(txCtx, parsedChannel)
		if err != nil {
			return err
		}

		memberRows, err := s.memberRepo.CreateBatch(txCtx, parsedMembers)
		if err != nil {
			return err
		}

		_, err = s.outboxRepo.Publish(txCtx, EventChannelCreated, ChannelCreated{})
		if err != nil {
			return err
		}

		newChannel = channelRow
		newChannelMembers = memberRows
		return nil
	})
	if err != nil {
		return err
	}

	s.cache.Set(ctx, newChannel)
	s.memberCache.SetBatch(ctx, newChannelMembers)

	// Set channelID: memberID[]?

	return nil
}

// Get

// Get User Sidebar

func (s *ChannelService) UpdateGroup(ctx context.Context, rawUserID, rawChannelID uuid.UUID, rawName, rawIconURL *string) (*Channel, error) {
	if rawName == nil && rawIconURL == nil {
		return nil, errs.InvalidArgument("No fields provided for update.")
	}

	userID, err := fields.ParseRequiredID("user_id", rawUserID)
	if err != nil {
		return nil, err
	}

	channelID, err := fields.ParseRequiredID("channel_id", rawChannelID)
	if err != nil {
		return nil, err
	}

	name, err := ParseChannelName(ptr.From(rawName))
	if err != nil {
		return nil, err
	}

	iconURL, err := fields.ParseURL("icon_url", ptr.From(rawIconURL))
	if err != nil {
		return nil, err
	}

	_, err = s.memberRepo.Get(ctx, channelID, userID)
	if err != nil {
		if errs.IsNotFound(err) {
			return nil, errs.PermissionDenied("You are not a member of this channel.")
		}
		return nil, err
	}

	updatedAt := fields.NewTimestamp(time.Now())
	var updatedChannel *Channel

	err = s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		channelRow, err := s.repo.UpdateGroup(txCtx, channelID, name, iconURL, updatedAt)
		if err != nil {
			return err
		}

		_, err = s.outboxRepo.Publish(txCtx, EventChannelUpdated, ChannelUpdatedPayload{})
		if err != nil {
			return err
		}

		updatedChannel = channelRow
		return nil
	})
	if err != nil {
		return nil, err
	}

	s.cache.Delete(ctx, updatedChannel.ID())

	return updatedChannel, nil
}
