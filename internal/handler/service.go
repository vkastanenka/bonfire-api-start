package handler

import (
	"bonfire-api/internal/auth"
	"bonfire-api/internal/channel"
	"bonfire-api/internal/fields"
	"bonfire-api/internal/relation"
	"bonfire-api/internal/session"
	"bonfire-api/internal/user"
	"context"
	"time"

	"github.com/google/uuid"
)

type AuthService interface {
	ForgotPassword(ctx context.Context, rawEmail string) error
	Login(ctx context.Context, p auth.LoginParams) (auth.LoginResult, error)
	PrintWSTicket(ctx context.Context, rawUserID uuid.UUID, rawSessionID uuid.UUID) (fields.ID, error)
	Refresh(ctx context.Context, p auth.RefreshParams) (auth.RefreshResult, error)
	Register(ctx context.Context, p auth.RegisterParams) (auth.RegisterResult, error)
	ResendVerify(ctx context.Context, rawUserID uuid.UUID) error
	ResetPassword(ctx context.Context, p auth.ResetPasswordParams) (auth.ResetPasswordResult, error)
	VerifyEmail(ctx context.Context, tokenStr string) error
}

type ChannelService interface {
	CreateGroup(ctx context.Context, rawActorID uuid.UUID, rawPeerIDs []uuid.UUID) error
	Get(ctx context.Context, rawActorID uuid.UUID, rawChannelID uuid.UUID, rawMessageID uuid.UUID) (*channel.Channel, []channel.MemberView, []channel.MessageView, error)
	GetSidebar(ctx context.Context, rawActorID uuid.UUID) (channelMap map[fields.ID]*channel.Channel, memberMap map[fields.ID]*channel.Member, peerIDsMap map[fields.ID][]fields.ID, channelIDs []fields.ID, peerIDs []fields.ID, directPeerIDs []fields.ID, err error)
	UpdateGroup(ctx context.Context, rawActorID uuid.UUID, rawChannelID uuid.UUID, rawName *string, rawIconURL *string) (*channel.Channel, error)
}

type MemberService interface {
	AddMembers(ctx context.Context, rawActorID uuid.UUID, rawChannelID uuid.UUID, rawMemberIDs []uuid.UUID) error
	CloseDirect(ctx context.Context, rawActorID uuid.UUID, rawChannelID uuid.UUID) error
	LeaveGroup(ctx context.Context, rawActorID uuid.UUID, rawChannelID uuid.UUID) error
	UpdateLastReadMessage(ctx context.Context, rawActorID uuid.UUID, rawChannelID uuid.UUID, rawLastReadMessageID uuid.UUID) (*channel.Member, error)
	UpdateMutedUntil(ctx context.Context, rawActorID uuid.UUID, rawChannelID uuid.UUID, rawDuration *int) (*channel.Member, error)
	UpdatePinnedAt(ctx context.Context, rawActorID uuid.UUID, rawChannelID uuid.UUID, isPinned bool) (*channel.Member, error)
}

type MessageService interface {
	Create(ctx context.Context, rawAuthorID uuid.UUID, rawChannelID uuid.UUID, rawContent *string, rawReplyToMsgID *uuid.UUID, rawFwdMsgID *uuid.UUID, rawFwdChannelID *uuid.UUID) (*channel.MessageView, error)
	Delete(ctx context.Context, rawActorID uuid.UUID, rawChannelID uuid.UUID, rawMessageID uuid.UUID) error
	ListAfter(ctx context.Context, rawActorID uuid.UUID, rawChannelID uuid.UUID, rawMsgCursorID uuid.UUID) ([]channel.MessageView, error)
	ListAround(ctx context.Context, rawActorID uuid.UUID, rawChannelID uuid.UUID, rawMsgCursorID uuid.UUID) ([]channel.MessageView, error)
	ListBefore(ctx context.Context, rawActorID uuid.UUID, rawChannelID uuid.UUID, rawMsgCursorID uuid.UUID) ([]channel.MessageView, error)
	ListPinned(ctx context.Context, rawActorID uuid.UUID, rawChannelID uuid.UUID, rawMsgCursorID *uuid.UUID, rawCursorPinnedAt *time.Time) ([]channel.MessagePinnedView, error)
	ToggleReaction(ctx context.Context, rawActorID uuid.UUID, rawChannelID uuid.UUID, rawMessageID uuid.UUID, rawEmoji string) (*channel.EmojiCount, error)
	UpdateContent(ctx context.Context, rawActorID uuid.UUID, rawChannelID uuid.UUID, rawMessageID uuid.UUID, rawContent string) (*channel.Message, error)
	UpdatePinnedAt(ctx context.Context, rawActorID uuid.UUID, rawChannelID uuid.UUID, rawMessageID uuid.UUID, isPinned bool) (*channel.Message, error)
}

type RelationService interface {
	DeleteByUserID(ctx context.Context, actorID uuid.UUID, peerID uuid.UUID) error
	TransitionBlocked(ctx context.Context, actorID uuid.UUID, peerID uuid.UUID) error
	TransitionFriends(ctx context.Context, actorID uuid.UUID, peerID uuid.UUID) (*relation.TransitionFriendsResult, error)
	TransitionPending(ctx context.Context, actorID uuid.UUID, peerID uuid.UUID) error
}

type SessionService interface {
	ListValidByUserID(ctx context.Context, userID uuid.UUID) ([]*session.Session, error)
	Revoke(ctx context.Context, p session.RevokeParams) error
	RevokeAll(ctx context.Context, userID uuid.UUID) error
}

type UserService interface {
	AnonymizeBatch(ctx context.Context) error
	Disable(ctx context.Context, p user.DisableParams) error
	Get(ctx context.Context, userID uuid.UUID) (*user.User, error)
	ScheduleDelete(ctx context.Context, p user.ScheduleDeleteParams) error
	UpdateEmail(ctx context.Context, p user.UpdateEmailParams) (*user.User, error)
	UpdatePassword(ctx context.Context, p user.UpdatePasswordParams) error
	UpdatePreferredPresence(ctx context.Context, p user.UpdatePreferredPresenceParams) (*user.User, error)
	UpdateProfile(ctx context.Context, p user.UpdateProfileParams) (*user.User, error)
	UpdateUsername(ctx context.Context, p user.UpdateUsernameParams) (*user.User, error)
	fetchAndAuthenticate(ctx context.Context, actorID uuid.UUID, password string) (*user.User, error)
	fetchBatchValid(ctx context.Context, userIDs []uuid.UUID) (map[uuid.UUID]*user.User, error)
	fetchValid(ctx context.Context, actorID uuid.UUID) (*user.User, error)
}
