package auth

import (
	"context"
	"errors"
	"log/slog"
	"net/netip"
	"time"

	"bonfire-api/internal/apperr"
	"bonfire-api/internal/crypto"
	"bonfire-api/internal/sanitize"
	"bonfire-api/internal/session"
	"bonfire-api/internal/user"

	"github.com/google/uuid"
)

// TODO: Move to configuration
const (
	loginTimingWindow    = 35 * time.Millisecond
	loginMaxAttempts     = 5
	loginFailureTTL      = 1 * time.Hour
	loginLockoutDuration = 15 * time.Minute
)

type LoginParams struct {
	Email     string
	Password  string
	IP        netip.Addr
	UserAgent string
	OS        string
	Browser   string
}

type LoginResult struct {
	AccessToken           string
	RefreshToken          string
	RefreshTokenExpiresAt time.Time
}

// auth/login.go
func (s *Service) Login(ctx context.Context, p LoginParams) (LoginResult, error) {
	defer crypto.ConstantWindow(loginTimingWindow)()

	email, err := user.NewEmail(sanitize.Email(p.Email))
	if err != nil || !email.IsValid() {
		return LoginResult{}, apperr.NewInvalidArgument(
			errors.New("invalid email address"),
			apperr.WithMsg("Invalid email address"),
		)
	}

	isLocked, err := s.shield.IsLocked(ctx, email.String())
	if err != nil {
		slog.ErrorContext(ctx, "login lockout cache lookup failed", "error", err, "email", email.String())
	} else if isLocked {
		return LoginResult{}, newLockedError()
	}

	userRow, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		if apperr.IsNotFound(err) {
			crypto.CompareDummyPassword(p.Password)
			return LoginResult{}, s.handleInvalidPassword(ctx, email.String())
		}
		return LoginResult{}, err
	}

	if err = crypto.ComparePassword(userRow.PasswordHash(), p.Password); err != nil {
		return LoginResult{}, s.handleInvalidPassword(ctx, email.String())
	}

	sessionID, err := uuid.NewV7()
	if err != nil {
		return LoginResult{}, apperr.NewInternal(err)
	}

	tokenPair, err := s.tokens.GeneratePair(userRow.ID(), sessionID)
	if err != nil {
		return LoginResult{}, apperr.NewInternal(err)
	}

	tokenHash, err := session.NewRefreshTokenHash(crypto.HashToken(tokenPair.Refresh))
	if err != nil {
		return LoginResult{}, apperr.NewInternal(err)
	}

	newSession, err := session.New(
		sessionID,
		userRow.ID(),
		tokenHash,
		tokenPair.RefreshExpiresAt,
		p.IP,
		p.UserAgent,
		p.OS,
		p.Browser,
	)
	if err != nil {
		return LoginResult{}, apperr.NewInternal(err)
	}

	persistCtx := context.WithoutCancel(ctx)

	if err := s.sessions.Create(persistCtx, newSession); err != nil {
		return LoginResult{}, err
	}

	if err := s.sessionCache.Set(persistCtx, newSession); err != nil {
		slog.WarnContext(persistCtx, "failed to update session cache during login", "error", err, "session_id", newSession.ID())
	}

	if err := s.shield.ResetFailures(persistCtx, email.String()); err != nil {
		slog.WarnContext(persistCtx, "failed to reset login failure count", "error", err, "email", email.String())
	}

	return LoginResult{
		AccessToken:           tokenPair.Access,
		RefreshToken:          tokenPair.Refresh,
		RefreshTokenExpiresAt: tokenPair.RefreshExpiresAt,
	}, nil
}

func (s *Service) handleInvalidPassword(ctx context.Context, email string) error {
	persistCtx := context.WithoutCancel(ctx)

	attempts, err := s.shield.IncrementFailures(persistCtx, email, loginFailureTTL)
	if err != nil {
		slog.ErrorContext(persistCtx, "failed to increment login failures", "error", err, "email", email)
		return newCredentialsError()
	}

	if attempts >= loginMaxAttempts {
		if err := s.shield.Lockout(persistCtx, email, loginLockoutDuration); err != nil {
			slog.ErrorContext(persistCtx, "failed to set login lockout", "error", err, "email", email)
		}
		return newLockedError()
	}

	return newCredentialsError()
}

func newLockedError() error {
	return apperr.NewPermissionDenied(
		errors.New("account locked"),
	)
}

func newCredentialsError() error {
	return apperr.NewPermissionDenied(
		errors.New("invalid credentials"),
		// "Invalid email or password.",
		// apperr.Param("email", "invalid credentials"),
		// apperr.Param("password", "invalid credentials"),
	)
}
