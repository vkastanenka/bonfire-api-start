package cache

import "bonfire-api/internal/fields"

func ChannelKey(id fields.ID) string {
	return "channel:" + id.String()
}

func ChannelMembersKey(channelID fields.ID) string {
	return "channel:" + channelID.String() + ":members"
}

func ChannelMessagesKey(channelID fields.ID) string {
	return "channel:" + channelID.String() + ":messages"
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


func UserMembersKey(userID fields.ID) string {
	return "user:" + userID.String() + ":members"
}
