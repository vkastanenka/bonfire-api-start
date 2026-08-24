package channel

import (
	"bonfire-api/internal/errs"
	"fmt"
)

// -----------------------------------------------------------------------------
// channels
// -----------------------------------------------------------------------------

func ErrAlreadyMembers() error {
	return errs.InvalidArgument("All specified users are already members of this channel.").
		Reason("ALREADY_MEMBERS").
		Meta("domain", "channels")
}

func ErrCannotLeaveDirectChannel() error {
	return errs.InvalidArgument("Cannot leave a direct message channel.").
		Reason("INVALID_CHANNEL_TYPE").
		Meta("domain", "channels")
}

func ErrChannelNameRequired() error {
	return errs.InvalidArgument("Channel name is required.").
		Reason("NAME_REQUIRED").
		FieldViolation("name", "Field is required.", "REQUIRED").
		Meta("domain", "channels")
}

func ErrChannelNameTooLong() error {
	return errs.InvalidArgument("Name too long.").
		Reason("NAME_TOO_LONG").
		FieldViolation("name", "Name must be 100 characters or fewer.", "MAX_LENGTH_EXCEEDED").
		Meta("domain", "channels")
}

func ErrChannelTypeInvalid() error {
	return errs.InvalidArgument("Invalid channel type.").
		Reason("CHANNEL_TYPE_INVALID").
		FieldViolation("type", "Must be one of: DIRECT, GROUP.", "INVALID_ENUM_VALUE").
		Meta("domain", "channels")
}

func ErrMaxCapacityExceeded() error {
	return errs.InvalidArgument(fmt.Sprintf("Adding these members exceeds the maximum limit of %d members.", ChannelMaxMembers)).
		Reason("MAX_CAPACITY_EXCEEDED").
		Meta("domain", "channels")
}

func ErrMaxPeersExceeded() error {
	return errs.InvalidArgument(fmt.Sprintf("Peer list cannot exceed %d items.", ChannelMaxPeers)).
		Reason("MAX_PEERS_EXCEEDED").
		FieldViolation("peer_ids", fmt.Sprintf("List cannot exceed %d items.", ChannelMaxPeers), "MAX_LENGTH_EXCEEDED").
		Meta("domain", "channels")
}

func ErrMembersNotFound() error {
	return errs.NotFound("Channel members not found.")
}

func ErrMinMembersInvalid() error {
	return errs.InvalidArgument(fmt.Sprintf("Member list must be at least %d items.", ChannelMinMembers)).
		Reason("MIN_MEMBERS_INVALID").
		FieldViolation("member_ids", fmt.Sprintf("List must be at least %d items.", ChannelMinMembers), "MAX_LENGTH_EXCEEDED").
		Meta("domain", "channels")
}

func ErrNotChannelMember() error {
	return errs.PermissionDenied("You are not a member of this channel.").
		Reason("NOT_A_MEMBER").
		Meta("domain", "channels")
}

func ErrOnlyDirectChannelsSupported() error {
	return errs.InvalidArgument("Only direct channels can be closed or hidden.").
		Reason("INVALID_CHANNEL_TYPE").
		Meta("domain", "channels")
}

// -----------------------------------------------------------------------------
// members
// -----------------------------------------------------------------------------

func ErrMuteDurationInvalid() error {
	return errs.InvalidArgument("Invalid mute duration.").
		Reason("MUTE_DURATION_INVALID").
		FieldViolation("mute_duration", "Must be one of: 15_MIN, 1_HOUR, 8_HOURS, 24_HOURS, 3_DAYS, FOREVER.", "INVALID_ENUM_VALUE").
		Meta("domain", "members")
}

// -----------------------------------------------------------------------------
// messages
// -----------------------------------------------------------------------------

func ErrMessageContentMinLength() error {
	return errs.InvalidArgument("Content must have at least 1 character.").
		Reason("CONTENT_TOO_SHORT").
		FieldViolation("content", "Content must have at least 1 character.", "MIN_LENGTH_EXCEEDED").
		Meta("domain", "messages")
}

func ErrMessageContentRequired() error {
	return errs.InvalidArgument("Message content is required.").
		Reason("CONTENT_REQUIRED").
		FieldViolation("content", "Field is required.", "REQUIRED").
		Meta("domain", "messages")
}

func ErrMessageContentTooLong() error {
	return errs.InvalidArgument("Content too long.").
		Reason("CONTENT_TOO_LONG").
		FieldViolation("content", "Content must be 4000 characters or fewer.", "MAX_LENGTH_EXCEEDED").
		Meta("domain", "messages")
}

func ErrMessageForwardIncomplete() error {
	return errs.InvalidArgument("Forwarded message ID and forwarded channel ID must be provided together.").
		Reason("FORWARD_IDS_INCOMPLETE").
		Meta("domain", "messages")
}

func ErrMessageNotAuthorizedToDelete() error {
	return errs.PermissionDenied("Actor is not authorized to delete this message.").
		Reason("NOT_AUTHORIZED_TO_DELETE").
		Meta("domain", "messages")
}

func ErrMessageNotAuthor() error {
	return errs.PermissionDenied("Actor is not the author of the message.").
		Reason("NOT_MESSAGE_AUTHOR").
		Meta("domain", "messages")
}

func ErrMessageNotFoundInChannel() error {
	return errs.NotFound("Message not found in this channel.").
		Reason("MESSAGE_NOT_IN_CHANNEL").
		Meta("domain", "messages")
}

func ErrMessageReplyConflict() error {
	return errs.InvalidArgument("Cannot reply to a message and forward a message at the same time.").
		Reason("REPLY_FORWARD_MUTUALLY_EXCLUSIVE").
		Meta("domain", "messages")
}

func ErrMessageReplyDifferentChannel() error {
	return errs.InvalidArgument("Cannot reply to a message in a different channel.").
		Reason("REPLY_DIFFERENT_CHANNEL").
		Meta("domain", "messages")
}

func ErrMessageTypeInvalid() error {
	return errs.InvalidArgument("Invalid message type.").
		Reason("MESSAGE_TYPE_INVALID").
		FieldViolation("type", "Must be a valid message type.", "INVALID_ENUM_VALUE").
		Meta("domain", "messages")
}

// -----------------------------------------------------------------------------
// reactions
// -----------------------------------------------------------------------------

func ErrReactionEmojiRequired() error {
	return errs.InvalidArgument("Emoji is required.").
		Reason("EMOJI_REQUIRED").
		FieldViolation("emoji", "Emoji cannot be empty.", "REQUIRED").
		Meta("domain", "reactions")
}

func ErrReactionEmojiTooLong() error {
	return errs.InvalidArgument("Emoji too long.").
		Reason("EMOJI_TOO_LONG").
		FieldViolation("emoji", "Emoji must be 64 characters or fewer.", "MAX_LENGTH_EXCEEDED").
		Meta("domain", "reactions")
}
