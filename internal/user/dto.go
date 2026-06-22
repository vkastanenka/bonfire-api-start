package user

import (
	"bonfire-api/internal/repository"
	"time"

	"github.com/google/uuid"
)

// ==========================================
// HANDLERS
// ==========================================

type PingRes struct {
	Status string `json:"status"`
}

type CountRes struct {
	Count int64 `json:"count"`
}

type CreateReq struct {
	Email    string `json:"email" validate:"required,email,max=255"`
	Username string `json:"username" validate:"required,min=4,max=32,valid_username"`
	Password string `json:"password" validate:"required,min=12,max=128"`
}

// ==========================================
// SERVICES
// ==========================================

type CheckAvailabilityParams struct {
	Email    string `json:"email"`
	Username string `json:"username"`
}

type CheckAvailabilityResult struct {
	Email    bool `json:"email"`
	Username bool `json:"username"`
}

type CreateParams struct {
	Email    string `json:"email"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type ListParams struct {
	Limit  int32      `json:"limit"`
	Cursor *uuid.UUID `json:"cursor"`
}

type UpdatePasswordParams struct {
	ID   uuid.UUID `json:"id"`
	Hash string    `json:"hash"`
}

type EnableTOTPParams struct {
	ID     uuid.UUID `json:"id"`
	Secret string    `json:"secret"`
}

// ==========================================
// VIEW
// ==========================================

type View struct {
	ID         uuid.UUID  `json:"id"`
	Email      string     `json:"email"`
	Username   string     `json:"username"`
	VerifiedAt *time.Time `json:"verified_at"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

func NewView(row repository.User) View {
	var verifiedAt *time.Time
	if row.VerifiedAt.Valid {
		t := row.VerifiedAt.Time
		verifiedAt = &t
	}

	return View{
		ID:         uuid.UUID(row.ID.Bytes),
		Email:      row.Email,
		Username:   row.Username,
		VerifiedAt: verifiedAt,
		CreatedAt:  row.CreatedAt.Time,
		UpdatedAt:  row.UpdatedAt.Time,
	}
}
