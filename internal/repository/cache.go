package repository

import (
	"bonfire-api/internal/channel"
	"bonfire-api/internal/fields"
	"context"
)

type ChannelCache interface {
	AddMembers(ctx context.Context, channelID fields.ID, members []*channel.Member) error
	CreateGroup(ctx context.Context, ch *channel.Channel, members []*channel.Member) error
	Delete(ctx context.Context, id fields.ID) error
	Get(ctx context.Context, id fields.ID) (*channel.Channel, error)
	GetBatchMembersByChannelIDs(ctx context.Context, channelIDs []fields.ID) (map[fields.ID][]*channel.Member, []fields.ID, error)
	InvalidateMember(ctx context.Context, channelID fields.ID, userID fields.ID) error
	InvalidateMembers(ctx context.Context, channelID fields.ID) error
	Set(ctx context.Context, ch *channel.Channel) error
	SetBatchMembers(ctx context.Context, channelMembersMap map[fields.ID][]*channel.Member) error
}
