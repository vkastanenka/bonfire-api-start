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

type RefreshRes struct {
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
	httpio.RespondOK(w, r, RefreshRes{AccessToken: data.AccessToken})
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
		sessionAuth, err = s.session.GetAuthByID(ctx, claims.SessionID)
		if err != nil {
			return RefreshResult{}, err
		}
		_ = s.cache.Set(context.WithoutCancel(ctx), sessionKey, sessionAuth, time.Until(sessionAuth.ExpiresAt))
	} else if err != nil {
		return RefreshResult{}, err
	}

	incomingHash := crypto.HashToken(r.RefreshToken)

	if subtle.ConstantTimeCompare(sessionAuth.RefreshTokenHash, incomingHash) != 1 {
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

	userAuth, err := s.user.GetAuthByID(ctx, sessionAuth.UserID)
	if err != nil {
		return RefreshResult{}, err
	}

	tokenPair, err := s.token.GeneratePair(token.PairParams{
		UserID:    userAuth.ID,
		SessionID: sessionAuth.ID,
	})
	if err != nil {
		return RefreshResult{}, apperr.NewInternal(err, "")
	}

	hashedRefreshToken := crypto.HashToken(tokenPair.RefreshToken)

	persistCtx := context.WithoutCancel(ctx)

	_, err = s.session.UpdateRefreshToken(persistCtx, session.UpdateRefreshTokenParams{
		ID:               sessionAuth.ID,
		RefreshTokenHash: hashedRefreshToken,
		ExpiresAt:        tokenPair.RefreshTokenExpiresAt,
	})
	if err != nil {
		return RefreshResult{}, err
	}

	sessionAuth.RefreshTokenHash = hashedRefreshToken
	sessionAuth.ExpiresAt = tokenPair.RefreshTokenExpiresAt
	if err := s.cache.Set(persistCtx, sessionKey, sessionAuth, time.Until(sessionAuth.ExpiresAt)); err != nil {
		_ = s.cache.Delete(persistCtx, sessionKey)
	}

	return RefreshResult{
		AccessToken:           tokenPair.AccessToken,
		RefreshToken:          tokenPair.RefreshToken,
		RefreshTokenExpiresAt: tokenPair.RefreshTokenExpiresAt,
	}, nil
}
