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
	
	if err != nil {
		if !cache.IsNotFoundError(err) {
			slog.WarnContext(ctx, "session cache degraded; falling back to database", "error", err, "session_id", claims.SessionID)
		}

		val, err, _ := s.flightGroup.Do(claims.SessionID.String(), func() (interface{}, error) {
			dbCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
			defer cancel()

			sessionRow, dbErr := s.session.GetByID(dbCtx, claims.SessionID)
			if dbErr != nil {
				return nil, dbErr
			}

			auth := sessionRow.ToAuthView()

			var currentCache session.AuthView
			checkErr := s.cache.Get(dbCtx, sessionKey, &currentCache)

			if cache.IsNotFoundError(checkErr) || (checkErr == nil && currentCache.UpdatedAt.Before(auth.UpdatedAt)) {
				if setErr := s.cache.Set(dbCtx, sessionKey, auth, time.Until(auth.ExpiresAt)); setErr != nil {
					slog.WarnContext(dbCtx, "failed to populate session cache during singleflight fill",
						"error", setErr,
						"session_id", auth.ID,
						"user_id", auth.UserID,
					)
				}
			}

			return auth, nil
		})
		if err != nil {
			return RefreshResult{}, err
		}

		sessionAuth = val.(session.AuthView)
	}

	persistCtx := context.WithoutCancel(ctx)

	if subtle.ConstantTimeCompare(sessionAuth.RefreshTokenHash, crypto.HashToken(r.RefreshToken)) != 1 {
		slog.WarnContext(persistCtx, "refresh token reuse detected: token hash mismatch",
			"session_id", sessionAuth.ID,
			"user_id", sessionAuth.UserID,
			"expired", sessionAuth.IsExpired(),
		)

		if _, revokeErr := s.session.Revoke(persistCtx, sessionAuth.ID); revokeErr != nil {
			slog.ErrorContext(persistCtx, "failed to revoke compromised session in database",
				"error", revokeErr,
				"session_id", sessionAuth.ID,
				"user_id", sessionAuth.UserID,
			)
		}

		if delErr := s.cache.Delete(persistCtx, sessionKey); delErr != nil {
			slog.ErrorContext(persistCtx, "failed to evict compromised session from cache",
				"error", delErr,
				"session_id", sessionAuth.ID,
				"user_id", sessionAuth.UserID,
			)
		}

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

	sessionView, err := s.session.UpdateRefreshToken(persistCtx, session.UpdateRefreshTokenParams{
		ID:               sessionAuth.ID,
		RefreshTokenHash: hashedRefreshToken,
		ExpiresAt:        tokenPair.RefreshExpiresAt,
	})
	if err != nil {
		return RefreshResult{}, err
	}

	s.updateCacheSession(persistCtx, sessionView.ToAuthView())

	return RefreshResult{
		AccessToken:           tokenPair.Access,
		RefreshToken:          tokenPair.Refresh,
		RefreshTokenExpiresAt: tokenPair.RefreshExpiresAt,
	}, nil
}
