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

		txErr := s.tx.ExecTx(ctx, func(txCtx context.Context) error {
			revokedIDs, err = s.sessionRepo.RevokeAll(txCtx, sess.UserID, now)
			if err != nil {
				return err
			}

			if len(revokedIDs) > 0 {
				event, err := outbox.New(
					EventSessionRevokedAll,
					outbox.NewMetadata(appctx.GetTraceID(ctx), &sess.UserID, &sess.ID, nil),
					struct{}{},
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
			s.cacheDeleteSessionBatch(ctx, claims.UserID, revokedIDs, "refresh failure")
		}

		return RefreshResult{}, ErrTokenInvalid()
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

	cacheCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
	defer cancel()

	if err := s.sessionCache.Set(cacheCtx, sess); err != nil {
		slog.WarnContext(cacheCtx, "failed to update session cache after token refresh",
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
