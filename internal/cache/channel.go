package cache

import "bonfire-api/internal/fields"

const (
	channelDomainKey = "channel:"
)

func channelMembersKey(channelID fields.ID) string {
	return "{channel:" + channelID.String() + "}:members"
}

//
