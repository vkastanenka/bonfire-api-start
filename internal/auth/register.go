package auth

import (
	"bonfire-api/internal/apperr"
	"bonfire-api/internal/crypto"
	"bonfire-api/internal/httpio"
	"bonfire-api/internal/repository"
	"bonfire-api/internal/session"
	"bonfire-api/internal/user"
	"context"
	"net/http"
	"time"

	"github.com/google/uuid"
)

type RegisterReq struct {
	Email       string  `json:"email" mod:"email" validate:"identity_email"`
	DisplayName *string `json:"display_name" mod:"text" validate:"profile_display_name"`
	Username    string  `json:"username" mod:"text" validate:"identity_username"`
	Password    string  `json:"password" validate:"security_password"`
}

type RegisterRes struct {
	AccessToken string `json:"access_token"`
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

	httpio.SetCookieRefreshToken(w, httpio.SetCookieRefreshTokenParams{
		Token:   data.RefreshToken,
		Expires: data.RefreshTokenExpiresAt,
	})
	httpio.RespondCreated(w, r, RegisterRes{AccessToken: data.AccessToken})
	return nil
}

type RegisterParams struct {
	Email       string
	Username    string
	DisplayName *string
	Password    string
}

type RegisterResult struct {
	AccessToken           string
	RefreshToken          string
	RefreshTokenExpiresAt time.Time
}

func (s *Service) Register(ctx context.Context, r RegisterParams) (RegisterResult, error) {
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

	userID, err := uuid.NewV7()
	if err != nil {
		return RegisterResult{}, apperr.NewInternal(err, "")
	}

	sessionID, err := uuid.NewV7()
	if err != nil {
		return RegisterResult{}, apperr.NewInternal(err, "")
	}

	tokenPair, err := s.token.GenerateTokenPair(userID, sessionID)
	if err != nil {
		return RegisterResult{}, apperr.NewInternal(err, "")
	}

	hashedRefreshToken := crypto.HashToken(tokenPair.RefreshToken)

	displayName := r.Username
	if r.DisplayName != nil && *r.DisplayName != "" {
		displayName = *r.DisplayName
	}

	persistCtx := context.WithoutCancel(ctx)
	txErr := s.store.ExecTx(persistCtx, func(qtx *repository.Queries) error {
		txUserService := user.NewService(qtx)
		txSessionService := session.NewService(qtx)

		_, err := txUserService.Create(persistCtx, user.CreateParams{
			ID:       &userID,
			Email:    r.Email,
			Username: r.Username,
			Password: passwordHash,
		})
		if err != nil {
			return err
		}

		_, err = txUserService.CreateProfile(persistCtx, user.CreateProfileParams{
			UserID:      userID,
			DisplayName: displayName,
		})
		if err != nil {
			return err
		}

		_, err = txSessionService.Create(persistCtx, session.CreateParams{
			ID:           &sessionID,
			UserID:       userID,
			RefreshToken: hashedRefreshToken,
			ExpiresAt:    tokenPair.RefreshTokenExpiresAt,
		})
		if err != nil {
			return err
		}

		return nil
	})

	if txErr != nil {
		return RegisterResult{}, txErr
	}

	return RegisterResult{
		AccessToken:           tokenPair.AccessToken,
		RefreshToken:          tokenPair.RefreshToken,
		RefreshTokenExpiresAt: tokenPair.RefreshTokenExpiresAt,
	}, nil
}
