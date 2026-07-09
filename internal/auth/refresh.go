package auth

import (
	"bonfire-api/internal/apperr"
	"bonfire-api/internal/cache"
	"bonfire-api/internal/crypto"
	"bonfire-api/internal/httpio"
	"bonfire-api/internal/session"
	"bonfire-api/internal/token"
	"context"
	"crypto/subtle"
	"log/slog"
	"net/http"
	"time"
)

const (
	errMissingToken     = "Missing refresh token, please log in."
	errSessionInvalid   = "Invalid or unrecognized session."
	errSessionRevoked   = "Access denied. This session has been revoked."
	errSessionExpired   = "Session expired. Please log in again."
	errSessionMalformed = "Invalid session format."
)

type RefreshResponse struct {
	AccessToken string `json:"access_token"`
}

func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) error {
	refreshToken, err := httpio.GetCookieRefreshToken(r)
	if err != nil {
		return apperr.NewUnauthorized(err, "Missing refresh token, please log in.")
	}

	data, err := h.service.Refresh(r.Context(), RefreshParams{
		RefreshToken: refreshToken,
	})
	if err != nil {
		return err
	}

	httpio.SetCookieRefreshToken(w, httpio.SetCookieRefreshTokenParams{
		Token:   data.RefreshToken,
		Expires: data.RefreshTokenExpiresAt,
	})
	httpio.RespondOK(w, r, RefreshResponse{AccessToken: data.AccessToken})

	return nil
}

type RefreshParams struct {
	RefreshToken string
}

type RefreshResult struct {
	AccessToken           string
	RefreshToken          string
	RefreshTokenExpiresAt time.Time
}

func (s *Service) Refresh(ctx context.Context, r RefreshParams) (RefreshResult, error) {
	claims, err := s.token.VerifyRefresh(r.RefreshToken)
	if err != nil {
		return RefreshResult{}, apperr.NewTokenExpired(err, errSessionInvalid)
	}

	if claims.SessionID.String() == "" {
		return RefreshResult{}, apperr.NewUnauthorized(nil, errSessionMalformed)
	}

	sessionKey := cache.SessionKey(claims.SessionID)
	var sessionAuth session.AuthView

	err = s.cache.Get(ctx, sessionKey, &sessionAuth)

	if cache.IsNotFoundError(err) {
		val, err, _ := s.flightGroup.Do(claims.SessionID.String(), func() (interface{}, error) {
			sessionRow, dbErr := s.session.GetByID(ctx, claims.SessionID)
			if dbErr != nil {
				return nil, dbErr
			}

			auth := sessionRow.ToAuthView()

			_ = s.cache.Set(context.WithoutCancel(ctx), sessionKey, auth, time.Until(auth.ExpiresAt))

			return auth, nil
		})
		if err != nil {
			return RefreshResult{}, err
		}

		sessionAuth = val.(session.AuthView)
	} else if err != nil {
		return RefreshResult{}, err
	}

	if subtle.ConstantTimeCompare(sessionAuth.RefreshTokenHash, crypto.HashToken(r.RefreshToken)) != 1 {
		persistCtx := context.WithoutCancel(ctx)
		if !sessionAuth.IsExpired() {
			_, _ = s.session.Revoke(persistCtx, sessionAuth.ID)
		}
		_ = s.cache.Delete(persistCtx, sessionKey)
		return RefreshResult{}, apperr.NewUnauthorized(err, errSessionInvalid)
	}

	if sessionAuth.IsRevoked() {
		return RefreshResult{}, apperr.NewUnauthorized(err, errSessionRevoked)
	}

	if sessionAuth.IsExpired() {
		return RefreshResult{}, apperr.NewUnauthorized(err, errSessionExpired)
	}

	tokenPair, err := s.token.GeneratePair(token.PairParams{
		UserID:    sessionAuth.UserID,
		SessionID: sessionAuth.ID,
	})
	if err != nil {
		return RefreshResult{}, apperr.NewInternal(err, "")
	}

	hashedRefreshToken := crypto.HashToken(tokenPair.Refresh)

	persistCtx := context.WithoutCancel(ctx)

	_, err = s.session.UpdateRefreshToken(persistCtx, session.UpdateRefreshTokenParams{
		ID:               sessionAuth.ID,
		RefreshTokenHash: hashedRefreshToken,
		ExpiresAt:        tokenPair.RefreshExpiresAt,
	})
	if err != nil {
		return RefreshResult{}, err
	}

	sessionAuth.RefreshTokenHash = hashedRefreshToken
	sessionAuth.ExpiresAt = tokenPair.RefreshExpiresAt
	if err := s.cache.Set(
		persistCtx,
		sessionKey,
		sessionAuth,
		time.Until(sessionAuth.ExpiresAt),
	); err != nil {
		slog.ErrorContext(ctx,
			"failed to set session",
			"error", err,
			"sessionID", sessionAuth.ID,
			"userID", sessionAuth.UserID,
		)
		_ = s.cache.Delete(persistCtx, sessionKey)
	}

	return RefreshResult{
		AccessToken:           tokenPair.Access,
		RefreshToken:          tokenPair.Refresh,
		RefreshTokenExpiresAt: tokenPair.RefreshExpiresAt,
	}, nil
}
