package cache

import (
	"github.com/redis/go-redis/v9"
)

// Cmdable mirrors DBTX in the db package.
// It allows query operations to run against a standard *redis.Client or a redis.Pipeliner.
type Cmdable interface {
	redis.Cmdable
}

type Queries struct {
	cmd Cmdable
}

func New(cmd Cmdable) *Queries {
	return &Queries{cmd: cmd}
}

// WithPipeline derives a new Queries instance bound to a Redis Pipeline or Tx.
func (q *Queries) WithPipeline(pipe redis.Pipeliner) *Queries {
	return &Queries{
		cmd: pipe,
	}
}
