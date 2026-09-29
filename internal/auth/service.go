package auth

import (
	"context"
	"log/slog"
	"time"

	"bonfire-api/internal/appctx"
	"bonfire-api/internal/httpio"
	"bonfire-api/internal/outbox"
	"bonfire-api/internal/pkg/crypto"
	"bonfire-api/internal/pkg/errs"
	"bonfire-api/internal/session"
	"bonfire-api/internal/token"
	"bonfire-api/internal/user"

	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"
)

type Service struct {
	userCache      UserCache
	userRepo       UserRepository
	cachedUserRepo CachedUserRepository
	sessionCache   SessionCache
	sessionRepo    SessionRepository
	outboxRepo     OutboxRepository
	ticketCache    WSTicketCache
	tokenCache     TokenCache
	tokenProvider  TokenProvider
	tx             TX
}

func NewService(
	userCache UserCache,
	userRepo UserRepository,
	cachedUserRepo CachedUserRepository,
	sessionCache SessionCache,
	sessionRepo SessionRepository,
	outboxRepo OutboxRepository,
	ticketCache WSTicketCache,
	tokenCache TokenCache,
	tokenProvider TokenProvider,
	tx TX,
) *Service {
	return &Service{
		userCache:      userCache,
		userRepo:       userRepo,
		cachedUserRepo: cachedUserRepo,
		sessionCache:   sessionCache,
		sessionRepo:    sessionRepo,
		outboxRepo:     outboxRepo,
		ticketCache:    ticketCache,
		tokenCache:     tokenCache,
		tokenProvider:  tokenProvider,
		tx:             tx,
	}
}

type LoginParams struct {
	Email    string
	Password string
}

type LoginResult struct {
	AccessToken           string
	RefreshToken          string
	RefreshTokenExpiresAt time.Time
}

func (s *Service) Login(ctx context.Context, p LoginParams) (LoginResult, error) {
	ctxMeta, err := appctx.GetMeta(ctx)
	if err != nil {
		return LoginResult{}, err
	}

	u, err := s.userRepo.GetByEmail(ctx, p.Email)
	if err != nil {
		if errs.IsNotFound(err) {
			crypto.CompareDummyPassword(p.Password)
			return LoginResult{}, ErrCredentialsInvalid()
		}
		return LoginResult{}, err
	}

	err = crypto.ComparePasswords(u.PasswordHash, p.Password)
	if err != nil {
		return LoginResult{}, ErrCredentialsInvalid()
	}

	now := time.Now()

	var (
		createdSession *session.Session
		newSession     *session.Session
		tokenPair      token.Pair
	)

	if u.IsDisabled() || u.IsScheduledForDeletion() {
		txErr := s.tx.ExecTx(ctx, func(txCtx context.Context) error {
			var err error

			if u.IsScheduledForDeletion() {
				u, err = s.userRepo.SetDeleteSchedule(txCtx, u.ID, nil, nil, now)
			} else {
				u, err = s.userRepo.SetDisabled(txCtx, u.ID, nil, now)
			}
			if err != nil {
				return err
			}

			newSession, tokenPair, err = s.generateSession(ctxMeta, u, now)
			if err != nil {
				return err
			}

			createdSession, err = s.sessionRepo.Create(txCtx, newSession)
			return err
		})

		if txErr != nil {
			return LoginResult{}, txErr
		}
	} else {
		var err error
		newSession, tokenPair, err = s.generateSession(ctxMeta, u, now)
		if err != nil {
			return LoginResult{}, err
		}

		createdSession, err = s.sessionRepo.Create(ctx, newSession)
		if err != nil {
			return LoginResult{}, err
		}
	}

	if err := s.userCache.Set(ctx, u); err != nil {
		slog.WarnContext(ctx, "failed to seed user into cache after login",
			slog.String("user_id", u.ID.String()),
			slog.Any("error", err),
		)
	}

	if err := s.sessionCache.Set(ctx, createdSession); err != nil {
		slog.WarnContext(ctx, "failed to seed new session into cache after login",
			slog.String("user_id", u.ID.String()),
			slog.String("session_id", createdSession.ID.String()),
			slog.Any("error", err),
		)
	}

	return LoginResult{
		AccessToken:           tokenPair.Access,
		RefreshToken:          tokenPair.Refresh,
		RefreshTokenExpiresAt: tokenPair.RefreshExpiresAt,
	}, nil
}

type RegisterParams struct {
	Email       string
	Username    string
	DisplayName *string
	Password    string
}

func (p RegisterParams) ResolveDisplayName() string {
	if p.DisplayName != nil && *p.DisplayName != "" {
		return *p.DisplayName
	}
	return p.Username
}

type RegisterResult struct {
	AccessToken           string
	RefreshToken          string
	RefreshTokenExpiresAt time.Time
}

func (s *Service) Register(ctx context.Context, p RegisterParams) (RegisterResult, error) {
	ctxMeta, err := appctx.GetMeta(ctx)
	if err != nil {
		return RegisterResult{}, err
	}

	var (
		passwordHash   string
		emailAvailable bool
		userAvailable  bool
	)

	g, gCtx := errgroup.WithContext(ctx)

	g.Go(func() error {
		var hErr error
		passwordHash, hErr = crypto.HashPassword(p.Password)
		if hErr != nil {
			return hErr
		}
		return nil
	})

	g.Go(func() error {
		var aErr error
		emailAvailable, userAvailable, aErr = s.userRepo.Availability(gCtx, &p.Email, &p.Username)
		return aErr
	})

	if err := g.Wait(); err != nil {
		return RegisterResult{}, err
	}

	if !emailAvailable || !userAvailable {
		return RegisterResult{}, ErrConflict(emailAvailable, userAvailable)
	}

	userID, err := uuid.NewV7()
	if err != nil {
		return RegisterResult{}, err
	}

	username := p.Username
	if p.DisplayName != nil {
		username = *p.DisplayName
	}

	now := time.Now()
	newUser := user.New(userID, p.Email, p.Username, username, passwordHash, now)
	newSession, tokenPair, err := s.generateSession(ctxMeta, newUser, now)
	if err != nil {
		return RegisterResult{}, err
	}

	evToken, _, err := s.tokenProvider.GenerateEmailVerify(newUser.ID)
	if err != nil {
		return RegisterResult{}, err
	}

	var createdSession *session.Session

	txErr := s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		if newUser, err = s.userRepo.Create(txCtx, newUser); err != nil {
			return err
		}

		var err error
		createdSession, err = s.sessionRepo.Create(txCtx, newSession)
		if err != nil {
			return err
		}

		payload := EventRegisteredPayload{
			Email:       newUser.Email,
			Username:    newUser.Username,
			Token:       evToken,
			RequestedAt: now,
		}

		event, err := outbox.New(
			newUser.ID,
			createdSession.ID,
			appctx.GetTraceID(txCtx),
			nil,
			EventRegistered,
			payload,
			now,
		)
		if err != nil {
			return err
		}

		return s.outboxRepo.Create(txCtx, event)
	})

	if txErr != nil {
		return RegisterResult{}, txErr
	}

	if err := s.userCache.Set(ctx, newUser); err != nil {
		slog.WarnContext(ctx, "failed to seed new user into cache after registration",
			slog.String("user_id", newUser.ID.String()),
			slog.Any("error", err),
		)
	}

	if err := s.sessionCache.Set(ctx, createdSession); err != nil {
		slog.WarnContext(ctx, "failed to seed new session into cache after registration",
			slog.String("user_id", newUser.ID.String()),
			slog.String("session_id", createdSession.ID.String()),
			slog.Any("error", err),
		)
	}

	return RegisterResult{
		AccessToken:           tokenPair.Access,
		RefreshToken:          tokenPair.Refresh,
		RefreshTokenExpiresAt: tokenPair.RefreshExpiresAt,
	}, nil
}

func (s *Service) generateSession(meta httpio.ClientMeta, u *user.User, now time.Time) (*session.Session, token.Pair, error) {
	sessionID, err := uuid.NewV7()
	if err != nil {
		return nil, token.Pair{}, err
	}

	tokenPair, err := s.tokenProvider.GeneratePair(u.ID, sessionID)
	if err != nil {
		return nil, token.Pair{}, errs.Internal("failed to generate token pair").Wrap(err)
	}

	tokenHash := crypto.HashToken(tokenPair.Refresh)

	newSession := session.Reconstitute(
		sessionID,
		u.ID,
		string(tokenHash),
		meta.IP,
		meta.UserAgent,
		meta.OS,
		meta.Browser,
		tokenPair.RefreshExpiresAt,
		now,
		nil,
		now,
		now,
	)

	return newSession, tokenPair, nil
}
