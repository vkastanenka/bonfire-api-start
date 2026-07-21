package auth

import (
	"bonfire-api/internal/httpio"
	"context"
	"net/http"
	"time"
)

const (
	errInvalidResetToken = "Invalid or expired reset token."
)

type ResetPasswordRequest struct {
	Token    string `json:"token" validate:"required"`
	Password string `json:"password" validate:"identity_password"`
}

type ResetPasswordResponse struct {
	AccessToken string `json:"access_token"`
}

func (h *Handler) ResetPassword(w http.ResponseWriter, r *http.Request) error {
	req, err := httpio.BindJSON[ResetPasswordRequest](nil, w, r)
	if err != nil {
		return err
	}

	clientMeta, err := httpio.GetClientMeta(r.Context())
	if err != nil {
		return err
	}

	data, err := h.service.ResetPassword(r.Context(), ResetPasswordParams{
		req.Token,
		req.Password,
		clientMeta,
	})
	if err != nil {
		return err
	}

	httpio.SetCookieRefreshToken(w, httpio.SetCookieRefreshTokenParams{
		Token:   data.RefreshToken,
		Expires: data.RefreshTokenExpiresAt,
	})
	httpio.RespondOK(w, r, RegisterResponse{AccessToken: data.AccessToken})
	return nil
}

type ResetPasswordParams struct {
	Token      string
	Password   string
	ClientMeta httpio.ClientMeta
}

type ResetPasswordResult struct {
	AccessToken           string
	RefreshToken          string
	RefreshTokenExpiresAt time.Time
}

func (s *Service) ResetPassword(ctx context.Context, p ResetPasswordParams) (ResetPasswordResult, error) {
	// claims, err := s.token.VerifyPasswordReset(p.Token)
	// if err != nil {
	// 	return ResetPasswordResult{}, apperr.NewTokenExpired(err, "")
	// }

	// sessionID, err := uuid.NewV7()
	// if err != nil {
	// 	return ResetPasswordResult{}, apperr.NewInternal(err, "")
	// }

	// tokenPair, err := s.token.GeneratePair(token.PairParams{
	// 	UserID:    claims.UserID,
	// 	SessionID: sessionID,
	// })
	// if err != nil {
	// 	return ResetPasswordResult{}, apperr.NewInternal(err, "")
	// }

	// hashedRefreshToken := crypto.HashToken(tokenPair.Refresh)

	// hashedPasswordBytes, err := crypto.HashPassword(p.Password)
	// if err != nil {
	// 	return ResetPasswordResult{}, apperr.NewInternal(err, "")
	// }

	// var sessionRaw repository.Session

	// persistCtx := context.WithoutCancel(ctx)

	// txErr := s.store.ExecTx(persistCtx, func(qtx *repository.Queries) error {
	// 	err = qtx.SessionDeleteByUserID(persistCtx, pgtype.UUID{Bytes: claims.UserID, Valid: true})
	// 	if err != nil {
	// 		return err
	// 	}

	// 	_, err = qtx.UserUpdatePassword(persistCtx, repository.UserUpdatePasswordParams{
	// 		ID:           pgtype.UUID{Bytes: claims.UserID, Valid: true},
	// 		PasswordHash: string(hashedPasswordBytes),
	// 	})
	// 	if err != nil {
	// 		return err
	// 	}

	// 	sessionRaw, err = qtx.SessionCreate(persistCtx, repository.SessionCreateParams{
	// 		ID:               pgtype.UUID{Bytes: sessionID, Valid: true},
	// 		UserID:           pgtype.UUID{Bytes: claims.UserID, Valid: true},
	// 		RefreshTokenHash: hashedRefreshToken,
	// 		ExpiresAt:        pgtype.Timestamptz{Time: tokenPair.RefreshExpiresAt, Valid: true},
	// 		ClientIP:         p.ClientMeta.IP,
	// 		UserAgent:        p.ClientMeta.UserAgent,
	// 		OS:               p.ClientMeta.OS,
	// 		Browser:          p.ClientMeta.Browser,
	// 	})
	// 	if err != nil {
	// 		return repository.NewError(err, repository.ScopeSession)
	// 	}

	// 	return nil
	// })

	// if txErr != nil {
	// 	return ResetPasswordResult{}, txErr
	// }

	// sessionRow := session.FromRepository(sessionRaw)
	// sessionAuth := session.ToAuthView(sessionRow)
	// sessionKey := cache.SessionKey(sessionAuth.ID)
	// s.cache.Set(ctx, sessionKey, sessionAuth, time.Until(sessionAuth.ExpiresAt))

	// return ResetPasswordResult{
	// 	AccessToken:           tokenPair.Access,
	// 	RefreshToken:          tokenPair.Refresh,
	// 	RefreshTokenExpiresAt: tokenPair.RefreshExpiresAt,
	// }, nil
	return ResetPasswordResult{}, nil
}
