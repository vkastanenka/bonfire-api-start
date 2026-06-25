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
	MsgLoginSuccess = "login_success"
)

// Errors
const (
	ErrCredentialsInvalid = "Invalid credentials."
	ErrCreatingSession    = "Unable to create session."
)

// --- LOGIN ERRORS ---

func NewLoginCredentialsError() error {
	return apperr.New(
		apperr.CodeUnauthenticated,
		ErrCredentialsInvalid,
		apperr.WithInvalidParams([]apperr.InvalidParam{
			{Name: "email", Reason: ErrCredentialsInvalid},
			{Name: "password", Reason: ErrCredentialsInvalid},
		}),
	)
}

// --- LOGIN TYPES ---

type LoginReq struct {
	Email    string `json:"email" validate:"required,email,max=255"`
	Password string `json:"password" validate:"required,min=12,max=128"`
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
	httpio.RespondOK(w, r, LoginRes{AccessToken: tokens.AccessToken}, MsgLoginSuccess)

	return nil
}

// --- LOGIN SERVICE ---

// Login
func (s *Service) Login(ctx context.Context, r LoginParams) (LoginResult, error) {
	// Fetch user credentials
	userAuth, err := s.user.GetAuthByEmail(ctx, r.Email)
	if err != nil {
		crypto.DummyVerify() // Simulates finding a user
		return LoginResult{}, err
	}

	// Check password
	if err = crypto.VerifyPassword(userAuth.PasswordHash, r.Password); err != nil {
		return LoginResult{}, NewLoginCredentialsError()
	}

	// Generate refresh token
	sessionID, err := uuid.NewV7()
	if err != nil {
		return LoginResult{}, apperr.New(apperr.CodeInternal, apperr.CodeInternal.Title(), apperr.WithErr(err))
	}

	// Generate token pair
	tokenPair, err := s.token.GenerateTokenPair(userAuth.ID, string(userAuth.Role), userAuth.VerifiedAt != nil, userAuth.SecurityVersion, sessionID)
	if err != nil {
		return LoginResult{}, apperr.New(apperr.CodeInternal, apperr.CodeInternal.Title(), apperr.WithErr(err))
	}

	// Create user session
	_, err = s.CreateUserSession(ctx, CreateUserSessionParams{
		ID:           sessionID,
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
