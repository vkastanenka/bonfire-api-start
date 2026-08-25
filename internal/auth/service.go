package auth

import (
	"bonfire-api/internal/crypto"
	"bonfire-api/internal/errs"
	"bonfire-api/internal/sanitize"
	"bonfire-api/internal/session"
	"bonfire-api/internal/user"
	"context"
	"errors"
	"log/slog"
	"net/netip"
	"time"

	"github.com/google/uuid"
)

type Service struct {
	userRepo      UserRepository
	outboxRepo    OutboxRepository
	sessionRepo   SessionRepository
	ticketCache   TicketCache
	tokenProvider TokenProvider
	tx            TX
}

func NewService(
	userRepo UserRepository,
	outboxRepo OutboxRepository,
	sessionRepo SessionRepository,
	ticketCache TicketCache,
	tokenProvider TokenProvider,
	tx TX,
) *Service {
	return &Service{
		userRepo:      userRepo,
		outboxRepo:    outboxRepo,
		sessionRepo:   sessionRepo,
		ticketCache:   ticketCache,
		tokenProvider: tokenProvider,
		tx:            tx,
	}
}

const (
	forgotPasswordTimingWindow = 35 * time.Millisecond
)

func (s *Service) ForgotPassword(ctx context.Context, rawEmail string) error {
	defer crypto.ConstantWindow(forgotPasswordTimingWindow)()

	email, err := user.ParseRequiredEmail("email", rawEmail)
	if err != nil || !email.IsValid() {
		return ErrEmailInvalid()
	}

	userRow, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		if errs.IsNotFound(err) {
			return nil
		}
		return err
	}

	t, _, err := s.tokenProvider.GeneratePasswordReset(userRow.ID().UUID())
	if err != nil {
		return err
	}

	_, err = s.outboxRepo.Publish(ctx, EventForgotPassword, ForgotPasswordPayload{
		Email: userRow.Email().String(),
		Token: t,
	})
	if err != nil {
		return err
	}

	return nil
}

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
		return LoginResult{}, errs.InvalidArgument("Invalid email address.").
			FieldViolation("email", "Must be a valid email address.", "INVALID_EMAIL").
			Wrap(errors.New("invalid email address"))
	}

	isLocked, err := s.shield.IsLocked(ctx, email.String())
	if err != nil {
		slog.ErrorContext(ctx, "login lockout cache lookup failed", "error", err, "email", email.String())
	} else if isLocked {
		return LoginResult{}, newLockedError()
	}

	userRow, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		if errs.IsNotFound(err) {
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
		return LoginResult{}, errs.Internal("failed to generate session ID").Wrap(err)
	}

	tokenPair, err := s.tokens.GeneratePair(userRow.ID(), sessionID)
	if err != nil {
		return LoginResult{}, errs.Internal("failed to generate token pair").Wrap(err)
	}

	tokenHash, err := session.NewRefreshTokenHash(crypto.HashToken(tokenPair.Refresh))
	if err != nil {
		return LoginResult{}, errs.Internal("failed to hash refresh token").Wrap(err)
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
		return LoginResult{}, errs.Internal("failed to create session entity").Wrap(err)
	}

	persistCtx := context.WithoutCancel(ctx)

	if err := s.sessions.Create(persistCtx, newSession); err != nil {
		return LoginResult{}, err
	}

	if err := s.sessionStore.Set(persistCtx, newSession); err != nil {
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
