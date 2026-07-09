package auth

import (
	"bonfire-api/internal/apperr"
	"bonfire-api/internal/crypto"
	"bonfire-api/internal/httpio"
	"bonfire-api/internal/repository"
	"bonfire-api/internal/token"
	"bonfire-api/internal/user"
	"context"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type RegisterReq struct {
	Email       string  `json:"email" mod:"email" validate:"identity_email"`
	Username    string  `json:"username" mod:"text" validate:"identity_username"`
	DisplayName *string `json:"display_name" mod:"text" validate:"profile_display_name"`
	Password    string  `json:"password" validate:"identity_password"`
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
	availability, err := s.user.CheckAvailability(ctx, user.CheckAvailabilityParams{
		Email:    r.Email,
		Username: r.Username,
	})
	if err != nil {
		return RegisterResult{}, err
	}

	if !availability.Email || !availability.Username {
		return RegisterResult{}, newRegisterConflictError(availability)
	}

	passwordHash, err := crypto.HashPassword(r.Password)
	if err != nil {
		return RegisterResult{}, apperr.NewInternal(err, "")
	}

	userID, err := uuid.NewV7()
	if err != nil {
		return RegisterResult{}, apperr.NewInternal(err, "")
	}

	sessionID, err := uuid.NewV7()
	if err != nil {
		return RegisterResult{}, apperr.NewInternal(err, "")
	}

	tokenPair, err := s.token.GeneratePair(token.PairParams{
		UserID:    userID,
		SessionID: sessionID,
	})
	if err != nil {
		return RegisterResult{}, apperr.NewInternal(err, "")
	}

	hashedRefreshToken := crypto.HashToken(tokenPair.RefreshToken)

	displayName := r.Username
	if r.DisplayName != nil && *r.DisplayName != "" {
		displayName = *r.DisplayName
	}

	txErr := s.store.ExecTx(ctx, func(qtx *repository.Queries) error {
		persistCtx := context.WithoutCancel(ctx)

		_, err := qtx.UserCreate(persistCtx, repository.UserCreateParams{
			ID:           pgtype.UUID{Bytes: userID, Valid: true},
			Email:        r.Email,
			Username:     r.Username,
			PasswordHash: passwordHash,
		})
		if err != nil {
			return repository.NewError(err, repository.ScopeUser)
		}

		_, err = qtx.UserProfileCreate(persistCtx, repository.UserProfileCreateParams{
			UserID:      pgtype.UUID{Bytes: userID, Valid: true},
			DisplayName: displayName,
		})
		if err != nil {
			return repository.NewError(err, repository.ScopeUserProfile)
		}

		_, err = qtx.SessionCreate(persistCtx, repository.SessionCreateParams{
			ID:               pgtype.UUID{Bytes: sessionID, Valid: true},
			UserID:           pgtype.UUID{Bytes: userID, Valid: true},
			RefreshTokenHash: hashedRefreshToken,
			ExpiresAt:        pgtype.Timestamptz{Time: tokenPair.RefreshTokenExpiresAt, Valid: true},
		})
		if err != nil {
			return repository.NewError(err, repository.ScopeSession)
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

func newRegisterConflictError(r user.CheckAvailabilityResult) error {
	var params []apperr.InvalidParam

	if !r.Email {
		params = append(params, apperr.InvalidParam{Name: "email", Reason: "This email is already taken."})
	}
	if !r.Username {
		params = append(params, apperr.InvalidParam{Name: "username", Reason: "This username is already taken."})
	}

	return apperr.NewInvalidInput(
		nil,
		"Validation failed for the request.",
		apperr.Params(params),
	)
}
