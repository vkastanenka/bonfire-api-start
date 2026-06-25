package auth

import (
	"bonfire-api/internal/apperr"
	"bonfire-api/internal/crypto"
	"bonfire-api/internal/httpio"
	"bonfire-api/internal/sanitize"
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
)

// --- LOGIN ERRORS ---

func NewLoginCredentialsError() error {
	return apperr.New(
		apperr.CodeUnauthorized,
		errCredentialsInvalid,
		apperr.WithInvalidParams([]apperr.InvalidParam{
			{Name: "email", Reason: errCredentialsInvalid},
			{Name: "password", Reason: errCredentialsInvalid},
		}),
	)
}

// --- LOGIN TYPES ---

type LoginReq struct {
	Email    string `json:"email" validate:"auth_email"`
	Password string `json:"password" validate:"auth_password"`
}

func (r *LoginReq) Sanitize() {
	r.Email = sanitize.SanitizeEmail(r.Email)
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

// --- LOGIN HANDLER ---

// Login
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) error {
	// Get JSON
	req, err := httpio.BindJSON[LoginReq](w, r, h.validator)
	if err != nil {
		return err
	}

	// Get client meta
	clientMeta := httpio.GetClientMeta(r, false)

	// Login user, get tokens
	tokens, err := h.service.Login(r.Context(), LoginParams{
		Email:    req.Email,
		Password: req.Password,
		Meta:     clientMeta,
	})
	if err != nil {
		return err
	}

	// Repond with tokens
	httpio.SetRefreshTokenCookie(w, tokens.RefreshToken)
	httpio.RespondOK(w, r, LoginRes{AccessToken: tokens.AccessToken}, msgLoginSuccess)

	return nil
}

// --- LOGIN SERVICE ---

// Login
func (s *Service) Login(ctx context.Context, r LoginParams) (LoginResult, error) {
	// Get user auth
	userAuth, err := s.user.GetAuthByEmail(ctx, r.Email)
	if err != nil {
		// Handle not found
		if apperr.IsNotFound(err) {
			crypto.DummyVerifyPassword()
			return LoginResult{}, NewLoginCredentialsError()
		}

		// Handle rest
		return LoginResult{}, err
	}

	// Check password
	if err = crypto.VerifyPassword(userAuth.PasswordHash, r.Password); err != nil {
		return LoginResult{}, NewLoginCredentialsError()
	}

	// Check status
	if !userAuth.IsActive() {
		return LoginResult{}, apperr.New(apperr.CodeForbidden, errAccountInactive)
	}

	// Generate refresh token
	userSessionID, err := uuid.NewV7()
	if err != nil {
		return LoginResult{}, apperr.NewInternal(err)
	}

	// Generate token pair
	tokenPair, err := s.token.GenerateTokenPair(userAuth.ID, string(userAuth.Role), userAuth.VerifiedAt != nil, userAuth.SecurityVersion, userSessionID)
	if err != nil {
		return LoginResult{}, apperr.NewInternal(err)
	}

	// Create user session
	_, err = s.CreateUserSession(ctx, CreateUserSessionParams{
		ID:           userSessionID,
		UserID:       userAuth.ID,
		RefreshToken: tokenPair.RefreshToken,
		UserAgent:    r.Meta.UserAgent,
		ClientIP:     r.Meta.IP,
		IsBlocked:    false,
		ExpiresAt:    time.Now().Add(token.RefreshTokenTTL),
	})
	if err != nil {
		return LoginResult{}, err
	}

	// Return the tokens
	return LoginResult{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
	}, nil
}
