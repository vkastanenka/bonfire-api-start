package auth

import (
	"bonfire-api/internal/appctx"
	"bonfire-api/internal/outbox"
	"bonfire-api/internal/pkg/crypto"
	"context"
	"crypto/subtle"
	"log/slog"
	"time"

	"github.com/google/uuid"
)

type RefreshParams struct {
	RefreshToken string
	ClientMeta   appctx.ClientMeta
}

type RefreshResult struct {
	AccessToken           string
	RefreshToken          string
	RefreshTokenExpiresAt time.Time
}

func (s *Service) Refresh(ctx context.Context, p RefreshParams) (RefreshResult, error) {
	claims, err := s.tokenProvider.VerifyRefresh(p.RefreshToken)
	if err != nil {
		return RefreshResult{}, err
	}

	if err = s.tokenCache.ConsumeRefresh(ctx, claims); err != nil {
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

	candidateBytes := crypto.HashToken(p.RefreshToken)

	if subtle.ConstantTimeCompare(candidateBytes[:], sess.RefreshTokenHash[:]) != 1 {
		var revokedIDs []uuid.UUID

		err = s.tx.ExecTx(ctx, func(txCtx context.Context) error {
			revokedIDs, err = s.sessionRepo.RevokeAll(txCtx, sess.UserID, now)
			if err != nil {
				return err
			}

			if len(revokedIDs) > 0 {
				event, err := outbox.New(
					EventSessionRevokedBatch,
					outbox.NewMetadata(appctx.GetTraceID(ctx), &sess.UserID, &sess.ID, nil),
					newEventSessionRevokedBatchPayload(sess.UserID, revokedIDs, now),
					now,
				)
				if err != nil {
					return err
				}

				return s.outboxRepo.Create(txCtx, event)
			}

			return nil
		})
		if err != nil {
			return RefreshResult{}, err
		}

		if len(revokedIDs) > 0 {
			if err := s.sessionCache.DeleteBatch(ctx, revokedIDs); err != nil {
				slog.WarnContext(ctx, "failed to invalidate revoked sessions in cache during token reuse detection",
					slog.String("user_id", sess.UserID.String()),
					slog.String("session_ids", sess.ID.String()),
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

	sess, err = s.sessionRepo.RotateRefreshTokenHash(
		ctx,
		sess.ID,
		sess.RefreshTokenHash,
		crypto.HashToken(tokenPair.Refresh),
		p.ClientMeta.IP,
		p.ClientMeta.UserAgent,
		p.ClientMeta.OS,
		p.ClientMeta.Client,
		tokenPair.RefreshExpiresAt,
		now,
	)
	if err != nil {
		return RefreshResult{}, err
	}

	if err := s.sessionCache.Set(ctx, sess); err != nil {
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
