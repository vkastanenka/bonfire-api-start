package auth

import (
	"context"
	"time"

	"bonfire-api/internal/crypto"
	"bonfire-api/internal/errs"
	"bonfire-api/internal/fields"
	"bonfire-api/internal/httpio"
	"bonfire-api/internal/pkg/ptr"
	"bonfire-api/internal/session"
	"bonfire-api/internal/token"
	"bonfire-api/internal/user"

	"golang.org/x/sync/errgroup"
)

type Service struct {
	userRepo      UserRepository
	sessionRepo   SessionRepository
	outboxRepo    OutboxRepository
	ticketCache   TicketCache
	tokenProvider TokenProvider
	tx            TX
}

func NewService(
	userRepo UserRepository,
	sessionRepo SessionRepository,
	outboxRepo OutboxRepository,
	ticketCache TicketCache,
	tokenProvider TokenProvider,
	tx TX,
) *Service {
	return &Service{
		userRepo:      userRepo,
		sessionRepo:   sessionRepo,
		outboxRepo:    outboxRepo,
		ticketCache:   ticketCache,
		tokenProvider: tokenProvider,
		tx:            tx,
	}
}

const (
	loginTimingWindow = 35 * time.Millisecond
)

type LoginParams struct {
	Email      string
	Password   string
	ClientMeta httpio.ClientMeta
}

type LoginResult struct {
	AccessToken           string
	RefreshToken          string
	RefreshTokenExpiresAt time.Time
}

func (s *Service) Login(ctx context.Context, p LoginParams) (LoginResult, error) {
	defer crypto.ConstantWindow(loginTimingWindow)()

	email, err := user.ParseRequiredEmail("email", p.Email)
	if err != nil {
		return LoginResult{}, err
	}

	password, err := user.ParseRequiredPassword("password", p.Password)
	if err != nil {
		return LoginResult{}, err
	}

	u, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		if errs.IsNotFound(err) {
			crypto.CompareDummyPassword(p.Password)
			return LoginResult{}, ErrCredentialsInvalid()
		}
		return LoginResult{}, err
	}

	err = crypto.ComparePassword(u.PasswordHash().String(), password.String())
	if err != nil {
		return LoginResult{}, ErrCredentialsInvalid()
	}

	now := fields.Now()

	newSession, tokenPair, err := s.generateSession(u, p.ClientMeta, now)

	_, err = s.sessionRepo.Create(ctx, newSession)
	if err != nil {
		return LoginResult{}, err
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
	ClientMeta  httpio.ClientMeta
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
	email, err := user.ParseRequiredEmail("email", p.Email)
	if err != nil {
		return RegisterResult{}, err
	}

	username, err := user.ParseRequiredUsername("username", p.Username)
	if err != nil {
		return RegisterResult{}, err
	}

	displayName, err := user.ParseDisplayName("display_name", p.ResolveDisplayName())
	if err != nil {
		return RegisterResult{}, err
	}

	password, err := user.ParseRequiredPassword("password", p.Password)
	if err != nil {
		return RegisterResult{}, err
	}

	var (
		passwordHash   user.PasswordHash
		emailAvailable bool
		userAvailable  bool
	)

	g, gCtx := errgroup.WithContext(ctx)

	g.Go(func() error {
		var hErr error
		rawPassHash, hErr := crypto.HashPassword(password.String())
		if hErr != nil {
			return err
		}
		passwordHash = user.NewPasswordHash(rawPassHash)
		return nil
	})

	g.Go(func() error {
		var aErr error
		emailAvailable, userAvailable, aErr = s.userRepo.Availability(gCtx, ptr.To(email), ptr.To(username))
		return aErr
	})

	if err := g.Wait(); err != nil {
		return RegisterResult{}, err
	}

	if !emailAvailable || !userAvailable {
		return RegisterResult{}, ErrConflict(emailAvailable, userAvailable)
	}

	userID, err := fields.NewID()
	if err != nil {
		return RegisterResult{}, err
	}

	now := fields.Now()
	newUser := user.New(userID, email, username, displayName, passwordHash, now)
	newSession, tokenPair, err := s.generateSession(newUser, p.ClientMeta, now)

	// evToken, _, err := s.tokenProvider.GenerateEmailVerify(newUser.ID())
	// if err != nil {
	// 	return RegisterResult{}, errs.Internal("failed to generate email verification token").Wrap(err)
	// }

	txErr := s.tx.ExecTx(ctx, func(txCtx context.Context) error {
		if _, err := s.userRepo.Create(txCtx, newUser); err != nil {
			return err
		}

		if _, err := s.sessionRepo.Create(txCtx, newSession); err != nil {
			return err
		}

		// return s.outboxRepo.Publish(txCtx, EventRegister, RegisterPayload{
		// 	Email:    newUser.Email().String(),
		// 	Username: newUser.Username().String(),
		// 	Token:    evToken,
		// })
		return nil
	})

	if txErr != nil {
		return RegisterResult{}, txErr
	}

	return RegisterResult{
		AccessToken:           tokenPair.Access,
		RefreshToken:          tokenPair.Refresh,
		RefreshTokenExpiresAt: tokenPair.RefreshExpiresAt,
	}, nil
}

func (s *Service) generateSession(u *user.User, clientMeta httpio.ClientMeta, now fields.Timestamp) (*session.Session, token.Pair, error) {
	sessionID, err := fields.NewID()
	if err != nil {
		return nil, token.Pair{}, err
	}

	tokenPair, err := s.tokenProvider.GeneratePair(u.ID(), sessionID)
	if err != nil {
		return nil, token.Pair{}, errs.Internal("failed to generate token pair").Wrap(err)
	}

	tokenHash, err := fields.NewTokenHash(crypto.HashToken(tokenPair.Refresh))
	if err != nil {
		return nil, token.Pair{}, errs.Internal("failed to hash refresh token").Wrap(err)
	}

	newSession := session.Reconstitute(
		sessionID,
		u.ID(),
		tokenHash,
		clientMeta.IP,
		clientMeta.UserAgent,
		clientMeta.OS,
		clientMeta.Browser,
		fields.NewTimestamp(tokenPair.RefreshExpiresAt),
		now,
		fields.Timestamp{},
		now,
		now,
	)

	return newSession, tokenPair, nil
}
