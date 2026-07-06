package auth

import (
	"bonfire-api/internal/apperr"
	"bonfire-api/internal/crypto"
	"bonfire-api/internal/httpio"
	"bonfire-api/internal/session"
	"bonfire-api/internal/token"
	"context"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// --- LOGIN CONSTANTS ---

// Messages
const (
	msgLoginSuccess = "login_success"
)

// Errors
const (
	errCredentialsInvalid = "Invalid credentials."
	errAccountInactive    = "Account inactive."
	errAccountLocked      = "Account locked from too many failed attempts. Please try again later."
)

// Values
const (
	loginMaxAttempts     = 5
	loginLockoutDuration = 15 * time.Minute
)

// --- LOGIN ERRORS ---

func newLoginCredentialsError() error {
	return apperr.NewUnauthorized(
		nil,
		errCredentialsInvalid,
		apperr.Param("email", errCredentialsInvalid),
		apperr.Param("password", errCredentialsInvalid),
	)
}

func newAccountLockedError() error {
	return apperr.NewForbidden(
		nil,
		errAccountLocked,
	)
}

// --- LOGIN TYPES ---

type LoginReq struct {
	Email    string `json:"email" mod:"email" validate:"identity_email"`
	Password string `json:"password" validate:"identity_password"`
}

type LoginParams struct {
	Email    string
	Password string
	Meta     httpio.ClientMeta
}

type LoginResult struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type LoginRes struct {
	AccessToken string `json:"access_token"`
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) error {
	req, err := httpio.BindJSON[LoginReq](w, r)
	if err != nil {
		return err
	}

	clientMeta, err := httpio.GetMeta(r.Context())
	if err != nil {
		return err
	}

	tokens, err := h.service.Login(r.Context(), LoginParams{
		Email:    req.Email,
		Password: req.Password,
		Meta:     clientMeta,
	})
	if err != nil {
		return err
	}

	// httpio.SetCookieRefreshToken(w, tokens.RefreshToken)
	httpio.RespondOK(w, r, LoginRes{AccessToken: tokens.AccessToken})
	return nil
}

func (s *Service) Login(ctx context.Context, r LoginParams) (LoginResult, error) {
	userAuth, err := s.user.GetAuthByEmail(ctx, r.Email)
	if err != nil {
		if apperr.IsNotFound(err) {
			return LoginResult{}, newLoginCredentialsError()
		}
		return LoginResult{}, err
	}

	if err = crypto.ComparePassword(userAuth.PasswordHash, r.Password); err != nil {
		return LoginResult{}, newLoginCredentialsError()
	}

	userSessionID, err := uuid.NewV7()
	if err != nil {
		return LoginResult{}, apperr.NewInternal(err, "")
	}

	tokenPair, err := s.token.GenerateTokenPair(userAuth.ID, userSessionID)
	if err != nil {
		return LoginResult{}, apperr.NewInternal(err, "")
	}

	persistCtx := context.WithoutCancel(ctx)

	_, err = s.session.Create(persistCtx, session.CreateParams{
		ID:           userSessionID,
		UserID:       userAuth.ID,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresAt:    time.Now().Add(token.RefreshTokenTTL),
	})
	if err != nil {
		return LoginResult{}, err
	}

	return LoginResult{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
	}, nil
}
