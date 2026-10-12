package auth

import (
	"context"
	"log/slog"
	"time"

	"bonfire-api/internal/appctx"
	"bonfire-api/internal/outbox"
	"bonfire-api/internal/pkg/crypto"
	"bonfire-api/internal/pkg/errs"
	"bonfire-api/internal/session"
	"bonfire-api/internal/token"
	"bonfire-api/internal/user"

	"github.com/google/uuid"
)

type Service struct {
	cache          Cache
	userCache      UserCache
	userRepo       UserRepository
	cachedUserRepo CachedUserRepository
	sessionCache   SessionCache
	sessionRepo    SessionRepository
	shieldCache    ShieldCache
	outboxRepo     OutboxRepository
	ticketCache    WSTicketCache
	tokenCache     TokenCache
	tokenProvider  TokenProvider
	tx             TX
}

func NewService(
	cache Cache,
	userCache UserCache,
	userRepo UserRepository,
	cachedUserRepo CachedUserRepository,
	sessionCache SessionCache,
	sessionRepo SessionRepository,
	shieldCache ShieldCache,
	outboxRepo OutboxRepository,
	ticketCache WSTicketCache,
	tokenCache TokenCache,
	tokenProvider TokenProvider,
	tx TX,
) *Service {
	return &Service{
		cache:          cache,
		userCache:      userCache,
		userRepo:       userRepo,
		cachedUserRepo: cachedUserRepo,
		sessionCache:   sessionCache,
		sessionRepo:    sessionRepo,
		shieldCache:    shieldCache,
		outboxRepo:     outboxRepo,
		ticketCache:    ticketCache,
		tokenCache:     tokenCache,
		tokenProvider:  tokenProvider,
		tx:             tx,
	}
}

type LoginParams struct {
	Email      string
	Password   string
	ClientMeta appctx.ClientMeta
}

type LoginResult struct {
	AccessToken           string
	RefreshToken          string
	RefreshTokenExpiresAt time.Time
}

func (s *Service) Login(ctx context.Context, p LoginParams) (LoginResult, error) {
	blocked, err := s.shieldCache.GetCooldown(ctx, "auth", "login", p.Email)
	if err != nil {
		slog.WarnContext(ctx, "failed to get login cooldown cache", slog.Any("error", err))
	} else if blocked {
		return LoginResult{}, ErrLoginCooldownExceeded()
	}

	u, err := s.userRepo.GetByEmail(ctx, p.Email)
	if err != nil {
		if errs.IsNotFound(err) {
			crypto.CompareDummyHashAndPassword(p.Password)
			return LoginResult{}, ErrCredentialsInvalid()
		}
		return LoginResult{}, err
	}

	err = crypto.CompareHashAndPassword(u.PasswordHash, p.Password)
	if err != nil {
		if _, shieldErr := s.shieldCache.RecordFailure(ctx, "auth", "login", p.Email); shieldErr != nil {
			slog.WarnContext(ctx, "failed to record login failure", slog.Any("error", shieldErr))
		}
		return LoginResult{}, ErrCredentialsInvalid()
	}

	now := time.Now()

	sess, tokenPair, err := s.generateSession(p.ClientMeta, u.ID, now)
	if err != nil {
		return LoginResult{}, err
	}

	if u.IsDisabled() || u.IsScheduledForDeletion() {
		txErr := s.tx.ExecTx(ctx, func(txCtx context.Context) error {
			if u.IsScheduledForDeletion() {
				u, err = s.userRepo.SetDeleteSchedule(txCtx, u.ID, nil, nil, now)
			} else {
				u, err = s.userRepo.SetDisabled(txCtx, u.ID, nil, now)
			}
			if err != nil {
				return err
			}

			sess, err = s.sessionRepo.Create(txCtx, sess)
			return err
		})
		if txErr != nil {
			return LoginResult{}, txErr
		}
	} else {
		sess, err = s.sessionRepo.Create(ctx, sess)
		if err != nil {
			return LoginResult{}, err
		}
	}

	if err := s.cache.Login(ctx, u, sess); err != nil {
		slog.WarnContext(ctx, "failed cache login after login",
			slog.String("user_id", u.ID.String()),
			slog.String("session_id", sess.ID.String()),
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
	ClientMeta  appctx.ClientMeta
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
	uID, err := uuid.NewV7()
	if err != nil {
		return RegisterResult{}, ErrIDGenerateFailed().
			Resource("domain", "user", "", "").
			Wrap(err)
	}

	pHash, err := crypto.HashPassword(p.Password)
	if err != nil {
		return RegisterResult{}, err
	}

	now := time.Now()

	u := user.New(
		uID,
		p.Email,
		p.Username,
		p.ResolveDisplayName(),
		pHash,
		now,
	)

	sess, tokenPair, err := s.generateSession(p.ClientMeta, u.ID, now)
	if err != nil {
		return RegisterResult{}, err
	}

	evToken, _, err := s.tokenProvider.GenerateEmailVerify(u.ID)
	if err != nil {
		return RegisterResult{}, err
	}

	txErr := s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		if u, err = s.userRepo.Create(txCtx, u); err != nil {
			return err
		}

		sess, err = s.sessionRepo.Create(txCtx, sess)
		if err != nil {
			return err
		}

		event, err := outbox.New(
			EventRegistered,
			outbox.NewMetadata(appctx.GetTraceID(txCtx), &u.ID, &sess.ID, nil),
			newEventRegisteredPayload(u.Email, u.Username, evToken, now),
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

	if err := s.cache.Login(ctx, u, sess); err != nil {
		slog.WarnContext(ctx, "failed cache login after registration",
			slog.String("user_id", u.ID.String()),
			slog.String("session_id", sess.ID.String()),
			slog.Any("error", err),
		)
	}

	return RegisterResult{
		AccessToken:           tokenPair.Access,
		RefreshToken:          tokenPair.Refresh,
		RefreshTokenExpiresAt: tokenPair.RefreshExpiresAt,
	}, nil
}

func (s *Service) generateSession(meta appctx.ClientMeta, userID uuid.UUID, now time.Time) (*session.Session, token.Pair, error) {
	sid, err := uuid.NewV7()
	if err != nil {
		return nil, token.Pair{}, ErrIDGenerateFailed().
			Resource("domain", "session", "", "").
			Wrap(err)
	}

	tokenPair, err := s.tokenProvider.GeneratePair(userID, sid)
	if err != nil {
		return nil, token.Pair{}, err
	}

	sess := session.New(
		sid,
		userID,
		crypto.HashToken(tokenPair.Refresh),
		meta.IP,
		meta.UserAgent,
		meta.OS,
		meta.Client,
		tokenPair.RefreshExpiresAt,
		now,
	)

	return sess, tokenPair, nil
}
