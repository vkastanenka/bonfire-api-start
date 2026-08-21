package channel

import (
	"bonfire-api/internal/errs"
	"bonfire-api/internal/fields"
	"fmt"

	"github.com/google/uuid"
)

func ValidateMaxPeers(rawPeerIDs []uuid.UUID) error {
	count := (len(rawPeerIDs))
	if count > ChannelMaxPeers {
		return errs.InvalidArgument(fmt.Sprintf("Peer list cannot exceed %d items.", ChannelMaxPeers)).
			Reason("MAX_PEERS_EXCEEDED")
	}
	return nil
}

func ValidateMinMembers(rawMemberIDs []uuid.UUID) error {
	count := (len(rawMemberIDs))
	if count < ChannelMinMembers {
		return errs.InvalidArgument(fmt.Sprintf("Member list must be at least %d items.", ChannelMinMembers)).
			Reason("MIN_MEMBERS_INVALID")
	}
	return nil
}

func ValidateMembership(userID fields.ID, members []*Member) (*Member, error) {
	for _, m := range members {
		if m.UserID() == userID {
			return m, nil
		}
	}
	return nil, errs.PermissionDenied("You are not a member of this channel.")
}
