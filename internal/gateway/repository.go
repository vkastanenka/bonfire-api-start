package gateway

import (
	"bonfire-api/internal/fields"
	"bonfire-api/internal/pubsub"
	"bonfire-api/internal/user"
	"context"
	"encoding/json"

	"github.com/google/uuid"
)

type GatewayCache interface{}

type UserCache interface {
	AddChannel(ctx context.Context, userID fields.ID, channelID fields.ID) error
	AddFriend(ctx context.Context, userID fields.ID, friendID fields.ID) error
	AddNode(ctx context.Context, userID fields.ID, nodeID fields.ID) error
	Delete(ctx context.Context, id fields.ID) error
	DeleteBatch(ctx context.Context, ids []fields.ID) error
	Get(ctx context.Context, id fields.ID) (*user.User, error)
	GetBatch(ctx context.Context, ids []fields.ID) (map[fields.ID]*user.User, []fields.ID, error)
	GetBatchNodes(ctx context.Context, userIDs []fields.ID) (map[fields.ID][]fields.ID, error)
	GetBatchPresence(ctx context.Context, userIDs []fields.ID) (map[fields.ID]user.Presence, error)
	GetFriends(ctx context.Context, userID fields.ID) ([]fields.ID, error)
	GetPresence(ctx context.Context, userID fields.ID) (user.Presence, error)
	GetUpdateRecipients(ctx context.Context, userID fields.ID) ([]fields.ID, error)
	Heartbeat(ctx context.Context, userID fields.ID, nodeID fields.ID) error
	RegisterWSConnection(ctx context.Context, userID fields.ID, nodeID fields.ID, presence user.Presence) error
	RemoveBatchNode(ctx context.Context, userIDs []fields.ID, nodeID fields.ID) error
	RemoveChannel(ctx context.Context, userID fields.ID, channelID fields.ID) error
	RemoveFriend(ctx context.Context, userID fields.ID, friendID fields.ID) error
	RemoveNode(ctx context.Context, userID fields.ID, nodeID fields.ID) error
	Set(ctx context.Context, usr *user.User) error
	SetBatch(ctx context.Context, users map[fields.ID]*user.User) error
	SetChannels(ctx context.Context, userID fields.ID, channelIDs []fields.ID) error
	SetFriends(ctx context.Context, userID fields.ID, friendIDs []fields.ID) error
	SetPresence(ctx context.Context, userID fields.ID, p user.Presence) error
	UnregisterWSConnection(ctx context.Context, userID fields.ID, nodeID fields.ID) (bool, error)
	GetFriendNodes(ctx context.Context, userID fields.ID) (map[fields.ID][]fields.ID, error)
	GetUpdateRecipientNodes(ctx context.Context, userID fields.ID) (map[fields.ID][]fields.ID, error)
}

type UserService interface {
	AnonymizeBatch(ctx context.Context) error
	Disable(ctx context.Context, p user.DisableParams) error
	Get(ctx context.Context, userID uuid.UUID) (*user.User, error)
	GetBatch(ctx context.Context, ids []fields.ID) (map[fields.ID]*user.User, error)
	GetBatchPresence(ctx context.Context, userIDs []fields.ID) (map[fields.ID]user.Presence, error)
	HandleHeartbeat(ctx context.Context, userID fields.ID, nodeID fields.ID, newPresence user.Presence) error
	Publish(ctx context.Context, nodeIDs []fields.ID, userIDs []fields.ID, eventType string, payload json.RawMessage) error
	RegisterWSConnection(ctx context.Context, userID fields.ID, nodeID fields.ID, presence user.Presence) error
	RemoveBatchNode(ctx context.Context, userIDs []fields.ID, nodeID fields.ID) error
	ScheduleDelete(ctx context.Context, p user.ScheduleDeleteParams) error
	UnregisterWSConnection(ctx context.Context, userID fields.ID, nodeID fields.ID) error
	UpdateEmail(ctx context.Context, p user.UpdateEmailParams) (*user.User, error)
	UpdatePassword(ctx context.Context, p user.UpdatePasswordParams) error
	UpdatePreferredPresence(ctx context.Context, p user.UpdatePreferredPresenceParams) (*user.User, error)
	UpdateProfile(ctx context.Context, p user.UpdateProfileParams) (*user.User, error)
	UpdateUsername(ctx context.Context, p user.UpdateUsernameParams) (*user.User, error)
}

type TicketCache interface {
	Print(ctx context.Context, ticketID fields.ID, userID fields.ID, sessionID fields.ID) error
	Punch(ctx context.Context, ticketID fields.ID) (fields.ID, fields.ID, error)
}

type UserRepository interface {
	Availability(ctx context.Context, email *user.Email, username *user.Username) (bool, bool, error)
	Create(ctx context.Context, u *user.User) (*user.User, error)
	Get(ctx context.Context, id fields.ID) (*user.User, error)
	GetBatch(ctx context.Context, ids []fields.ID) (map[fields.ID]*user.User, error)
	GetByEmail(ctx context.Context, email user.Email) (*user.User, error)
	ListDeleteScheduled(ctx context.Context, currentTime fields.Timestamp, limitVal int) ([]*user.User, error)
	SetDeleteSchedule(ctx context.Context, id fields.ID, deleteScheduledAt fields.Timestamp, disabledAt fields.Timestamp, updatedAt fields.Timestamp) (*user.User, error)
	SetDisabled(ctx context.Context, id fields.ID, disabledAt fields.Timestamp, updatedAt fields.Timestamp) (*user.User, error)
	Update(ctx context.Context, u *user.User) (*user.User, error)
	UpdateBatch(ctx context.Context, users []*user.User) ([]*user.User, error)
	UpdateEmail(ctx context.Context, id fields.ID, email user.Email, updatedAt fields.Timestamp) (*user.User, error)
	UpdatePasswordHash(ctx context.Context, id fields.ID, passwordHash user.PasswordHash, updatedAt fields.Timestamp) (*user.User, error)
	UpdatePhone(ctx context.Context, id fields.ID, phone user.Phone, updatedAt fields.Timestamp) (*user.User, error)
	UpdatePresence(ctx context.Context, id fields.ID, presence user.PreferredPresence, presenceUntil fields.Timestamp, updatedAt fields.Timestamp) (*user.User, error)
	UpdateProfile(ctx context.Context, id fields.ID, displayName user.DisplayName, bio user.Bio, avatarURL fields.URL, bannerColor fields.HexColor, updatedAt fields.Timestamp) (*user.User, error)
	UpdateUsername(ctx context.Context, id fields.ID, username user.Username, updatedAt fields.Timestamp) (*user.User, error)
	Verify(ctx context.Context, id fields.ID, verifiedAt fields.Timestamp, updatedAt fields.Timestamp) (*user.User, error)
}

type OutboxRepository interface {
	Publish(ctx context.Context, eventType string, payload any, now fields.Timestamp) error
}

type TX interface {
	ExecTx(ctx context.Context, fn func(txCtx context.Context) error) error
}

type GatewayPub interface {
	PublishNodeEvent(ctx context.Context, nodeID fields.ID, event pubsub.NodeEvent) error
	PublishNodeEvents(ctx context.Context, nodeIDs []fields.ID, event pubsub.NodeEvent) error
	PublishBatchNodeEvents(ctx context.Context, events map[fields.ID]pubsub.NodeEvent) error
}
