package auth

import (
	"bonfire-api/internal/appctx"
	"bonfire-api/internal/pkg/crypto"
	"bonfire-api/internal/token"
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
)

const (
	listValidByUserIDLimit  = 20
	deleteBatchExpiredLimit = 100
)

type Service struct {
	cache          *Cache
	sessionCache   *SessionCache
	sessionRepo    *SessionRepository
	outboxRepo     OutboxRepository
	shieldCache    ShieldCache
	tokenCache     TokenCache
	tokenProvider  TokenProvider
	userCache      UserCache
	userRepo       UserRepository
	cachedUserRepo CachedUserRepository
	tx             TX
}

func NewService(
	cache *Cache,
	sessionCache *SessionCache,
	sessionRepo *SessionRepository,
	outboxRepo OutboxRepository,
	shieldCache ShieldCache,
	tokenCache TokenCache,
	tokenProvider TokenProvider,
	userCache UserCache,
	userRepo UserRepository,
	cachedUserRepo CachedUserRepository,
	tx TX,
) *Service {
	return &Service{
		cache:          cache,
		sessionCache:   sessionCache,
		sessionRepo:    sessionRepo,
		outboxRepo:     outboxRepo,
		shieldCache:    shieldCache,
		tokenCache:     tokenCache,
		tokenProvider:  tokenProvider,
		userCache:      userCache,
		userRepo:       userRepo,
		cachedUserRepo: cachedUserRepo,
		tx:             tx,
	}
}

func (s *Service) ListValidUserSessions(ctx context.Context, uid uuid.UUID) ([]*Session, error) {
	sessions, err := s.sessionRepo.ListValidByUserID(ctx, uid, time.Now(), listValidByUserIDLimit)
	if err != nil {
		return nil, err
	}

	sortSessions(sessions)

	return sessions, nil
}

func (s *Service) generateSession(meta appctx.ClientMeta, userID uuid.UUID, now time.Time) (*Session, token.Pair, error) {
	sid, err := uuid.NewV7()
	if err != nil {
		return nil, token.Pair{}, ErrIDGenerateFailed().
			Resource("domain", "session", "", "").
			Wrap(err)
	}

	tokenPair, err := s.tokenProvider.GeneratePair(userID, sid)
	if err != nil {
		return nil, token.Pair{}, err
	}

	sess := New(
		sid,
		userID,
		crypto.HashToken(tokenPair.Refresh),
		meta.IP,
		meta.UserAgent,
		meta.OS,
		meta.Client,
		tokenPair.RefreshExpiresAt,
		now,
	)

	return sess, tokenPair, nil
}

func (s *Service) deleteSessionCache(ctx context.Context, userID, sessionID uuid.UUID, action string) {
	cacheCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
	defer cancel()

	if err := s.sessionCache.Delete(cacheCtx, sessionID); err != nil {
		slog.WarnContext(cacheCtx, "failed to invalidate session cache after "+action,
			slog.String("user_id", userID.String()),
			slog.String("session_id", sessionID.String()),
			slog.Any("error", err),
		)
	}
}

func (s *Service) deleteSessionBatchCache(ctx context.Context, userID uuid.UUID, sessionIDs []uuid.UUID, action string) {
	cacheCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
	defer cancel()

	if len(sessionIDs) > 0 {
		if err := s.sessionCache.DeleteBatch(cacheCtx, sessionIDs); err != nil {
			slog.WarnContext(cacheCtx, "failed to batch invalidate session cache after "+action,
				slog.String("user_id", userID.String()),
				slog.Int("session_count", len(sessionIDs)),
				slog.Any("error", err),
			)
		}
	}
}
