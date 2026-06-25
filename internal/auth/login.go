package auth

import (
	"bonfire-api/internal/apperr"
	"bonfire-api/internal/cache"
	"bonfire-api/internal/crypto"
	"bonfire-api/internal/httpio"
	"bonfire-api/internal/sanitize"
	"bonfire-api/internal/token"
	"context"
	"log/slog"
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
	return apperr.New(
		apperr.CodeUnauthorized,
		errCredentialsInvalid,
		apperr.WithInvalidParams([]apperr.InvalidParam{
			{Name: "email", Reason: errCredentialsInvalid},
			{Name: "password", Reason: errCredentialsInvalid},
		}),
	)
}

func newAccountLockedError() error {
	return apperr.New(
		apperr.CodeForbidden,
		errAccountLocked,
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
	// Check if account locked
	lockoutKey := cache.LoginLockoutKey(r.Email)
	isLocked, err := s.cache.Exists(ctx, lockoutKey)
	if err != nil {
		// Fail open cache lookup
		slog.ErrorContext(ctx, "login lockout cache lookup failed", "error", err, "email", r.Email)
	} else if isLocked {
		// Prevent login
		return LoginResult{}, newAccountLockedError()
	}

	// Get user auth
	userAuth, err := s.user.GetAuthByEmail(ctx, r.Email)
	if err != nil {
		// Handle not found
		if apperr.IsNotFound(err) {
			crypto.DummyVerifyPassword()
			return LoginResult{}, newLoginCredentialsError()
		}

		// Handle rest
		return LoginResult{}, err
	}

	// Check password
	if err = crypto.VerifyPassword(userAuth.PasswordHash, r.Password); err != nil {
		// Generate lock context to ensure cache write
		persistCtx := context.WithoutCancel(ctx)

		failureKey := cache.LoginFailuresKey(r.Email)
		attempts, incrErr := s.cache.Increment(persistCtx, failureKey, 1*time.Hour)
		if incrErr != nil {
			// Fail open cache lookup
			slog.ErrorContext(ctx, "failed to increment login failures", "error", incrErr, "email", r.Email)
		} else if attempts >= loginMaxAttempts {
			// Trigger account lockout
			if lockErr := s.cache.Set(persistCtx, lockoutKey, true, loginLockoutDuration); lockErr != nil {
				slog.ErrorContext(ctx, "failed to set login lockout", "error", lockErr, "email", r.Email)
			}
			return LoginResult{}, newAccountLockedError()
		}

		return LoginResult{}, newLoginCredentialsError()
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

	// Clear fail count
	persistCtx := context.WithoutCancel(ctx)
	if delErr := s.cache.Delete(persistCtx, cache.LoginFailuresKey(r.Email)); delErr != nil {
		slog.WarnContext(ctx, "failed to clear login failures cache", "error", delErr, "email", r.Email)
	}

	// Create user session
	userSession, err := s.CreateUserSession(persistCtx, CreateUserSessionParams{
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

	// Add session to cache
	sessionKey := cache.UserSessionKey(userSessionID.String())
	_ = s.cache.Set(persistCtx, sessionKey, userSession, token.RefreshTokenTTL)

	// Return the tokens
	return LoginResult{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
	}, nil
}
