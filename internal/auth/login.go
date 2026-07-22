package auth

import (
	"bonfire-api/internal/apperr"
	"bonfire-api/internal/httpio"
	"context"
	"time"
)

// TODO: Move to config?
const (
	loginMaxAttempts     = 5
	loginLockoutDuration = 15 * time.Minute
)

type LoginParams struct {
	Email      string
	Password   string
	ClientMeta httpio.ClientMeta
}

type LoginResult struct {
	AccessToken           string
	RefreshToken          string
	RefreshTokenExpiresAt time.Time
}

func (s *Service) Login(ctx context.Context, p LoginParams) (LoginResult, error) {
	// lockoutKey := cache.AuthLoginLockoutKey(p.Email)
	// var isLocked bool
	// err := s.cache.Get(ctx, lockoutKey, &isLocked)
	// if err == nil && isLocked {
	// 	return LoginResult{}, newLockedError()
	// } else if err != nil && !cache.IsNotFoundError(err) {
	// 	slog.ErrorContext(ctx, "login lockout cache lookup failed", "error", err, "email", p.Email)
	// }

	// userRow, err := s.user.GetByEmail(ctx, p.Email)
	// if err != nil {
	// 	if repository.IsNotFoundError(err) {
	// 		crypto.CompareDummyPassword(p.Password)
	// 		return LoginResult{}, s.handleInvalidPassword(ctx, p.Email, lockoutKey)
	// 	}
	// 	return LoginResult{}, err
	// }

	// userAuth := user.ToAuthView(userRow)

	// if err = crypto.ComparePassword(userAuth.PasswordHash, p.Password); err != nil {
	// 	return LoginResult{}, s.handleInvalidPassword(ctx, p.Email, lockoutKey)
	// }

	// sessionID, err := uuid.NewV7()
	// if err != nil {
	// 	return LoginResult{}, apperr.NewInternal(err, "")
	// }

	// tokenPair, err := s.token.GeneratePair(token.PairParams{
	// 	UserID:    userAuth.ID,
	// 	SessionID: sessionID,
	// })
	// if err != nil {
	// 	return LoginResult{}, apperr.NewInternal(err, "")
	// }

	// persistCtx := context.WithoutCancel(ctx)

	// sessionRow, err := s.session.Create(persistCtx, session.CreateParams{
	// 	ID:               &sessionID,
	// 	UserID:           userAuth.ID,
	// 	RefreshTokenHash: crypto.HashToken(tokenPair.Refresh),
	// 	ExpiresAt:        tokenPair.RefreshExpiresAt,
	// 	ClientIP:         p.ClientMeta.IP,
	// 	UserAgent:        p.ClientMeta.UserAgent,
	// 	OS:               p.ClientMeta.OS,
	// 	Browser:          p.ClientMeta.Browser,
	// })
	// if err != nil {
	// 	return LoginResult{}, err
	// }

	// sessionAuth := session.ToAuthView(sessionRow)
	// sessionKey := cache.SessionKey(sessionAuth.ID)
	// s.cache.Set(ctx, sessionKey, sessionAuth, time.Until(sessionAuth.ExpiresAt))
	// s.cache.Delete(persistCtx, cache.AuthLoginFailuresKey(p.Email))

	// return LoginResult{
	// 	AccessToken:           tokenPair.Access,
	// 	RefreshToken:          tokenPair.Refresh,
	// 	RefreshTokenExpiresAt: tokenPair.RefreshExpiresAt,
	// }, nil
	return LoginResult{}, nil
}

func (s *Service) handleInvalidPassword(ctx context.Context, email string, lockoutKey string) error {
	return apperr.NewInternal(nil)
	// persistCtx := context.WithoutCancel(ctx)
	// failureKey := cache.AuthLoginFailuresKey(email)

	// attempts, err := s.cache.Increment(persistCtx, failureKey, 1*time.Hour)
	// if err != nil {
	// 	slog.ErrorContext(persistCtx, "failed to increment login failures",
	// 		"error", err,
	// 		"email", email,
	// 	)
	// 	return newCredentialsError()
	// }

	// if attempts >= loginMaxAttempts {
	// 	if err := s.cache.Set(persistCtx, lockoutKey, true, loginLockoutDuration); err != nil {
	// 		slog.ErrorContext(persistCtx, "failed to set login lockout",
	// 			"error", err,
	// 			"email", email,
	// 		)
	// 	}
	// 	return newLockedError()
	// }

	// return newCredentialsError()
}

func newLockedError() error {
	return apperr.NewInternal(nil)
	// return apperr.New(nil, "Account locked from too many failed attempts. Please try again later.")
}

func newCredentialsError() error {
	return apperr.NewInternal(nil)
	// return apperr.NewUnauthorized(
	// 	nil,
	// 	"",
	// 	apperr.Param("email", "invalid email"),
	// 	apperr.Param("password", "invalid password"),
	// )
}
