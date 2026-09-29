package handler

import (
	"bonfire-api/internal/auth"
	"bonfire-api/internal/bootstrap"
	"bonfire-api/internal/channel"
	"bonfire-api/internal/relation"
	"bonfire-api/internal/session"
	"bonfire-api/internal/user"
	"context"
	"time"

	"github.com/google/uuid"
)

type AuthService interface {
	ForgotPassword(ctx context.Context, email string) error
	Login(ctx context.Context, p auth.LoginParams) (auth.LoginResult, error)
	PrintWSTicket(ctx context.Context) (uuid.UUID, error)
	Refresh(ctx context.Context, refreshToken string) (auth.RefreshResult, error)
	Register(ctx context.Context, p auth.RegisterParams) (auth.RegisterResult, error)
	ResendVerify(ctx context.Context) error
	ResetPassword(ctx context.Context, p auth.ResetPasswordParams) (auth.ResetPasswordResult, error)
	VerifyEmail(ctx context.Context, token string) (*user.User, error)
}

type BootstrapService interface {
	Bootstrap(ctx context.Context) (*bootstrap.Result, error)
}

type ChannelService interface {
	CreateGroup(ctx context.Context, rawMemberIDs []uuid.UUID) (*channel.CreateGroupResult, error)
	UpdateGroup(ctx context.Context, channelID uuid.UUID, name *string, iconURL *string) (*channel.UpdateGroupResult, error)
}

type MemberService interface {
	AddMembers(ctx context.Context, channelID uuid.UUID, newPeerIDs []uuid.UUID) (*channel.AddMembersResult, error)
	CloseDirect(ctx context.Context, channelID uuid.UUID) error
	LeaveGroup(ctx context.Context, channelID uuid.UUID) error
	UpdateLastReadMessage(ctx context.Context, channelID uuid.UUID, lastReadMessageID *uuid.UUID) (*channel.Member, error)
	UpdateMutedUntil(ctx context.Context, channelID uuid.UUID, rawDuration *string) (*channel.Member, error)
	UpdatePinnedAt(ctx context.Context, channelID uuid.UUID, isPinned bool) (*channel.Member, error)
}

type MessageService interface {
	Create(ctx context.Context, channelID uuid.UUID, content *string, replyToMsgID *uuid.UUID, fwdMsgID *uuid.UUID, fwdChannelID *uuid.UUID) (*channel.CreateMessageResult, error)
	Delete(ctx context.Context, actorID uuid.UUID, sessionID uuid.UUID, channelID uuid.UUID, messageID uuid.UUID) error
	ListAfter(ctx context.Context, actorID uuid.UUID, channelID uuid.UUID, msgCursorID uuid.UUID) (*channel.GetMessageViewsResult, bool, error)
	ListAround(ctx context.Context, actorID uuid.UUID, channelID uuid.UUID, msgCursorID uuid.UUID) (*channel.GetMessageViewsResult, bool, bool, error)
	ListBefore(ctx context.Context, actorID uuid.UUID, channelID uuid.UUID, msgCursorID uuid.UUID) (*channel.GetMessageViewsResult, bool, error)
	ListPinned(ctx context.Context, actorID uuid.UUID, channelID uuid.UUID, msgCursorID *uuid.UUID, cursorPinnedAt *time.Time) ([]*channel.Message, map[uuid.UUID]*user.User, bool, error)
	ToggleReaction(ctx context.Context, actorID uuid.UUID, sessionID uuid.UUID, channelID uuid.UUID, messageID uuid.UUID, emoji string) (*channel.EmojiCount, error)
	UpdateContent(ctx context.Context, actorID uuid.UUID, sessionID uuid.UUID, channelID uuid.UUID, messageID uuid.UUID, content string) (*channel.Message, error)
	UpdatePinnedAt(ctx context.Context, actorID uuid.UUID, sessionID uuid.UUID, channelID uuid.UUID, messageID uuid.UUID, isPinned bool) (*channel.Message, error)
}

type RelationService interface {
	DeleteByUserID(ctx context.Context, peerID uuid.UUID) error
	TransitionBlocked(ctx context.Context, peerID uuid.UUID) error
	TransitionFriends(ctx context.Context, peerID uuid.UUID) (*relation.TransitionFriendsResult, error)
	TransitionPending(ctx context.Context, peerID uuid.UUID) error
}

type SessionService interface {
	ListValidByUserID(ctx context.Context) ([]*session.Session, error)
	Revoke(ctx context.Context, p session.RevokeParams) error
	RevokeAll(ctx context.Context) error
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
}
