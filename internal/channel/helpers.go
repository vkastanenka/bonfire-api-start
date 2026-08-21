package channel

import (
	"bonfire-api/internal/errs"
	"bonfire-api/internal/fields"
	"bonfire-api/internal/user"
	"fmt"
	"slices"
	"strings"
)

func FilterNewMemberIDs(userID fields.ID, existingMembers []*Member, newPeerIDs []fields.ID) ([]fields.ID, error) {
	existingMemberSet := make(map[fields.ID]struct{}, len(existingMembers))
	for _, m := range existingMembers {
		existingMemberSet[m.UserID()] = struct{}{}
	}

	cleanNewPeerIDs := fields.RemoveID(newPeerIDs, userID)
	toAddIDs := make([]fields.ID, 0, len(cleanNewPeerIDs))

	for _, id := range cleanNewPeerIDs {
		if _, exists := existingMemberSet[id]; !exists {
			toAddIDs = append(toAddIDs, id)
		}
	}

	if len(toAddIDs) == 0 {
		return nil, errs.InvalidArgument("All specified users are already members of this channel.").
			Reason("ALREADY_MEMBERS")
	}

	if len(existingMembers)+len(toAddIDs) > ChannelMaxPeers+1 {
		return nil, errs.InvalidArgument(fmt.Sprintf("Adding these members exceeds the maximum limit of %d members.", ChannelMaxPeers+1)).
			Reason("MAX_CAPACITY_EXCEEDED")
	}

	return toAddIDs, nil
}

func IndexMemberships(members []*Member) ([]fields.ID, map[fields.ID]*Member) {
	channelIDs := make([]fields.ID, len(members))
	membershipMap := make(map[fields.ID]*Member, len(members))
	for i, m := range members {
		chID := m.ChannelID()
		channelIDs[i] = chID
		membershipMap[chID] = m
	}
	return channelIDs, membershipMap
}

func SortMembers(members []*Member, userMap map[fields.ID]*user.User) {
	slices.SortFunc(members, func(a, b *Member) int {
		uA, okA := userMap[a.UserID()]
		uB, okB := userMap[b.UserID()]

		nameA := ""
		if okA && uA != nil {
			nameA = uA.DisplayName().String()
		}
		nameB := ""
		if okB && uB != nil {
			nameB = uB.DisplayName().String()
		}

		return strings.Compare(nameA, nameB)
	})
}

func SortMessages(messages []*Message) {
	slices.SortFunc(messages, func(a, b *Message) int {
		return a.ID().Compare(b.ID())
	})
}

func SortPinnedMessages(messages []*Message) {
	slices.SortFunc(messages, func(a, b *Message) int {
		return b.ID().Compare(a.ID())
	})
}

func SortSidebar(channels []*Channel, userMembersMap map[fields.ID]*Member) {
	slices.SortFunc(channels, func(a, b *Channel) int {
		mA := userMembersMap[a.ID()]
		mB := userMembersMap[b.ID()]

		// 1. Pinned priority
		aPinned := mA != nil && mA.PinnedAt().IsValid()
		bPinned := mB != nil && mB.PinnedAt().IsValid()
		if aPinned != bPinned {
			if aPinned {
				return -1
			}
			return 1
		}
		if aPinned {
			if mA.PinnedAt().After(mB.PinnedAt()) {
				return -1
			}
			if mB.PinnedAt().After(mA.PinnedAt()) {
				return 1
			}
		}

		// 2. Activity (lastMessageAt)
		aLast := a.LastMessageAt()
		bLast := b.LastMessageAt()
		if !aLast.Equals(bLast) {
			if aLast.After(bLast) {
				return -1
			}
			return 1
		}

		// 3. Creation date
		if a.CreatedAt().After(b.CreatedAt()) {
			return -1
		}
		if b.CreatedAt().After(a.CreatedAt()) {
			return 1
		}

		// 4. Guaranteed deterministic ID tie-breaker
		return a.ID().Compare(b.ID())
	})
}
