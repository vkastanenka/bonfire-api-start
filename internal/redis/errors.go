package redis

import "bonfire-api/internal/pkg/errs"

const (
	ErrReasonCacheCorrupted          = "CACHE_CORRUPTED"
	ErrReasonCacheTimeout            = "CACHE_TIMEOUT"
	ErrReasonCacheCanceled           = "CACHE_CANCELED"
	ErrReasonCacheUnavailable        = "CACHE_UNAVAILABLE"
	ErrReasonCacheInternal           = "CACHE_INTERNAL_ERROR"
	ErrReasonResourceUnmarshalFailed = "RESOURCE_UNMARSHAL_FAILED"
	ErrReasonResourceMarshalFailed   = "RESOURCE_MARSHAL_FAILED"
)

// ErrCacheCorrupted returns an Internal error when cached data fails deserialization or schema validation.
func ErrCacheCorrupted() *errs.Error {
	return errs.Internal("Cached data is corrupted or schema mismatched.").
		Reason(ErrReasonCacheCorrupted)
}

// ErrCacheTimeout returns a DeadlineExceeded error when a cache operation exceeds its allowed time.
func ErrCacheTimeout() *errs.Error {
	return errs.DeadlineExceeded("Cache operation timed out.").
		Reason(ErrReasonCacheTimeout)
}

// ErrCacheCanceled returns a Cancelled error when a cache operation is canceled by its context.
func ErrCacheCanceled() *errs.Error {
	return errs.Cancelled("Cache operation was canceled.").
		Reason(ErrReasonCacheCanceled)
}

// ErrCacheUnavailable returns an Unavailable error when the cache service cannot be reached.
func ErrCacheUnavailable() *errs.Error {
	return errs.Unavailable("Cache service is temporarily unavailable.").
		Reason(ErrReasonCacheUnavailable)
}

// ErrCacheInternal returns an Internal error for unexpected caching failures.
func ErrCacheInternal() *errs.Error {
	return errs.Internal("An internal caching error occurred.").
		Reason(ErrReasonCacheInternal)
}

func ErrResourceUnmarshalFailed() *errs.Error {
	return errs.Internal("Failed to unmarshal cached resource.").
		Reason(ErrReasonResourceUnmarshalFailed)
}

func ErrResourceMarshalFailed() *errs.Error {
	return errs.Internal("Failed to marshal resource for cache.").
		Reason(ErrReasonResourceMarshalFailed)
}

// ErrWithResource attaches standard resource metadata matching the DB error pattern.
func ErrWithResource(e *errs.Error, resource Resource) *errs.Error {
	return e.Resource("redis", resource.String(), "", "")
}
