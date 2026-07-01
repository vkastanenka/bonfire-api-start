package relationship

import (
	"bytes"
	"time"

	"bonfire-api/internal/presence"
	"bonfire-api/internal/repository"

	"github.com/google/uuid"
)

// --- relationship Status ---

type Status string

const (
	StatusAll     Status = "all"
	StatusPending Status = "pending"
	StatusFriends Status = "friends"
	StatusBlocked Status = "blocked"
	StatusOnline  Status = "online"
)

// --- relationship Views ---

type View struct {
	User1ID      uuid.UUID `json:"user1_id"`
	User2ID      uuid.UUID `json:"user2_id"`
	ActionUserID uuid.UUID `json:"action_user_id"`
	Status       Status    `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
}

func NewView(row repository.Relationship) View {
	return View{
		User1ID:      uuid.UUID(row.User1ID.Bytes),
		User2ID:      uuid.UUID(row.User2ID.Bytes),
		ActionUserID: uuid.UUID(row.ActionUserID.Bytes),
		Status:       Status(row.Status),
		CreatedAt:    row.CreatedAt.Time,
	}
}

type UserView struct {
	UserID       uuid.UUID         `json:"user_id"`
	Username     string            `json:"username"`
	ActionUserID uuid.UUID         `json:"action_user_id"`
	Status       Status            `json:"status"`
	Activity     presence.Activity `json:"activity"`
	CreatedAt    time.Time         `json:"created_at"`
}

func NewUserView(row repository.RelationshipsListByUserRow, activity presence.Activity) UserView {
	return UserView{
		UserID:       row.RelatedUserID.Bytes,
		Username:     row.Username,
		ActionUserID: row.ActionUserID.Bytes,
		Status:       Status(row.Status),
		Activity:     presence.Activity(row.Status),
		CreatedAt:    row.CreatedAt.Time,
	}
}

// --- relationship orderUUIDs ---

func orderUUIDs(id1, id2 uuid.UUID) (uuid.UUID, uuid.UUID) {
	if bytes.Compare(id1[:], id2[:]) < 0 {
		return id1, id2
	}
	return id2, id1
}
