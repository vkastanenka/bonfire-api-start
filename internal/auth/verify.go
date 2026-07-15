package auth

import (
	"bonfire-api/internal/apperr"
	"bonfire-api/internal/cache"
	"bonfire-api/internal/httpio"
	"context"
	"net/http"
	"time"
)

type VerifyEmailReq struct {
	Token string `json:"token" validate:"required"`
}

func (h *Handler) VerifyEmail(w http.ResponseWriter, r *http.Request) error {
	req, err := httpio.BindJSON[VerifyEmailReq](w, r)
	if err != nil {
		return err
	}

	if err := h.service.VerifyEmail(r.Context(), req.Token); err != nil {
		return err
	}

	httpio.RespondNoContent(w)

	return nil
}

func (s *Service) VerifyEmail(ctx context.Context, tokenStr string) error {
	claims, err := s.token.VerifyEmailVerify(tokenStr)
	if err != nil {
		return apperr.NewTokenExpired(nil, apperr.CodeTokenExpired.Detail())
	}

	blacklistKey := cache.TokenBlacklistKey(claims.ID)
	isBlacklisted, err := s.cache.Exists(ctx, blacklistKey)
	if err != nil {
		return apperr.NewInternal(err, "")
	}
	if isBlacklisted {
		return apperr.NewTokenExpired(nil, apperr.CodeTokenExpired.Detail())
	}

	persistCtx := context.WithoutCancel(ctx)

	_, err = s.user.MarkVerified(persistCtx, claims.UserID)
	if err != nil {
		return err
	}

	remainingTTL := time.Until(claims.ExpiresAt.Time)
	if remainingTTL > 0 {
		s.cache.Set(persistCtx, blacklistKey, "true", remainingTTL)
	}

	return nil
}
