package auth

import (
	"bonfire-api/internal/crypto"
	"bonfire-api/internal/fields"
	"bonfire-api/internal/sanitize"
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
	refreshToken := sanitize.Text(p.RefreshToken)
	if refreshToken == "" {
		return RefreshResult{}, ErrRefreshTokenRequired()
	}

	claims, err := s.tokenProvider.VerifyRefresh(refreshToken)
	if err != nil {
		return RefreshResult{}, ErrRefreshTokenInvalid()
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

	presentedBytes := crypto.HashToken(refreshToken)
	currentBytes := sess.RefreshTokenHash().Bytes.Bytes()

	if subtle.ConstantTimeCompare(presentedBytes, currentBytes) != 1 {
		slog.WarnContext(ctx, "refresh token reuse detected: token hash mismatch",
			"session_id", sess.ID(),
			"user_id", sess.UserID(),
		)

		if err = s.sessionRepo.Revoke(ctx, claims.SessionID, claims.UserID, now); err != nil {
			return RefreshResult{}, err
		}

		return RefreshResult{}, ErrRefreshTokenInvalidReuse()
	}

	tokenPair, err := s.tokenProvider.GeneratePair(sess.UserID().UUID(), sess.ID().UUID())
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

	_, err = s.sessionRepo.RotateRefreshTokenHash(
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

	return RefreshResult{
		AccessToken:           tokenPair.Access,
		RefreshToken:          tokenPair.Refresh,
		RefreshTokenExpiresAt: tokenPair.RefreshExpiresAt,
	}, nil
}
