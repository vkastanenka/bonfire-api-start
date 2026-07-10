package auth

import (
	"bonfire-api/internal/apperr"
	"bonfire-api/internal/crypto"
	"bonfire-api/internal/httpio"
	"bonfire-api/internal/repository"
	"bonfire-api/internal/session"
	"bonfire-api/internal/token"
	"bonfire-api/internal/user"
	"context"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/sync/errgroup"
)

type RegisterRequest struct {
	Email       string  `json:"email" mod:"email" validate:"identity_email"`
	Username    string  `json:"username" mod:"text" validate:"identity_username"`
	DisplayName *string `json:"display_name" mod:"text" validate:"profile_display_name"`
	Password    string  `json:"password" validate:"identity_password"`
}

type RegisterResponse struct {
	AccessToken string `json:"access_token"`
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) error {
	req, err := httpio.BindJSON[RegisterRequest](w, r)
	if err != nil {
		return err
	}

	clientMeta, err := httpio.GetClientMeta(r.Context())
	if err != nil {
		return err
	}

	data, err := h.service.Register(r.Context(), RegisterParams{
		Email:       req.Email,
		Username:    req.Username,
		DisplayName: req.DisplayName,
		Password:    req.Password,
		ClientMeta:  clientMeta,
	})
	if err != nil {
		return err
	}

	httpio.SetCookieRefreshToken(w, httpio.SetCookieRefreshTokenParams{
		Token:   data.RefreshToken,
		Expires: data.RefreshTokenExpiresAt,
	})
	httpio.RespondCreated(w, r, RegisterResponse{AccessToken: data.AccessToken})

	return nil
}

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
	var passwordHash string
	var availability user.CheckAvailabilityResult

	g, gCtx := errgroup.WithContext(ctx)

	g.Go(func() error {
		var hErr error
		passwordHash, hErr = crypto.HashPassword(p.Password)
		return hErr
	})

	g.Go(func() error {
		var aErr error
		availability, aErr = s.user.CheckAvailability(gCtx, user.CheckAvailabilityParams{
			Email:    p.Email,
			Username: p.Username,
		})
		return aErr
	})

	if err := g.Wait(); err != nil {
		return RegisterResult{}, apperr.NewInternal(err, "")
	}

	if !availability.Email || !availability.Username {
		return RegisterResult{}, newConflictError(availability)
	}

	userID, err := uuid.NewV7()
	if err != nil {
		return RegisterResult{}, apperr.NewInternal(err, "")
	}

	sessionID, err := uuid.NewV7()
	if err != nil {
		return RegisterResult{}, apperr.NewInternal(err, "")
	}

	tokenPair, err := s.token.GeneratePair(token.PairParams{
		UserID:    userID,
		SessionID: sessionID,
	})
	if err != nil {
		return RegisterResult{}, apperr.NewInternal(err, "")
	}

	hashedRefreshToken := crypto.HashToken(tokenPair.Refresh)

	displayName := p.Username
	if p.DisplayName != nil && *p.DisplayName != "" {
		displayName = *p.DisplayName
	}

	var sessionRaw repository.Session

	persistCtx := context.WithoutCancel(ctx)

	txErr := s.store.ExecTx(persistCtx, func(qtx *repository.Queries) error {

		_, err := qtx.UserCreate(persistCtx, repository.UserCreateParams{
			ID:           pgtype.UUID{Bytes: userID, Valid: true},
			Email:        p.Email,
			Username:     p.Username,
			PasswordHash: passwordHash,
		})
		if err != nil {
			return repository.NewError(err, repository.ScopeUser)
		}

		_, err = qtx.UserProfileCreate(persistCtx, repository.UserProfileCreateParams{
			UserID:      pgtype.UUID{Bytes: userID, Valid: true},
			DisplayName: displayName,
		})
		if err != nil {
			return repository.NewError(err, repository.ScopeUserProfile)
		}

		sessionRaw, err = qtx.SessionCreate(persistCtx, repository.SessionCreateParams{
			ID:               pgtype.UUID{Bytes: sessionID, Valid: true},
			UserID:           pgtype.UUID{Bytes: userID, Valid: true},
			RefreshTokenHash: hashedRefreshToken,
			ExpiresAt:        pgtype.Timestamptz{Time: tokenPair.RefreshExpiresAt, Valid: true},
			ClientIP:         p.ClientMeta.IP,
			UserAgent:        p.ClientMeta.UserAgent,
			OS:               p.ClientMeta.OS,
			Browser:          p.ClientMeta.Browser,
		})
		if err != nil {
			return repository.NewError(err, repository.ScopeSession)
		}

		// err = outbox.EmitRegister(persistCtx, qtx, outbox.RegisterPayload{
		// 	Email:    userRow.Email,
		// 	Username: userRow.Username,
		// 	Token:    verificationToken,
		// })
		// if err != nil {
		// 	return repository.NewError(err, repository.ScopeOutboxEvent)
		// }

		return nil
	})

	if txErr != nil {
		return RegisterResult{}, txErr
	}

	sessionRow := session.FromRepository(sessionRaw)

	s.createCacheSession(persistCtx, session.ToAuthView(sessionRow))

	return RegisterResult{
		AccessToken:           tokenPair.Access,
		RefreshToken:          tokenPair.Refresh,
		RefreshTokenExpiresAt: tokenPair.RefreshExpiresAt,
	}, nil
}

func newConflictError(r user.CheckAvailabilityResult) error {
	var params []apperr.InvalidParam

	if !r.Email {
		params = append(params, apperr.InvalidParam{
			Name:   "email",
			Reason: "This email is already taken.",
		})
	}
	if !r.Username {
		params = append(params, apperr.InvalidParam{
			Name:   "username",
			Reason: "This username is already taken.",
		})
	}

	return apperr.NewInvalidInput(
		nil,
		"Validation failed for the request.",
		apperr.Params(params),
	)
}
