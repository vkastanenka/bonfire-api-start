package channel

import "bonfire-api/internal/errs"

func ErrChannelTypeInvalid() error {
	return errs.InvalidArgument("Invalid channel type.").
		Reason("CHANNEL_TYPE_INVALID").
		FieldViolation("type", "Must be one of: DIRECT, GROUP.", "INVALID_ENUM_VALUE").
		Meta("domain", "channels")
}

func ErrMembersNotFound() error {
	return errs.NotFound("Channel members not found.")
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

func ErrMessageContentTooLong() error {
	return errs.InvalidArgument("Content too long.").
		Reason("CONTENT_TOO_LONG").
		FieldViolation("content", "Content must be 4000 characters or fewer.", "MAX_LENGTH_EXCEEDED").
		Meta("domain", "messages")
}

func ErrMessageContentRequired() error {
	return errs.InvalidArgument("Message content is required.").
		Reason("CONTENT_REQUIRED").
		FieldViolation("content", "Field is required.", "REQUIRED").
		Meta("domain", "messages")
}

func ErrMessageTypeInvalid() error {
	return errs.InvalidArgument("Invalid message type.").
		Reason("MESSAGE_TYPE_INVALID").
		FieldViolation("type", "Must be a valid message type.", "INVALID_ENUM_VALUE").
		Meta("domain", "messages")
}

func ErrMuteDurationInvalid() error {
	return errs.InvalidArgument("Invalid mute duration.").
		Reason("MUTE_DURATION_INVALID").
		FieldViolation("mute_duration", "Must be one of: 15_MIN, 1_HOUR, 8_HOURS, 24_HOURS, 3_DAYS, FOREVER.", "INVALID_ENUM_VALUE").
		Meta("domain", "members")
}

func ErrReactionEmojiTooLong() error {
	return errs.InvalidArgument("Emoji too long.").
		Reason("EMOJI_TOO_LONG").
		FieldViolation("emoji", "Emoji must be 64 characters or fewer.", "MAX_LENGTH_EXCEEDED").
		Meta("domain", "reactions")
}

func ErrReactionEmojiRequired() error {
	return errs.InvalidArgument("Emoji is required.").
		Reason("EMOJI_REQUIRED").
		FieldViolation("emoji", "Emoji cannot be empty.", "REQUIRED").
		Meta("domain", "reactions")
}
