package channel

import (
	"bonfire-api/internal/fields"
	"bonfire-api/internal/presence"
	"bonfire-api/internal/user"
	"context"
	"time"

	"github.com/google/uuid"
)

type channelService interface {
	CreateGroup(ctx context.Context, rawUserID uuid.UUID, rawMemberIDs []uuid.UUID) error
	Get(ctx context.Context, rawUserID uuid.UUID, rawChannelID uuid.UUID) (*Channel, []MemberView, []MessageView, error)
	GetSidebar(ctx context.Context, rawUserID uuid.UUID) ([]ChannelSidebarView, error)
	ListPinned(ctx context.Context, rawActorID uuid.UUID, rawChannelID uuid.UUID, rawCursorID *uuid.UUID, rawCursorPinnedAt *time.Time) ([]MessagePinnedView, error)
	UpdateGroup(ctx context.Context, rawUserID uuid.UUID, rawChannelID uuid.UUID, rawName *string, rawIconURL *string) (*Channel, error)
	hydrateSidebarViews(channels []*Channel, membershipMap map[fields.ID]*Member, peerMembersMap map[fields.ID][]*Member, userMap map[fields.ID]*user.User, presenceMap map[fields.ID]presence.Presence) []ChannelSidebarView
}

type memberService interface {
	AddMembers(ctx context.Context, rawActorID uuid.UUID, rawChannelID uuid.UUID, rawMemberIDs []uuid.UUID) error
	CloseDirect(ctx context.Context, rawActorID uuid.UUID, rawChannelID uuid.UUID) error
	LeaveGroup(ctx context.Context, rawActorID uuid.UUID, rawChannelID uuid.UUID) error
	UpdateLastReadMessage(ctx context.Context, rawActorID uuid.UUID, rawChannelID uuid.UUID, rawLastReadMessageID uuid.UUID, rawLastReadAt time.Time) (*Member, error)
	UpdateGroupMutedUntil(ctx context.Context, rawActorID uuid.UUID, rawChannelID uuid.UUID, rawMutedUntil *time.Time) (*Member, error)
	UpdatePinnedAt(ctx context.Context, rawActorID uuid.UUID, rawChannelID uuid.UUID, rawPinnedAt *time.Time) (*Member, error)
}

type messageService interface {
	Create(ctx context.Context, rawAuthorID uuid.UUID, rawChannelID uuid.UUID, rawContent *string, rawReplyToMsgID *uuid.UUID, rawFwdMsgID *uuid.UUID, rawFwdChannelID *uuid.UUID) (*MessageView, error)
	Delete(ctx context.Context, rawActorID uuid.UUID, rawMessageID uuid.UUID) error
	ListAfter(ctx context.Context, rawUserID uuid.UUID, rawChannelID uuid.UUID, rawAfterID uuid.UUID) ([]MessageView, error)
	ListBefore(ctx context.Context, rawActorID uuid.UUID, rawChannelID uuid.UUID, rawMsgCursorID uuid.UUID) ([]MessageView, error)
	ToggleReaction(ctx context.Context, rawActorID uuid.UUID, rawMessageID uuid.UUID, rawEmoji string) (*ReactionView, error)
	UpdateContent(ctx context.Context, rawActorID uuid.UUID, rawMessageID uuid.UUID, rawContent string) (*Message, error)
	UpdatePinnedAt(ctx context.Context, rawActorID uuid.UUID, rawMessageID uuid.UUID, isPinned bool) (*Message, error)
}

// TODO: Add mark unread
