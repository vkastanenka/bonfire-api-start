package user

import (
	"bonfire-api/internal/repository"
	"time"

	"github.com/google/uuid"
)

type UserView struct {
	ID         uuid.UUID  `json:"id"`
	Email      string     `json:"email"`
	Username   string     `json:"username"`
	VerifiedAt *time.Time `json:"verified_at"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

func NewUserView(row repository.User) UserView {
	var verifiedAt *time.Time
	if row.VerifiedAt.Valid {
		t := row.VerifiedAt.Time
		verifiedAt = &t
	}

	return UserView{
		ID:         uuid.UUID(row.ID.Bytes),
		Email:      row.Email,
		Username:   row.Username,
		VerifiedAt: verifiedAt,
		CreatedAt:  row.CreatedAt.Time,
		UpdatedAt:  row.UpdatedAt.Time,
	}
}
