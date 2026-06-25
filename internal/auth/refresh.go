package auth

import (
	"bonfire-api/internal/apperr"
	"bonfire-api/internal/cache"
	"bonfire-api/internal/httpio"
	"bonfire-api/internal/token"
	"context"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// --- REFRESH CONSTANTS ---

// Messages
const (
	msgRefreshTokenSuccess = "refresh_token_success"
)

// Errors
const (
	errMissingRefreshToken = "Missing refresh token, please log in."
	errSessionInvalid      = "Invalid or unrecognized session."
	errSessionBlocked      = "Access denied. This session has been blocked."
	errSessionExpired      = "Session expired. Please log in again."
	errSessionMalformed    = "Invalid session format."
)

// --- REFRESH DTO ---

type RefreshParams struct {
	RefreshToken string
}

type RefreshResult struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type RefreshRes struct {
	AccessToken string `json:"access_token"`
}

// --- REFRESH HANDLER ---

// Refresh
func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) error {
	// Check refresh token
	cookie, err := r.Cookie(httpio.RefreshTokenCookie)
	if err != nil {
		return apperr.New(apperr.CodeUnauthorized, errMissingRefreshToken, apperr.WithErr(err))
	}

	// Rotate access token
	tokens, err := h.service.Refresh(r.Context(), RefreshParams{
		RefreshToken: cookie.Value,
	})
	if err != nil {
		return err
	}

	// Repond with tokens
	httpio.SetRefreshTokenCookie(w, tokens.RefreshToken)
	httpio.RespondOK(w, r, RefreshRes{AccessToken: tokens.AccessToken}, msgRefreshTokenSuccess)

	return nil
}

// --- REFRESH SERVICE ---

// Refresh
func (s *Service) Refresh(ctx context.Context, r RefreshParams) (RefreshResult, error) {
	// Check old token
	claims, err := s.token.VerifyRefresh(r.RefreshToken)
	if err != nil {
		return RefreshResult{}, apperr.New(apperr.CodeUnauthorized, errSessionInvalid, apperr.WithErr(err))
	}

	// Check session
	if claims.SessionID.String() == "" {
		return RefreshResult{}, apperr.New(apperr.CodeUnauthorized, errSessionMalformed, apperr.WithErr(err))
	}

	// Parse session id
	sessionID, err := uuid.Parse(claims.SessionID.String())
	if err != nil {
		return RefreshResult{}, apperr.New(apperr.CodeUnauthorized, errSessionInvalid, apperr.WithErr(err))
	}

	// Get cache session
	sessionKey := cache.UserSessionKey(claims.SessionID.String())
	var session UserSessionView
	err = s.cache.Get(ctx, sessionKey, &session)

	// Fallback to DB if cache miss
	if err == cache.ErrCacheMiss {
		session, err = s.GetUserSessionByID(ctx, claims.SessionID)
		if err != nil {
			return RefreshResult{}, err
		}
		// Backfill cache
		_ = s.cache.Set(ctx, sessionKey, session, time.Until(session.ExpiresAt))
	} else if err != nil {
		return RefreshResult{}, err
	}

	// Check for an un-rotated token
	if session.RefreshToken != r.RefreshToken {
		if !session.IsBlocked {
			_, _ = s.MarkUserSessionBlocked(ctx, session.ID)
		}
		_ = s.cache.Delete(ctx, sessionKey)
		return RefreshResult{}, apperr.New(apperr.CodeUnauthorized, errSessionInvalid, apperr.WithErr(err))
	}

	// Check if session blocked
	if session.IsBlocked {
		return RefreshResult{}, apperr.New(apperr.CodeUnauthorized, errSessionBlocked, apperr.WithErr(err))
	}

	// Check if session expired
	if time.Now().After(session.ExpiresAt) {
		return RefreshResult{}, apperr.New(apperr.CodeUnauthorized, errSessionExpired, apperr.WithErr(err))
	}

	// Get user
	userAuth, err := s.user.GetAuthByID(ctx, session.UserID)
	if err != nil {
		return RefreshResult{}, err
	}

	// Check if user active
	if !userAuth.IsActive() {
		if !session.IsBlocked {
			_, _ = s.MarkUserSessionBlocked(ctx, session.ID)
		}
		_ = s.cache.Delete(ctx, sessionKey)
		return RefreshResult{}, apperr.New(apperr.CodeUnauthorized, errSessionBlocked)
	}

	// Generate token pair
	tokenPair, err := s.token.GenerateTokenPair(userAuth.ID, string(userAuth.Role), userAuth.VerifiedAt != nil, userAuth.SecurityVersion, sessionID)
	if err != nil {
		return RefreshResult{}, apperr.NewInternal(err)
	}

	lockCtx := context.WithoutCancel(ctx)

	// Update refresh token
	_, err = s.UpdateUserSessionRefreshToken(lockCtx, UpdateRefreshTokenParams{
		ID:           session.ID,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresAt:    time.Now().Add(token.RefreshTokenTTL),
	})
	if err != nil {
		return RefreshResult{}, err
	}

	// Update cache
	session.RefreshToken = tokenPair.RefreshToken
	session.ExpiresAt = time.Now().Add(token.RefreshTokenTTL)
	if err := s.cache.Set(lockCtx, sessionKey, session, time.Until(session.ExpiresAt)); err != nil {
		_ = s.cache.Delete(ctx, sessionKey)
	}

	// Return new tokens
	return RefreshResult{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
	}, nil
}
