package redis

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

type pipeKey struct{}

// InjectPipeline embeds a Redis Pipeliner into the context.
func InjectPipeline(ctx context.Context, pipe redis.Pipeliner) context.Context {
	return context.WithValue(ctx, pipeKey{}, pipe)
}

// ExtractPipeline attempts to retrieve a Redis Pipeliner from the context.
func ExtractPipeline(ctx context.Context) (redis.Pipeliner, bool) {
	pipe, ok := ctx.Value(pipeKey{}).(redis.Pipeliner)
	return pipe, ok
}

// runner abstracts the exact go-redis commands used by Queries.
type runner interface {
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) *redis.StatusCmd
	Get(ctx context.Context, key string) *redis.StringCmd
	MGet(ctx context.Context, keys ...string) *redis.SliceCmd
	Del(ctx context.Context, keys ...string) *redis.IntCmd
	Exists(ctx context.Context, keys ...string) *redis.IntCmd
	HSet(ctx context.Context, key string, values ...interface{}) *redis.IntCmd
	HGet(ctx context.Context, key, field string) *redis.StringCmd
	HDel(ctx context.Context, key string, fields ...string) *redis.IntCmd
	HGetAll(ctx context.Context, key string) *redis.MapStringStringCmd
	ZAdd(ctx context.Context, key string, members ...redis.Z) *redis.IntCmd
	ZRem(ctx context.Context, key string, members ...interface{}) *redis.IntCmd
	ZRangeArgsWithScores(ctx context.Context, args redis.ZRangeArgs) *redis.ZSliceCmd
	ZRemRangeByRank(ctx context.Context, key string, start, stop int64) *redis.IntCmd
	Expire(ctx context.Context, key string, ttl time.Duration) *redis.BoolCmd
	Publish(ctx context.Context, channel string, payload interface{}) *redis.IntCmd
	Eval(ctx context.Context, script string, keys []string, args ...interface{}) *redis.Cmd
	EvalSha(ctx context.Context, sha1 string, keys []string, args ...interface{}) *redis.Cmd
	ScriptLoad(ctx context.Context, script string) *redis.StringCmd
	ScriptExists(ctx context.Context, hashes ...string) *redis.BoolSliceCmd
}

type contextCmdable struct {
	client redis.Cmdable
}

func newContextCmdable(client redis.Cmdable) runner {
	return &contextCmdable{client: client}
}

func (c *contextCmdable) target(ctx context.Context) runner {
	if pipe, ok := ExtractPipeline(ctx); ok {
		return pipe
	}
	return c.client
}

func (c *contextCmdable) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) *redis.StatusCmd {
	return c.target(ctx).Set(ctx, key, value, ttl)
}

func (c *contextCmdable) Get(ctx context.Context, key string) *redis.StringCmd {
	return c.target(ctx).Get(ctx, key)
}

func (c *contextCmdable) MGet(ctx context.Context, keys ...string) *redis.SliceCmd {
	return c.target(ctx).MGet(ctx, keys...)
}

func (c *contextCmdable) Del(ctx context.Context, keys ...string) *redis.IntCmd {
	return c.target(ctx).Del(ctx, keys...)
}

func (c *contextCmdable) Exists(ctx context.Context, keys ...string) *redis.IntCmd {
	return c.target(ctx).Exists(ctx, keys...)
}

func (c *contextCmdable) HSet(ctx context.Context, key string, values ...interface{}) *redis.IntCmd {
	return c.target(ctx).HSet(ctx, key, values...)
}

func (c *contextCmdable) HGet(ctx context.Context, key, field string) *redis.StringCmd {
	return c.target(ctx).HGet(ctx, key, field)
}

func (c *contextCmdable) HDel(ctx context.Context, key string, fields ...string) *redis.IntCmd {
	return c.target(ctx).HDel(ctx, key, fields...)
}

func (c *contextCmdable) HGetAll(ctx context.Context, key string) *redis.MapStringStringCmd {
	return c.target(ctx).HGetAll(ctx, key)
}

func (c *contextCmdable) ZAdd(ctx context.Context, key string, members ...redis.Z) *redis.IntCmd {
	return c.target(ctx).ZAdd(ctx, key, members...)
}

func (c *contextCmdable) ZRem(ctx context.Context, key string, members ...interface{}) *redis.IntCmd {
	return c.target(ctx).ZRem(ctx, key, members...)
}

func (c *contextCmdable) ZRangeArgsWithScores(ctx context.Context, args redis.ZRangeArgs) *redis.ZSliceCmd {
	return c.target(ctx).ZRangeArgsWithScores(ctx, args)
}

func (c *contextCmdable) ZRemRangeByRank(ctx context.Context, key string, start, stop int64) *redis.IntCmd {
	return c.target(ctx).ZRemRangeByRank(ctx, key, start, stop)
}

func (c *contextCmdable) Expire(ctx context.Context, key string, ttl time.Duration) *redis.BoolCmd {
	return c.target(ctx).Expire(ctx, key, ttl)
}

func (c *contextCmdable) Publish(ctx context.Context, channel string, payload interface{}) *redis.IntCmd {
	return c.target(ctx).Publish(ctx, channel, payload)
}

func (c *contextCmdable) Eval(ctx context.Context, script string, keys []string, args ...interface{}) *redis.Cmd {
	return c.target(ctx).Eval(ctx, script, keys, args...)
}

func (c *contextCmdable) EvalSha(ctx context.Context, sha1 string, keys []string, args ...interface{}) *redis.Cmd {
	return c.target(ctx).EvalSha(ctx, sha1, keys, args...)
}

func (c *contextCmdable) ScriptLoad(ctx context.Context, script string) *redis.StringCmd {
	return c.target(ctx).ScriptLoad(ctx, script)
}

func (c *contextCmdable) ScriptExists(ctx context.Context, hashes ...string) *redis.BoolSliceCmd {
	return c.target(ctx).ScriptExists(ctx, hashes...)
}
