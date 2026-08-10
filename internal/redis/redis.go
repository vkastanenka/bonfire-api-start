package redis

import "github.com/redis/go-redis/v9"

type Queries struct {
	cmd runner
}

func New(cmd redis.Cmdable) *Queries {
	return &Queries{
		cmd: newContextCmdable(cmd),
	}
}
