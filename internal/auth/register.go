package auth

import (
	"bonfire-api/internal/apperr"
	"bonfire-api/internal/crypto"
	"bonfire-api/internal/httpio"
	"bonfire-api/internal/repository"
	"bonfire-api/internal/user"
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
)

// --- REFRESH ERRORS ---

func NewRegisterConflictError(emailAvailable, usernameAvailable bool) error {
	var opts []apperr.ErrorOption

	if !emailAvailable {
		opts = append(opts, apperr.Param("email", errEmailTaken))
	}
	if !usernameAvailable {
		opts = append(opts, apperr.Param("username", errUsernameTaken))
	}

	return apperr.NewConflict(nil, errCredentialsTaken, opts...)
}

type RegisterReq struct {
	Email       string  `json:"email" mod:"email" validate:"identity_email"`
	DisplayName *string `json:"display_name" mod:"text" validate:"profile_display_name"`
	Username    string  `json:"username" mod:"text" validate:"identity_username"`
	Password    string  `json:"password" validate:"security_password"`
}

type RegisterParams struct {
	Email       string
	Username    string
	DisplayName *string
	Password    string
}

type RegisterResult struct {
	User    user.View        `json:"user"`
	Profile user.ProfileView `json:"user_profile"`
}

// --- REGISTER HANDLER ---

// Register
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) error {
	// Bind JSON
	req, err := httpio.BindJSON[RegisterReq](w, r)
	if err != nil {
		return err
	}

	// Register user
	data, err := h.service.Register(r.Context(), RegisterParams{
		Email:       req.Email,
		Username:    req.Username,
		DisplayName: req.DisplayName,
		Password:    req.Password,
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
		return RegisterResult{}, repository.NewError(err, repository.ScopeUser)
	}

	// Cleanly handle conflict
	if !availability.EmailAvailable || !availability.UsernameAvailable {
		return RegisterResult{}, NewRegisterConflictError(availability.EmailAvailable, availability.UsernameAvailable)
	}

	// Hash password
	hashedPasswordBytes, err := crypto.HashPassword(r.Password)
	if err != nil {
		return RegisterResult{}, apperr.NewInternal(err, "")
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
			return repository.NewError(err, repository.ScopeUser)
		}

		// Set display name
		displayName := r.Username
		if r.DisplayName != nil && *r.DisplayName != "" {
			displayName = *r.DisplayName
		}

		// Create profile
		userProfileRow, err := qtx.UserProfileCreate(persistCtx, repository.UserProfileCreateParams{
			UserID:      userRow.ID,
			DisplayName: displayName,
		})
		if err != nil {
			return repository.NewError(err, repository.ScopeProfile)
		}

		result = RegisterResult{
			User:    user.NewView(userRow),
			Profile: user.NewProfileView(userProfileRow),
		}

		return nil
	})

	// Handle tx errors
	if txErr != nil {
		return RegisterResult{}, txErr
	}

	// Return result
	return result, nil
}
