package redis

import (
	"github.com/redis/go-redis/v9"
)

type Cmdable interface {
	redis.Cmdable
}

func New(cmd Cmdable) *Queries {
	return &Queries{cmd: cmd}
}

type Queries struct {
	cmd Cmdable
}

func (q *Queries) WithPipeline(pipe redis.Pipeliner) *Queries {
	return &Queries{
		cmd: pipe,
	}
}
