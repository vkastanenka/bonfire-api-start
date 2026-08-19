package cache

import "bonfire-api/internal/fields"

func ChannelKey(id fields.ID) string {
	return "channel:" + id.String()
}

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

func MemberKey(k MemberKeyIDs) string {
	return "member:" + k.ChannelID.String() + ":" + k.UserID.String()
}

func MessageKey(id fields.ID) string {
	return "message:" + id.String()
}

func UserChannelIDsKey(userID fields.ID) string {
	return "user:" + userID.String() + ":channel_ids"
}
