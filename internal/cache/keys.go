package cache

import (
	"bonfire-api/internal/fields"
)

const (
	MaxBatchSize = 500
)

func ChannelLoadedKey(channelID fields.ID) string {
	return "channel:" + channelID.String() + ":loaded"
}

func ChannelMemberIDsKey(channelID fields.ID) string {
	return "channel:" + channelID.String() + ":member_ids"
}

func ChannelMessageIDsKey(channelID fields.ID) string {
	return "channel:" + channelID.String() + ":message_ids"
}

type MemberKeyIDs struct {
	ChannelID fields.ID
	UserID    fields.ID
}

func MessageKey(msgID fields.ID) string {
	return "message:" + msgID.String()
}

func MessageReactionsKey(msgID fields.ID) string {
	return "message:" + msgID.String() + ":reactions"
}
