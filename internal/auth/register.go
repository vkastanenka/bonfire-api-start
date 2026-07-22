package auth

import (
	"bonfire-api/internal/apperr"
	"bonfire-api/internal/crypto"
	"bonfire-api/internal/httpio"
	"bonfire-api/internal/outbox"
	"bonfire-api/internal/sanitize"
	"bonfire-api/internal/session"
	"bonfire-api/internal/user"
	"context"
	"errors"
	"log/slog"
	"time"

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
		return RegisterResult{}, apperr.NewInvalidArgument(
			errors.New("invalid email address"),
			apperr.WithMsg("Invalid email address"),
		)
	}

	username, err := user.NewUsername(sanitize.Text(p.Username))
	if err != nil || !username.IsValid() {
		return RegisterResult{}, apperr.NewInvalidArgument(
			errors.New("invalid username address"),
			apperr.WithMsg("Invalid username address"),
		)
	}

	rawDisplayName := p.Username
	if p.DisplayName != nil && *p.DisplayName != "" {
		rawDisplayName = *p.DisplayName
	}

	displayName, err := user.NewProfileDisplayName(rawDisplayName)
	if err != nil {
		return RegisterResult{}, apperr.NewInvalidArgument(err, apperr.WithMsg("Invalid display name format"))
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
			return apperr.NewInternal(hErr)
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
		return RegisterResult{}, apperr.NewInternal(err)
	}

	sessionID, err := uuid.NewV7()
	if err != nil {
		return RegisterResult{}, apperr.NewInternal(err)
	}

	tokenPair, err := s.tokens.GeneratePair(newUser.ID(), sessionID)
	if err != nil {
		return RegisterResult{}, apperr.NewInternal(err)
	}

	evToken, _, err := s.tokens.GenerateEmailVerify(newUser.ID())
	if err != nil {
		return RegisterResult{}, apperr.NewInternal(err)
	}

	tokenHash, err := session.NewRefreshTokenHash(crypto.HashToken(tokenPair.Refresh))
	if err != nil {
		return RegisterResult{}, apperr.NewInternal(err)
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
		return RegisterResult{}, apperr.NewInternal(err)
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

	if err := s.sessionCache.Set(persistCtx, newSession); err != nil {
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

// newConflictError formats a detailed 409 Conflict error with field parameters.
func newConflictError(emailAvailable, usernameAvailable bool) error {
	// var params []apperr.InvalidParam

	// if !emailAvailable {
	// 	params = append(params, apperr.InvalidParam{
	// 		Name:   "email",
	// 		Reason: "This email address is already registered.",
	// 	})
	// }
	// if !usernameAvailable {
	// 	params = append(params, apperr.InvalidParam{
	// 		Name:   "username",
	// 		Reason: "This username is already taken.",
	// 	})
	// }

	return apperr.NewAlreadyExists(
		errors.New("registration conflict"),
		apperr.WithMsg("The provided email or username is already taken."),
		// apperr.WithParams(params),
	)
}
