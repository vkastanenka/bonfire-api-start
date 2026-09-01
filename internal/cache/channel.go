package cache

import "bonfire-api/internal/fields"

const (
	channelDomainKey = "channel:"
)

func channelActiveUsersKey(channelID fields.ID) string {
	return "{" + channelDomainKey + channelID.String() + "}:active-users"
}
