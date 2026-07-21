package cache

import (
	"context"
	"errors"
	"fmt"
	"net"

	"bonfire-api/internal/apperr"

	"github.com/redis/go-redis/v9"
)

type Scope string

const (
	ScopeAuth      Scope = "auth"
	ScopeEvents    Scope = "events"
	ScopePresence  Scope = "presence"
	ScopeRateLimit Scope = "rate_limit"
	ScopeSession   Scope = "session"
	ScopeStore     Scope = "store"
)

func (s Scope) String() string {
	return string(s)
}

var ErrNotFound = errors.New("cache: key not found")

func IsNotFoundError(err error) bool {
	return errors.Is(err, redis.Nil) || errors.Is(err, ErrNotFound)
}

func NewError(err error, scope Scope) error {
	if err == nil {
		return nil
	}

	var appErr *apperr.Error
	if errors.As(err, &appErr) {
		return err
	}

	meta := apperr.WithMeta("scope", scope.String())
	resourceInfo := apperr.WithResourceInfo("cache", scope.String(), "", "")
	options := apperr.WithOptions(meta, resourceInfo)

	if IsNotFoundError(err) {
		return apperr.NewNotFound(
			err,
			apperr.WithMsg(fmt.Sprintf("The requested %s was not found in cache.", scope.String())),
			options,
		)
	}

	return handleCacheError(err, scope, options)
}

func handleCacheError(err error, scope Scope, options apperr.Option) error {
	if errors.Is(err, context.DeadlineExceeded) {
		return apperr.NewDeadlineExceeded(
			err,
			apperr.WithMsg(fmt.Sprintf("Cache operation for %s timed out.", scope.String())),
			options,
		)
	}

	if errors.Is(err, context.Canceled) {
		return apperr.NewAborted(
			err,
			apperr.WithMsg(fmt.Sprintf("Cache operation for %s was canceled by the client.", scope.String())),
			options,
		)
	}

	var netErr net.Error
	if errors.As(err, &netErr) {
		return apperr.NewUnavailable(
			err,
			apperr.WithMsg(fmt.Sprintf("Cache service for %s is temporarily unavailable.", scope.String())),
			options,
		)
	}

	return apperr.NewInternal(
		err,
		apperr.WithMsg(fmt.Sprintf("An internal caching error occurred while processing %s.", scope.String())),
		options,
	)
}
