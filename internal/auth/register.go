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

type RegisterReq struct {
	Email       string  `json:"email" mod:"email" validate:"identity_email"`
	DisplayName *string `json:"display_name" mod:"text" validate:"profile_display_name"`
	Username    string  `json:"username" mod:"text" validate:"identity_username"`
	Password    string  `json:"password" validate:"security_password"`
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) error {
	req, err := httpio.BindJSON[RegisterReq](w, r)
	if err != nil {
		return err
	}

	data, err := h.service.Register(r.Context(), RegisterParams{
		Email:       req.Email,
		Username:    req.Username,
		DisplayName: req.DisplayName,
		Password:    req.Password,
	})
	if err != nil {
		return err
	}

	httpio.RespondCreated(w, r, data)
	return nil
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

func (s *Service) Register(ctx context.Context, r RegisterParams) (RegisterResult, error) {
	var result RegisterResult

	availability, err := s.store.UserCheckAvailability(ctx, repository.UserCheckAvailabilityParams{
		Email:    r.Email,
		Username: r.Username,
	})
	if err != nil {
		return RegisterResult{}, repository.NewError(err, repository.ScopeUser)
	}

	if !availability.EmailAvailable || !availability.UsernameAvailable {
		var opts []apperr.ErrorOption

		if !availability.EmailAvailable {
			opts = append(opts, apperr.Param("email", "Email unavailable."))
		}
		if !availability.UsernameAvailable {
			opts = append(opts, apperr.Param("username", "Username unavailable."))
		}

		return RegisterResult{}, apperr.NewConflict(nil, "Email and/or username unavailable.", opts...)
	}

	hashedPasswordBytes, err := crypto.HashPassword(r.Password)
	if err != nil {
		return RegisterResult{}, apperr.NewInternal(err, "")
	}
	passwordHash := string(hashedPasswordBytes)

	persistCtx := context.WithoutCancel(ctx)
	txErr := s.store.ExecTx(persistCtx, func(qtx *repository.Queries) error {
		userRow, err := qtx.UserCreate(persistCtx, repository.UserCreateParams{
			Email:        r.Email,
			Username:     r.Username,
			PasswordHash: passwordHash,
		})
		if err != nil {
			return repository.NewError(err, repository.ScopeUser)
		}

		displayName := r.Username
		if r.DisplayName != nil && *r.DisplayName != "" {
			displayName = *r.DisplayName
		}

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

	if txErr != nil {
		return RegisterResult{}, txErr
	}

	return result, nil
}
