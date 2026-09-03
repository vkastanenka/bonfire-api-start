package auth

import (
	"bonfire-api/internal/crypto"
	"bonfire-api/internal/fields"
	"bonfire-api/internal/session"
	"context"
	"crypto/subtle"
	"log/slog"
	"time"
)

type RefreshParams struct {
	RefreshToken string
	ClientIP     fields.IP
	UserAgent    fields.UserAgent
}

type RefreshResult struct {
	AccessToken           string
	RefreshToken          string
	RefreshTokenExpiresAt time.Time
}

func (s *Service) Refresh(ctx context.Context, p RefreshParams) (RefreshResult, error) {
	refreshToken, err := fields.ParseRequiredToken("token", p.RefreshToken)
	if err != nil {
		return RefreshResult{}, err
	}

	claims, err := s.tokenProvider.VerifyRefresh(refreshToken.String())
	if err != nil {
		return RefreshResult{}, ErrRefreshTokenInvalid()
	}

	if err := s.tokenCache.ConsumeRefreshToken(ctx, claims); err != nil {
		return RefreshResult{}, err
	}

	sess, err := s.sessionRepo.Get(ctx, claims.SessionID)
	if err != nil {
		return RefreshResult{}, err
	}

	if sess.IsRevoked() {
		return RefreshResult{}, ErrSessionRevoked()
	}

	now := fields.Now()

	if sess.IsExpired(now) {
		return RefreshResult{}, ErrSessionExpired()
	}

	presentedBytes := crypto.HashToken(refreshToken.String())
	currentBytes := sess.RefreshTokenHash().Bytes.Bytes()

	if subtle.ConstantTimeCompare(presentedBytes, currentBytes) != 1 {
		slog.WarnContext(ctx, "refresh token reuse detected: token hash mismatch",
			"session_id", sess.ID(),
			"user_id", sess.UserID(),
		)

		var revokedIDs []fields.ID

		txErr := s.tx.ExecTx(ctx, func(txCtx context.Context) error {
			var err error
			revokedIDs, err = s.sessionRepo.RevokeAll(txCtx, sess.UserID(), now)
			if err != nil {
				return err
			}

			if len(revokedIDs) > 0 {
				revokedIDStrings := make([]string, len(revokedIDs))
				for i, id := range revokedIDs {
					revokedIDStrings[i] = id.String()
				}

				revokePayload := session.EventRevokeAllPayload{
					UserID:     sess.UserID().String(),
					SessionIDs: revokedIDStrings,
					RevokedAt:  now.String(),
				}

				if err := s.outboxRepo.Publish(txCtx, session.EventRevokeAll, revokePayload, now); err != nil {
					return err
				}
			}

			return nil
		})

		if txErr != nil {
			return RefreshResult{}, txErr
		}

		if len(revokedIDs) > 0 {
			_ = s.sessionCache.DeleteBatch(ctx, revokedIDs)
		}

		return RefreshResult{}, ErrRefreshTokenInvalidReuse()
	}

	tokenPair, err := s.tokenProvider.GeneratePair(sess.UserID(), sess.ID())
	if err != nil {
		return RefreshResult{}, err
	}

	oldHash, err := fields.NewTokenHash(currentBytes)
	if err != nil {
		return RefreshResult{}, err
	}

	newHash, err := fields.NewTokenHash(crypto.HashToken(tokenPair.Refresh))
	if err != nil {
		return RefreshResult{}, err
	}

	newSess, err := s.sessionRepo.RotateRefreshTokenHash(
		ctx,
		sess.ID(),
		oldHash,
		newHash,
		p.ClientIP,
		p.UserAgent,
		fields.NewTimestamp(tokenPair.RefreshExpiresAt),
		now,
	)
	if err != nil {
		return RefreshResult{}, err
	}

	_ = s.sessionCache.Set(ctx, newSess)

	return RefreshResult{
		AccessToken:           tokenPair.Access,
		RefreshToken:          tokenPair.Refresh,
		RefreshTokenExpiresAt: tokenPair.RefreshExpiresAt,
	}, nil
}
