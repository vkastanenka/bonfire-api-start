package auth

import (
	"bonfire-api/internal/apperr"
	"bonfire-api/internal/crypto"
	"bonfire-api/internal/httpio"
	"bonfire-api/internal/repository"
	"bonfire-api/internal/sanitize"
	"bonfire-api/internal/user"
	"bonfire-api/internal/worker"
	"context"
	"net/http"
)

// --- REFRESH CONSTANTS ---

// Messages
const (
	msgRegisterSuccess = "register_success"
)

// Errors
const (
	errEmailTaken       = "Email taken."
	errUsernameTaken    = "Username taken."
	errCredentialsTaken = "Email and/or username taken."
	ErrHashPassword     = "Hash password failed."
)

// --- REFRESH ERRORS ---

func NewRegisterConflictError(emailAvailable bool, usernameAvailable bool) error {
	var params []apperr.InvalidParam

	if !emailAvailable {
		params = append(params, apperr.InvalidParam{Name: "email", Reason: errEmailTaken})
	}
	if !usernameAvailable {
		params = append(params, apperr.InvalidParam{Name: "username", Reason: errUsernameTaken})
	}

	return apperr.New(
		apperr.CodeConflict,
		errCredentialsTaken,
		apperr.WithInvalidParams(params),
	)
}

// --- REGISTER DTOs ---

type RegisterReq struct {
	Email       string  `json:"email" validate:"required,email,max=255"`
	DisplayName *string `json:"display_name" validate:"omitempty,min=3,max=32"`
	Username    string  `json:"username" validate:"required,min=4,max=32,valid_username"`
	Password    string  `json:"password" validate:"required,min=12,max=128"`
}

func (r *RegisterReq) Sanitize() {
	r.Email = sanitize.SanitizeEmail(r.Email)

	if r.DisplayName != nil {
		cleaned := sanitize.SanitizeText(*r.DisplayName)
		r.DisplayName = &cleaned
	}
}

type RegisterParams struct {
	Email       string
	Username    string
	DisplayName *string
	Password    string
}

type RegisterResult struct {
	User        user.View        `json:"user"`
	UserProfile user.ProfileView `json:"user_profile"`
}

// --- REGISTER HANDLER ---

// Register
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) error {
	// Bind JSON
	reqData, err := httpio.BindJSON[RegisterReq](w, r, h.validator)
	if err != nil {
		return err
	}

	// Register user
	data, err := h.service.Register(r.Context(), RegisterParams{
		Email:       reqData.Email,
		Username:    reqData.Username,
		DisplayName: reqData.DisplayName,
		Password:    reqData.Password,
	})
	if err != nil {
		return err
	}

	// Respond
	httpio.RespondCreated(w, r, data, msgRegisterSuccess)

	return nil
}

// --- REGISTER SERVICE ---

// Register
func (s *Service) Register(ctx context.Context, r RegisterParams) (RegisterResult, error) {
	// Define result
	var result RegisterResult

	// Check if credentials are available
	availability, err := s.store.UserCheckAvailability(ctx, repository.UserCheckAvailabilityParams{
		Email:    r.Email,
		Username: r.Username,
	})
	if err != nil {
		return RegisterResult{}, apperr.NewDBError(err)
	}

	// Cleanly handle conflict
	if !availability.EmailAvailable || !availability.UsernameAvailable {
		return RegisterResult{}, NewRegisterConflictError(availability.EmailAvailable, availability.UsernameAvailable)
	}

	// Hash password
	hashedPasswordBytes, err := crypto.HashPassword(r.Password)
	if err != nil {
		return RegisterResult{}, apperr.NewInternal(err)
	}
	passwordHash := string(hashedPasswordBytes)

	// Execute DB tx
	persistCtx := context.WithoutCancel(ctx)
	txErr := s.store.ExecTx(persistCtx, func(qtx *repository.Queries) error {
		// Create user
		userRow, err := qtx.UserCreate(persistCtx, repository.UserCreateParams{
			Email:        r.Email,
			Username:     r.Username,
			PasswordHash: passwordHash,
		})
		if err != nil {
			return err
		}

		// Set display name
		displayName := r.Username
		if r.DisplayName != nil && *r.DisplayName != "" {
			displayName = *r.DisplayName
		}

		// Create user profile
		userProfileRow, err := qtx.UserProfileCreate(persistCtx, repository.UserProfileCreateParams{
			UserID:      userRow.ID,
			DisplayName: displayName,
		})
		if err != nil {
			return err
		}

		// Create register event
		err = worker.EmitRegister(persistCtx, qtx, worker.RegisterEventPayload{
			UserID: userRow.ID,
		})
		if err != nil {
			return err
		}

		result = RegisterResult{
			User:        user.NewView(userRow),
			UserProfile: user.NewProfileView(userProfileRow),
		}

		return nil
	})

	// Handle tx errors
	if txErr != nil {
		return RegisterResult{}, apperr.NewDBError(txErr)
	}

	// Return result
	return result, nil
}
