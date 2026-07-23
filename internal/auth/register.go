package auth

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"bonfire-api/internal/crypto"
	"bonfire-api/internal/errs"
	"bonfire-api/internal/httpio"
	"bonfire-api/internal/outbox"
	"bonfire-api/internal/sanitize"
	"bonfire-api/internal/session"
	"bonfire-api/internal/user"

	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"
)

type RegisterParams struct {
	Email       string
	Username    string
	DisplayName *string
	Password    string
	ClientMeta  httpio.ClientMeta
}

type RegisterResult struct {
	AccessToken           string
	RefreshToken          string
	RefreshTokenExpiresAt time.Time
}

func (s *Service) Register(ctx context.Context, p RegisterParams) (RegisterResult, error) {
	email, err := user.NewEmail(sanitize.Email(p.Email))
	if err != nil || !email.IsValid() {
		return RegisterResult{}, errs.InvalidArgument("Invalid email address.").
			FieldViolation("email", "Must be a valid email address.", "INVALID_EMAIL").
			Wrap(errors.New("invalid email address"))
	}

	username, err := user.NewUsername(sanitize.Text(p.Username))
	if err != nil || !username.IsValid() {
		return RegisterResult{}, errs.InvalidArgument("Invalid username.").
			FieldViolation("username", "Must be a valid username.", "INVALID_USERNAME").
			Wrap(errors.New("invalid username"))
	}

	rawDisplayName := p.Username
	if p.DisplayName != nil && *p.DisplayName != "" {
		rawDisplayName = *p.DisplayName
	}

	displayName, err := user.NewProfileDisplayName(rawDisplayName)
	if err != nil {
		return RegisterResult{}, errs.InvalidArgument("Invalid display name format.").
			FieldViolation("display_name", "Invalid display name format.", "INVALID_DISPLAY_NAME").
			Wrap(err)
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
			return errs.Internal("failed to hash password").Wrap(hErr)
		}
		return nil
	})

	g.Go(func() error {
		var aErr error
		emailAvailable, userAvailable, aErr = s.users.CheckAvailability(gCtx, email, username)
		return aErr
	})

	if err := g.Wait(); err != nil {
		return RegisterResult{}, err
	}

	if !emailAvailable || !userAvailable {
		return RegisterResult{}, newConflictError(emailAvailable, userAvailable)
	}

	newUser, err := user.New(email, username, passwordHash, displayName)
	if err != nil {
		return RegisterResult{}, errs.Internal("failed to instantiate user").Wrap(err)
	}

	sessionID, err := uuid.NewV7()
	if err != nil {
		return RegisterResult{}, errs.Internal("failed to generate session ID").Wrap(err)
	}

	tokenPair, err := s.tokens.GeneratePair(newUser.ID(), sessionID)
	if err != nil {
		return RegisterResult{}, errs.Internal("failed to generate token pair").Wrap(err)
	}

	evToken, _, err := s.tokens.GenerateEmailVerify(newUser.ID())
	if err != nil {
		return RegisterResult{}, errs.Internal("failed to generate email verification token").Wrap(err)
	}

	tokenHash, err := session.NewRefreshTokenHash(crypto.HashToken(tokenPair.Refresh))
	if err != nil {
		return RegisterResult{}, errs.Internal("failed to hash refresh token").Wrap(err)
	}

	newSession, err := session.New(
		sessionID,
		newUser.ID(),
		tokenHash,
		tokenPair.RefreshExpiresAt,
		p.ClientMeta.IP,
		p.ClientMeta.UserAgent,
		p.ClientMeta.OS,
		p.ClientMeta.Browser,
	)
	if err != nil {
		return RegisterResult{}, errs.Internal("failed to instantiate session").Wrap(err)
	}

	persistCtx := context.WithoutCancel(ctx)

	txErr := s.tx.ExecTx(persistCtx, func(txCtx context.Context) error {
		// Persists both User & UserProfile aggregate via repository.Create
		if err := s.users.Create(txCtx, newUser); err != nil {
			return err
		}

		// Persists Session aggregate
		if err := s.sessions.Create(txCtx, newSession); err != nil {
			return err
		}

		// Emits outbox event for verification email sending
		_, err := s.outbox.Publish(txCtx, outbox.PublishParams{
			Variant: EventRegister,
			Payload: RegisterPayload{
				Email:    newUser.Email().String(),
				Username: newUser.Username().String(),
				Token:    evToken,
			},
		})
		if err != nil {
			return err
		}

		return nil
	})

	if txErr != nil {
		return RegisterResult{}, txErr
	}

	if err := s.sessionStore.Set(persistCtx, newSession); err != nil {
		slog.WarnContext(persistCtx, "failed to set session cache during registration",
			"error", err,
			"session_id", newSession.ID(),
			"user_id", newUser.ID(),
		)
	}

	return RegisterResult{
		AccessToken:           tokenPair.Access,
		RefreshToken:          tokenPair.Refresh,
		RefreshTokenExpiresAt: tokenPair.RefreshExpiresAt,
	}, nil
}

// newConflictError formats a detailed 409 Conflict error with field violations.
func newConflictError(emailAvailable, usernameAvailable bool) error {
	e := errs.AlreadyExists("The provided email or username is already taken.").
		Wrap(errors.New("registration conflict"))

	if !emailAvailable {
		e = e.FieldViolation("email", "This email address is already registered.", "ALREADY_EXISTS")
	}
	if !usernameAvailable {
		e = e.FieldViolation("username", "This username is already taken.", "ALREADY_EXISTS")
	}

	return e
}
