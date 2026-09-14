package auth

import (
	"bonfire-api/internal/crypto"
	"bonfire-api/internal/session"
	"context"
	"crypto/subtle"
	"log/slog"
	"net/netip"
	"time"

	"github.com/google/uuid"
)

type RefreshParams struct {
	RefreshToken string
	ClientIP     netip.Addr
	UserAgent    string
}

type RefreshResult struct {
	AccessToken           string
	RefreshToken          string
	RefreshTokenExpiresAt time.Time
}

func (s *Service) Refresh(ctx context.Context, p RefreshParams) (RefreshResult, error) {
	claims, err := s.tokenProvider.VerifyRefresh(p.RefreshToken)
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

	now := time.Now()

	if sess.IsExpired(now) {
		return RefreshResult{}, ErrSessionExpired()
	}

	presentedBytes := crypto.HashToken(p.RefreshToken)
	currentBytes := []byte(sess.RefreshTokenHash)

	if subtle.ConstantTimeCompare(presentedBytes, currentBytes) != 1 {
		slog.WarnContext(ctx, "refresh token reuse detected: token hash mismatch",
			"session_id", sess.ID,
			"user_id", sess.UserID,
		)

		var revokedIDs []uuid.UUID

		txErr := s.tx.ExecTx(ctx, func(txCtx context.Context) error {
			var err error
			revokedIDs, err = s.sessionRepo.RevokeAll(txCtx, sess.UserID, now)
			if err != nil {
				return err
			}

			if len(revokedIDs) > 0 {
				revokePayload := session.EventRevokeAllPayload{
					UserID:     sess.UserID,
					SessionIDs: revokedIDs,
					RevokedAt:  now,
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
		p.ClientIP,
		p.UserAgent,
		tokenPair.RefreshExpiresAt,
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
