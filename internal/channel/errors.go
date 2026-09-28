package channel

import (
	"fmt"

	"bonfire-api/internal/pkg/errs"
)

func ErrCannotAddNonFriendUserToGroup() *errs.Error {
	return errs.InvalidArgument("Cannot add non-friend user to group.").
		Reason("NON_FRIEND_USER").
		FieldViolation("peerIds", "Cannot add users who are not in your friends list.", "NOT_FRIENDS").
		Meta("domain", "channels")
}

func ErrUserMaxChannelsReached() *errs.Error {
	return errs.FailedPrecondition("You have reached the maximum number of channels.").
		Reason("MAX_CHANNELS_REACHED").
		Meta("domain", "channels")
}

func ErrPeerMaxChannelsReached() *errs.Error {
	return errs.FailedPrecondition("One or more users have reached their maximum channel limit.").
		Reason("PEER_MAX_CHANNELS_REACHED").
		FieldViolation("peerIds", "User cannot be added because they reached their channel limit.", "MAX_CHANNELS_REACHED").
		Meta("domain", "channels")
}

func ErrAlreadyMembers() *errs.Error {
	return errs.InvalidArgument("All specified users are already members of this channel.").
		Reason("ALREADY_MEMBERS").
		Meta("domain", "channels")
}

func ErrCannotLeaveDirectChannel() *errs.Error {
	return errs.InvalidArgument("Cannot leave a direct message channel.").
		Reason("INVALID_CHANNEL_TYPE").
		Meta("domain", "channels")
}

func ErrCannotAddMembersToDirectChannel() *errs.Error {
	return errs.InvalidArgument("Cannot add members to direct channel.")
}

func ErrChannelNameRequired() *errs.Error {
	return errs.InvalidArgument("Channel name is required.").
		Reason("NAME_REQUIRED").
		FieldViolation("name", "Field is required.", "REQUIRED").
		Meta("domain", "channels")
}

func ErrChannelNameTooLong() *errs.Error {
	return errs.InvalidArgument("Name too long.").
		Reason("NAME_TOO_LONG").
		FieldViolation("name", "Name must be 100 characters or fewer.", "MAX_LENGTH_EXCEEDED").
		Meta("domain", "channels")
}

func ErrChannelTypeInvalid() *errs.Error {
	return errs.InvalidArgument("Invalid channel type.").
		Reason("CHANNEL_TYPE_INVALID").
		FieldViolation("type", "Must be one of: DIRECT, GROUP.", "INVALID_ENUM_VALUE").
		Meta("domain", "channels")
}

func ErrMaxCapacityExceeded() *errs.Error {
	return errs.InvalidArgument(fmt.Sprintf("Adding these members exceeds the maximum limit of %d members.", ChannelMaxMembers)).
		Reason("MAX_CAPACITY_EXCEEDED").
		Meta("domain", "channels")
}

func ErrMaxPeersExceeded() *errs.Error {
	return errs.InvalidArgument(fmt.Sprintf("Peer list cannot exceed %d items.", ChannelMaxPeers)).
		Reason("MAX_PEERS_EXCEEDED").
		FieldViolation("peer_ids", fmt.Sprintf("List cannot exceed %d items.", ChannelMaxPeers), "MAX_LENGTH_EXCEEDED").
		Meta("domain", "channels")
}

func ErrMembersNotFound() *errs.Error {
	return errs.NotFound("Channel members not found.")
}

func ErrMinMembersInvalid() *errs.Error {
	return errs.InvalidArgument(fmt.Sprintf("Member list must be at least %d items.", ChannelMinMembers)).
		Reason("MIN_MEMBERS_INVALID").
		FieldViolation("member_ids", fmt.Sprintf("List must be at least %d items.", ChannelMinMembers), "MAX_LENGTH_EXCEEDED").
		Meta("domain", "channels")
}

func ErrNoNewMembers() *errs.Error {
	return errs.InvalidArgument("No new members to add.").
		Reason("NO_NEW_MEMBERS").
		Meta("domain", "channels")
}

func ErrNotChannelMember() *errs.Error {
	return errs.PermissionDenied("You are not a member of this channel.").
		Reason("NOT_A_MEMBER").
		Meta("domain", "channels")
}

func ErrOnlyDirectChannelsSupported() *errs.Error {
	return errs.InvalidArgument("Only direct channels can be closed or hidden.").
		Reason("INVALID_CHANNEL_TYPE").
		Meta("domain", "channels")
}

func ErrMuteDurationInvalid() *errs.Error {
	return errs.InvalidArgument("Invalid mute duration.").
		Reason("MUTE_DURATION_INVALID").
		FieldViolation("mute_duration", "Must be one of: 15_MIN, 1_HOUR, 8_HOURS, 24_HOURS, 3_DAYS, FOREVER.", "INVALID_ENUM_VALUE").
		Meta("domain", "members")
}

func ErrMessageContentMinLength() *errs.Error {
	return errs.InvalidArgument("Content must have at least 1 character.").
		Reason("CONTENT_TOO_SHORT").
		FieldViolation("content", "Content must have at least 1 character.", "MIN_LENGTH_EXCEEDED").
		Meta("domain", "messages")
}

func ErrMessageContentRequired() *errs.Error {
	return errs.InvalidArgument("Message content is required.").
		Reason("CONTENT_REQUIRED").
		FieldViolation("content", "Field is required.", "REQUIRED").
		Meta("domain", "messages")
}

func ErrMessageContentTooLong() *errs.Error {
	return errs.InvalidArgument("Content too long.").
		Reason("CONTENT_TOO_LONG").
		FieldViolation("content", "Content must be 4000 characters or fewer.", "MAX_LENGTH_EXCEEDED").
		Meta("domain", "messages")
}

func ErrMessageForwardIncomplete() *errs.Error {
	return errs.InvalidArgument("Forwarded message ID and forwarded channel ID must be provided together.").
		Reason("FORWARD_IDS_INCOMPLETE").
		Meta("domain", "messages")
}

func ErrMessageNotAuthorizedToDelete() *errs.Error {
	return errs.PermissionDenied("Actor is not authorized to delete this message.").
		Reason("NOT_AUTHORIZED_TO_DELETE").
		Meta("domain", "messages")
}

func ErrMessageNotAuthor() *errs.Error {
	return errs.PermissionDenied("Actor is not the author of the message.").
		Reason("NOT_MESSAGE_AUTHOR").
		Meta("domain", "messages")
}

func ErrMessageNotFoundInChannel() *errs.Error {
	return errs.NotFound("Message not found in this channel.").
		Reason("MESSAGE_NOT_IN_CHANNEL").
		Meta("domain", "messages")
}

func ErrMessageReplyConflict() *errs.Error {
	return errs.InvalidArgument("Cannot reply to a message and forward a message at the same time.").
		Reason("REPLY_FORWARD_MUTUALLY_EXCLUSIVE").
		Meta("domain", "messages")
}

func ErrMessageReplyDifferentChannel() *errs.Error {
	return errs.InvalidArgument("Cannot reply to a message in a different channel.").
		Reason("REPLY_DIFFERENT_CHANNEL").
		Meta("domain", "messages")
}

func ErrMessageTypeInvalid() *errs.Error {
	return errs.InvalidArgument("Invalid message type.").
		Reason("MESSAGE_TYPE_INVALID").
		FieldViolation("type", "Must be a valid message type.", "INVALID_ENUM_VALUE").
		Meta("domain", "messages")
}

func ErrReactionEmojiRequired() *errs.Error {
	return errs.InvalidArgument("Emoji is required.").
		Reason("EMOJI_REQUIRED").
		FieldViolation("emoji", "Emoji cannot be empty.", "REQUIRED").
		Meta("domain", "reactions")
}

func ErrReactionEmojiTooLong() *errs.Error {
	return errs.InvalidArgument("Emoji too long.").
		Reason("EMOJI_TOO_LONG").
		FieldViolation("emoji", "Emoji must be 64 characters or fewer.", "MAX_LENGTH_EXCEEDED").
		Meta("domain", "reactions")
}
