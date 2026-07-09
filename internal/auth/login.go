package auth

import (
	"bonfire-api/internal/apperr"
	"bonfire-api/internal/cache"
	"bonfire-api/internal/crypto"
	"bonfire-api/internal/httpio"
	"bonfire-api/internal/repository"
	"bonfire-api/internal/session"
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// TODO: Move to config?
const (
	loginMaxAttempts     = 5
	loginLockoutDuration = 15 * time.Minute
)

type LoginRequest struct {
	Email    string `json:"email" mod:"email" validate:"identity_email"`
	Password string `json:"password" validate:"identity_password"`
}

type LoginResponse struct {
	AccessToken string `json:"access_token"`
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) error {
	req, err := httpio.BindJSON[LoginRequest](w, r)
	if err != nil {
		return err
	}

	clientMeta, err := httpio.GetMeta(r.Context())
	if err != nil {
		return err
	}

	data, err := h.service.Login(r.Context(), LoginParams{
		Email:    req.Email,
		Password: req.Password,
		Meta:     clientMeta,
	})
	if err != nil {
		return err
	}

	httpio.SetCookieRefreshToken(w, httpio.SetCookieRefreshTokenParams{
		Token:   data.RefreshToken,
		Expires: data.RefreshTokenExpiresAt,
	})
	httpio.RespondOK(w, r, LoginResponse{AccessToken: data.AccessToken})

	return nil
}

type LoginParams struct {
	Email    string
	Password string
	Meta     httpio.ClientMeta
}

type LoginResult struct {
	AccessToken           string
	RefreshToken          string
	RefreshTokenExpiresAt time.Time
}

func (s *Service) Login(ctx context.Context, r LoginParams) (LoginResult, error) {
	lockoutKey := cache.AuthLoginLockoutKey(r.Email)
	if isLocked, err := s.cache.Exists(ctx, lockoutKey); err == nil && isLocked {
		return LoginResult{}, newLockedError()
	} else if err != nil {
		slog.ErrorContext(ctx, "login lockout cache lookup failed", "error", err, "email", r.Email)
	}

	userAuth, err := s.user.GetAuthByEmail(ctx, r.Email)
	if err != nil {
		if repository.IsNotFoundError(err) {
			return LoginResult{}, newCredentialsError()
		}
		return LoginResult{}, err
	}

	if err = crypto.ComparePassword(userAuth.PasswordHash, r.Password); err != nil {
		return LoginResult{}, s.handleInvalidPassword(ctx, r.Email, lockoutKey)
	}

	userSessionID, err := uuid.NewV7()
	if err != nil {
		return LoginResult{}, apperr.NewInternal(err, "")
	}

	tokenPair, err := s.token.GenerateTokenPair(userAuth.ID, userSessionID)
	if err != nil {
		return LoginResult{}, apperr.NewInternal(err, "")
	}

	hashedRefreshToken := crypto.HashToken(tokenPair.RefreshToken)

	persistCtx := context.WithoutCancel(ctx)

	_, err = s.session.Create(persistCtx, session.CreateParams{
		ID:               &userSessionID,
		UserID:           userAuth.ID,
		RefreshTokenHash: hashedRefreshToken,
		ExpiresAt:        tokenPair.RefreshTokenExpiresAt,
	})
	if err != nil {
		return LoginResult{}, err
	}

	return LoginResult{
		AccessToken:           tokenPair.AccessToken,
		RefreshToken:          tokenPair.RefreshToken,
		RefreshTokenExpiresAt: tokenPair.RefreshTokenExpiresAt,
	}, nil
}

func (s *Service) handleInvalidPassword(ctx context.Context, email string, lockoutKey string) error {
	persistCtx := context.WithoutCancel(ctx)
	failureKey := cache.AuthLoginFailuresKey(email)

	attempts, err := s.cache.Increment(persistCtx, failureKey, 1*time.Hour)
	if err != nil {
		slog.ErrorContext(ctx, "failed to increment login failures", "error", err, "email", email)
		return newCredentialsError()
	}

	if attempts >= loginMaxAttempts {
		if err := s.cache.Set(persistCtx, lockoutKey, true, loginLockoutDuration); err != nil {
			slog.ErrorContext(ctx, "failed to set login lockout", "error", err, "email", email)
		}
		return newLockedError()
	}

	return newCredentialsError()
}

func newLockedError() error {
	return apperr.NewForbidden(nil, "Account locked from too many failed attempts. Please try again later.")
}

func newCredentialsError() error {
	return apperr.NewUnauthorized(
		nil,
		"",
		apperr.Param("email", apperr.CodeBadRequest.Detail()),
		apperr.Param("password", apperr.CodeBadRequest.Detail()),
	)
}
