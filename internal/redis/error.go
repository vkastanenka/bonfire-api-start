package redis

import (
	"context"
	"errors"
	"net"

	"bonfire-api/internal/pkg/errs"

	redisdriver "github.com/redis/go-redis/v9"
)

// Resource identifies the Redis resource associated with an operation or error.
type Resource string

const (
	ResourceAuth        Resource = "auth"
	ResourceChannel     Resource = "channel"
	ResourceGateway     Resource = "gateway"
	ResourceMember      Resource = "member"
	ResourceOutboxEvent Resource = "outbox_event"
	ResourceMessage     Resource = "message"
	ResourceReaction    Resource = "reaction"
	ResourcePresence    Resource = "presence"
	ResourceRelation    Resource = "relation"
	ResourceSession     Resource = "session"
	ResourceShield      Resource = "shield"
	ResourceStore       Resource = "store"
	ResourceTicket      Resource = "ticket"
	ResourceToken       Resource = "token"
	ResourceUser        Resource = "user"
)

func (e Resource) String() string { return string(e) }

// IsCacheMiss checks if the underlying error is a redisdriver.Nil or package sentinel ErrCacheMiss.
func IsCacheMiss(err error) bool {
	return errors.Is(err, redisdriver.Nil)
}

// NewError transforms raw Redis errors into structured domain errors with resource metadata.
func NewError(err error, resource Resource) error {
	if err == nil || IsCacheMiss(err) {
		return nil
	}

	if appErr := errs.As(err); appErr != nil {
		return err
	}

	return handleCacheError(err, resource)
}

// handleCacheError classifies general cache failures into domain errors.
func handleCacheError(err error, resource Resource) error {
	var e *errs.Error

	switch {
	case errors.Is(err, context.DeadlineExceeded):
		e = ErrCacheTimeout()

	case errors.Is(err, context.Canceled):
		e = ErrCacheCanceled()

	case isNetworkError(err):
		e = ErrCacheUnavailable()

	default:
		e = ErrCacheInternal()
	}

	return ErrWithResource(e, resource).Wrap(err)
}

func isNetworkError(err error) bool {
	var netErr net.Error
	return errors.As(err, &netErr)
}
