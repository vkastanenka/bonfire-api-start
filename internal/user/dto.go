package user

import (
	"bonfire-api/internal/repository"
	"time"

	"github.com/google/uuid"
)

type UserResponse struct {
	ID         uuid.UUID  `json:"id"`
	Email      string     `json:"email"`
	Username   string     `json:"username"`
	VerifiedAt *time.Time `json:"verified_at"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

// CreateUserResponse translates a raw sqlc DB user row into a public UserResponse DTO.
func CreateUserResponse(row repository.User) UserResponse {
	var verifiedAt *time.Time
	if row.VerifiedAt.Valid {
		t := row.VerifiedAt.Time
		verifiedAt = &t
	}

	return UserResponse{
		ID:         uuid.UUID(row.ID.Bytes),
		Email:      row.Email,
		Username:   row.Username,
		VerifiedAt: verifiedAt,
		CreatedAt:  row.CreatedAt.Time,
		UpdatedAt:  row.UpdatedAt.Time,
	}
}
