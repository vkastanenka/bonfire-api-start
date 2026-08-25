package relation

import "bonfire-api/internal/errs"

func ErrTypeInvalid() *errs.Error {
	return errs.InvalidArgument("Invalid relationship type.").
		Reason("RELATIONSHIP_TYPE_INVALID").
		FieldViolation("type", "Must be one of: pending, friends, blocked.", "INVALID_ENUM_VALUE")
}

func ErrPeerIDInvalid() *errs.Error {
	return errs.InvalidArgument("Relation ids cannot match.").
		FieldViolation("peer_id", "ID is the same as user ID", "PEER_ID_INVALID")
}

// Added Errors

func ErrBlockedActor() *errs.Error {
	return errs.InvalidArgument("Action cannot be performed on a blocked relationship.").
		Reason("RELATION_BLOCKED_ACTOR").
		FieldViolation("actor_id", "Actor is blocked", "BLOCKED_ACTOR")
}

func ErrNotPending() *errs.Error {
	return errs.Internal("Cannot accept relation that is not pending.").
		Reason("RELATION_NOT_PENDING")
}

func ErrAlreadyFriends() *errs.Error {
	return errs.AlreadyExists("Already friends with this user.").
		Reason("RELATION_ALREADY_FRIENDS")
}

func ErrPermissionDenied() *errs.Error {
	return errs.PermissionDenied("Cannot interact with this user.").
		Reason("RELATION_PERMISSION_DENIED")
}

func ErrAlreadyPending() *errs.Error {
	return errs.AlreadyExists("Friend request already pending.").
		Reason("RELATION_ALREADY_PENDING")
}
