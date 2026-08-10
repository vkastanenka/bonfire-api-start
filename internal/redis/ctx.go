package redis

import (
	"context"

	"github.com/redis/go-redis/v9"
)

type cmdableKey struct{}

// InjectCmdable injects a redis.Cmdable (e.g., redis.Pipeliner) into context.
func InjectCmdable(ctx context.Context, cmd redis.Cmdable) context.Context {
	return context.WithValue(ctx, cmdableKey{}, cmd)
}

// ExtractCmdable returns the context-bound Cmdable or falls back to defaultClient.
func ExtractCmdable(ctx context.Context, defaultClient redis.Cmdable) redis.Cmdable {
	if cmd, ok := ctx.Value(cmdableKey{}).(redis.Cmdable); ok {
		return cmd
	}
	return defaultClient
}

// IsPipelined checks whether the current context is running within an active pipeline execution.
func IsPipelined(ctx context.Context) bool {
	val := ctx.Value(cmdableKey{})
	if val == nil {
		return false
	}

	_, isPipe := val.(redis.Pipeliner)
	if !isPipe {
		return false
	}

	switch val.(type) {
	case *redis.Client, *redis.ClusterClient:
		return false
	default:
		return true
	}
}
