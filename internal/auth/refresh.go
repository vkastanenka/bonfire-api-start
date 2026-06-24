package auth

import (
	"bonfire-api/internal/apperr"
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
	MsgRefreshTokenSuccess = "refresh_token_success"
)

// Errors
const (
	ErrMissingRefreshToken = "Missing refresh token, please log in."
	ErrSessionInvalid      = "Invalid or unrecognized session."
	ErrSessionBlocked      = "Access denied. This session has been blocked."
	ErrSessionExpired      = "Session expired. Please log in again."
	ErrSessionMalformed    = "Invalid session format."
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
		return apperr.New(apperr.CodeUnauthenticated, ErrMissingRefreshToken, apperr.WithErr(err))
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
	httpio.RespondOK(w, r, RefreshRes{AccessToken: tokens.AccessToken}, MsgRefreshTokenSuccess)

	return nil
}

// --- REFRESH SERVICE ---

// Refresh
func (s *Service) Refresh(ctx context.Context, r RefreshParams) (RefreshResult, error) {
	// Check old token
	claims, err := s.token.VerifyRefresh(r.RefreshToken)
	if err != nil {
		return RefreshResult{}, apperr.New(apperr.CodeUnauthenticated, ErrSessionInvalid, apperr.WithErr(err))
	}

	// Check session
	if claims.SessionID.String() == "" {
		return RefreshResult{}, apperr.New(apperr.CodeUnauthenticated, ErrSessionMalformed, apperr.WithErr(err))
	}

	// Parse session id
	sessionID, err := uuid.Parse(claims.SessionID.String())
	if err != nil {
		return RefreshResult{}, apperr.New(apperr.CodeUnauthenticated, ErrSessionInvalid, apperr.WithErr(err))
	}

	// Get user session
	session, err := s.GetUserSessionByID(ctx, sessionID)
	if err != nil {
		return RefreshResult{}, err
	}

	// Check for an un-rotated token
	if session.RefreshToken != r.RefreshToken {
		_, err = s.MarkUserSessionBlocked(ctx, session.ID)
		return RefreshResult{}, apperr.New(apperr.CodeUnauthenticated, ErrSessionInvalid, apperr.WithErr(err))
	}

	// Check if session blocked
	if session.IsBlocked {
		return RefreshResult{}, apperr.New(apperr.CodeUnauthenticated, ErrSessionBlocked, apperr.WithErr(err))
	}

	// Check if session expired
	if time.Now().After(session.ExpiresAt) {
		return RefreshResult{}, apperr.New(apperr.CodeUnauthenticated, ErrSessionExpired, apperr.WithErr(err))
	}

	// Get user
	userAuth, err := s.user.GetAuthByID(ctx, session.UserID)
	if err != nil {
		return RefreshResult{}, err
	}

	// Generate token pair
	tokenPair, err := s.token.GenerateTokenPair(userAuth.ID, string(userAuth.Role), userAuth.VerifiedAt != nil, sessionID)
	if err != nil {
		return RefreshResult{}, apperr.New(apperr.CodeInternal, apperr.CodeInternal.Title(), apperr.WithErr(err))
	}

	// Update refresh token
	_, err = s.UpdateUserSessionRefreshToken(ctx, UpdateRefreshTokenParams{
		ID:           session.ID,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresAt:    time.Now().Add(token.RefreshTokenTTL),
	})
	if err != nil {
		return RefreshResult{}, err
	}

	// Return new tokens
	return RefreshResult{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
	}, nil
}
