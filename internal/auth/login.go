package auth

import (
	"bonfire-api/internal/apperr"
	"bonfire-api/internal/crypto"
	"bonfire-api/internal/httpio"
	"bonfire-api/internal/repository"
	"bonfire-api/internal/session"
	"context"
	"net/http"
	"time"

	"github.com/google/uuid"
)

func newLoginCredentialsError() error {
	const msg = "Invalid credentials."
	return apperr.NewUnauthorized(
		nil,
		msg,
		apperr.Param("email", msg),
		apperr.Param("password", msg),
	)
}

type LoginReq struct {
	Email    string `json:"email" mod:"email" validate:"identity_email"`
	Password string `json:"password" validate:"identity_password"`
}

type LoginRes struct {
	AccessToken string `json:"access_token"`
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) error {
	req, err := httpio.BindJSON[LoginReq](w, r)
	if err != nil {
		return err
	}

	clientMeta, err := httpio.GetMeta(r.Context())
	if err != nil {
		return err
	}

	data, err := h.service.Login(r.Context(), LoginParams{
		Email:    req.Email,
		Password: req.Password,
		Meta:     clientMeta,
	})
	if err != nil {
		return err
	}

	httpio.SetCookieRefreshToken(w, httpio.SetCookieRefreshTokenParams{
		Token:   data.RefreshToken,
		Expires: data.RefreshTokenExpiresAt,
	})
	httpio.RespondOK(w, r, RegisterRes{AccessToken: data.AccessToken})
	return nil
}

type LoginParams struct {
	Email    string
	Password string
	Meta     httpio.ClientMeta
}

type LoginResult struct {
	AccessToken           string
	RefreshToken          string
	RefreshTokenExpiresAt time.Time
}

func (s *Service) Login(ctx context.Context, r LoginParams) (LoginResult, error) {
	userAuth, err := s.user.GetAuthByEmail(ctx, r.Email)
	if err != nil {
		if repository.IsNotFoundError(err) {
			return LoginResult{}, newLoginCredentialsError()
		}
		return LoginResult{}, err
	}

	if err = crypto.ComparePassword(userAuth.PasswordHash, r.Password); err != nil {
		return LoginResult{}, newLoginCredentialsError()
	}

	userSessionID, err := uuid.NewV7()
	if err != nil {
		return LoginResult{}, apperr.NewInternal(err, "")
	}

	tokenPair, err := s.token.GenerateTokenPair(userAuth.ID, userSessionID)
	if err != nil {
		return LoginResult{}, apperr.NewInternal(err, "")
	}

	hashedRefreshToken := crypto.HashToken(tokenPair.RefreshToken)

	persistCtx := context.WithoutCancel(ctx)

	_, err = s.session.Create(persistCtx, session.CreateParams{
		ID:               &userSessionID,
		UserID:           userAuth.ID,
		RefreshTokenHash: hashedRefreshToken,
		ExpiresAt:        tokenPair.RefreshTokenExpiresAt,
	})
	if err != nil {
		return LoginResult{}, err
	}

	return LoginResult{
		AccessToken:           tokenPair.AccessToken,
		RefreshToken:          tokenPair.RefreshToken,
		RefreshTokenExpiresAt: tokenPair.RefreshTokenExpiresAt,
	}, nil
}
