package channel

import (
	"bonfire-api/internal/fields"

	"github.com/google/uuid"
)

func validateMaxPeers(rawPeerIDs []uuid.UUID) error {
	if len(rawPeerIDs) > ChannelMaxPeers {
		return ErrMaxPeersExceeded(ChannelMaxPeers)
	}
	return nil
}

func validateMinMembers(rawMemberIDs []uuid.UUID) error {
	if len(rawMemberIDs) < ChannelMinMembers {
		return ErrMinMembersInvalid(ChannelMinMembers)
	}
	return nil
}

func validateMembership(userID fields.ID, members []*Member) (*Member, error) {
	for _, m := range members {
		if m != nil && m.UserID() == userID {
			return m, nil
		}
	}
	return nil, ErrNotChannelMember()
}
