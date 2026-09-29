package auth

import (
	"bonfire-api/internal/appctx"
	"bonfire-api/internal/outbox"
	"bonfire-api/internal/pkg/crypto"
	"bonfire-api/internal/session"
	"context"
	"crypto/subtle"
	"log/slog"
	"time"

	"github.com/google/uuid"
)

type RefreshParams struct {
	RefreshToken string
}

type RefreshResult struct {
	AccessToken           string
	RefreshToken          string
	RefreshTokenExpiresAt time.Time
}

func (s *Service) Refresh(ctx context.Context, refreshToken string) (RefreshResult, error) {
	ctxMeta, err := appctx.GetMeta(ctx)
	if err != nil {
		return RefreshResult{}, err
	}

	claims, err := s.tokenProvider.VerifyRefresh(refreshToken)
	if err != nil {
		return RefreshResult{}, ErrRefreshTokenInvalid()
	}

	if err := s.tokenCache.ConsumeRefresh(ctx, claims); err != nil {
		return RefreshResult{}, err
	}

	sess, err := s.sessionRepo.Get(ctx, claims.SessionID)
	if err != nil {
		return RefreshResult{}, err
	}

	if sess.IsRevoked() {
		return RefreshResult{}, ErrSessionRevoked()
	}

	now := time.Now()

	if sess.IsExpired(now) {
		return RefreshResult{}, ErrSessionExpired()
	}

	presentedBytes := crypto.HashToken(refreshToken)
	currentBytes := []byte(sess.RefreshTokenHash)

	if subtle.ConstantTimeCompare(presentedBytes, currentBytes) != 1 {
		slog.WarnContext(ctx, "refresh token reuse detected: token hash mismatch",
			slog.String("session_id", sess.ID.String()),
			slog.String("user_id", sess.UserID.String()),
		)

		var revokedIDs []uuid.UUID

		txErr := s.tx.ExecTx(ctx, func(txCtx context.Context) error {
			var err error
			revokedIDs, err = s.sessionRepo.RevokeAll(txCtx, sess.UserID, now)
			if err != nil {
				return err
			}

			if len(revokedIDs) > 0 {
				payload := session.EventRevokedAllPayload{
					UserID:     sess.UserID,
					SessionIDs: revokedIDs,
					RevokedAt:  now,
				}

				event, err := outbox.New(
					sess.UserID,
					sess.ID,
					appctx.GetTraceID(txCtx),
					nil,
					session.EventRevokedAll,
					payload,
					now,
				)
				if err != nil {
					return err
				}

				return s.outboxRepo.Create(txCtx, event)
			}

			return nil
		})

		if txErr != nil {
			return RefreshResult{}, txErr
		}

		if len(revokedIDs) > 0 {
			if err := s.sessionCache.DeleteBatch(ctx, revokedIDs); err != nil {
				slog.WarnContext(ctx, "failed to invalidate revoked sessions in cache during token reuse detection",
					slog.String("user_id", sess.UserID.String()),
					slog.Int("count", len(revokedIDs)),
					slog.Any("error", err),
				)
			}
		}

		return RefreshResult{}, ErrRefreshTokenInvalidReuse()
	}

	tokenPair, err := s.tokenProvider.GeneratePair(sess.UserID, sess.ID)
	if err != nil {
		return RefreshResult{}, err
	}

	newHash := crypto.HashToken(tokenPair.Refresh)

	newSess, err := s.sessionRepo.RotateRefreshTokenHash(
		ctx,
		sess.ID,
		sess.RefreshTokenHash,
		string(newHash),
		ctxMeta.IP,
		ctxMeta.UserAgent,
		tokenPair.RefreshExpiresAt,
		now,
	)
	if err != nil {
		return RefreshResult{}, err
	}

	if err := s.sessionCache.Set(ctx, newSess); err != nil {
		slog.WarnContext(ctx, "failed to update session cache after token refresh",
			slog.String("user_id", sess.UserID.String()),
			slog.String("session_id", sess.ID.String()),
			slog.Any("error", err),
		)
	}

	return RefreshResult{
		AccessToken:           tokenPair.Access,
		RefreshToken:          tokenPair.Refresh,
		RefreshTokenExpiresAt: tokenPair.RefreshExpiresAt,
	}, nil
}
