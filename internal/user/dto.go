package user

import (
	"bonfire-api/internal/repository"
	"time"

	"github.com/google/uuid"
)

// ==========================================
// HANDLERS
// ==========================================

// users

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

// user_delete_requests

type CreateDeleteRequestReq struct {
	UserID      string    `json:"user_id" validate:"required,uuid"`
	ScheduledAt time.Time `json:"scheduled_at" validate:"required"`
}

// user_profiles

type CreateProfileReq struct {
	UserID      string `json:"user_id" validate:"required,uuid"`
	DisplayName string `json:"display_name" validate:"required,min=3,max=32"`
}

type UpdateProfileDisplayNameReq struct {
	DisplayName string `json:"display_name" validate:"required,min=3,max=32"`
}

// ==========================================
// SERVICES
// ==========================================

// users

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

// user_delete_requests

type CreateDeleteRequestParams struct {
	UserID      uuid.UUID `json:"user_id"`
	ScheduledAt time.Time `json:"scheduled_at"`
}

// user_profiles

type CreateProfileParams struct {
	UserID      uuid.UUID
	DisplayName string
}

type UpdateProfileDisplayNameParams struct {
	UserID      uuid.UUID
	DisplayName string
}

// ==========================================
// VIEWS
// ==========================================

// users

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

type AuthView struct {
	ID              uuid.UUID  `json:"id"`
	Email           string     `json:"email"`
	PasswordHash    string     `json:"password"`
	IsTOTPEnabled   bool       `json:"is_totp_enabled"`
	TOTPSecret      *string    `json:"totp_secret"`
	VerifiedAt      *time.Time `json:"verified_at"`
	Role            Role       `json:"role"`
	Status          Status     `json:"status"`
	SecurityVersion int        `json:"security_version"`
}

func (a *AuthView) IsActive() bool {
	return a.Status == "active"
}

func NewAuthView(row repository.User) AuthView {
	var verifiedAt *time.Time
	if row.VerifiedAt.Valid {
		t := row.VerifiedAt.Time
		verifiedAt = &t
	}

	var totpSecret *string
	if row.TotpSecret.Valid {
		s := row.TotpSecret.String
		totpSecret = &s
	}

	return AuthView{
		ID:              uuid.UUID(row.ID.Bytes),
		Email:           row.Email,
		PasswordHash:    row.PasswordHash,
		IsTOTPEnabled:   row.IsTotpEnabled,
		TOTPSecret:      totpSecret,
		VerifiedAt:      verifiedAt,
		Role:            Role(row.Role),
		Status:          Status(row.Status),
		SecurityVersion: int(row.SecurityVersion),
	}
}

// user_delete_requests

type DeleteRequestView struct {
	UserID      uuid.UUID `json:"user_id"`
	CreatedAt   time.Time `json:"created_at"`
	ScheduledAt time.Time `json:"scheduled_at"`
}

func NewDeleteRequestView(row repository.UserDeleteRequest) DeleteRequestView {
	return DeleteRequestView{
		UserID:      row.UserID.Bytes,
		CreatedAt:   row.CreatedAt.Time,
		ScheduledAt: row.ScheduledAt.Time,
	}
}

// user_profiles

type ProfileView struct {
	UserID      uuid.UUID `json:"user_id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	DisplayName string    `json:"display_name"`
}

func NewProfileView(row repository.UserProfile) ProfileView {
	return ProfileView{
		UserID:      uuid.UUID(row.UserID.Bytes),
		CreatedAt:   row.CreatedAt.Time,
		UpdatedAt:   row.UpdatedAt.Time,
		DisplayName: row.DisplayName,
	}
}
