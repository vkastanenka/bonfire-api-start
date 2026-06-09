package domain

import (
	"context"
)

type DBRepository interface {
	WithTx(ctx context.Context, fn func(ctx context.Context) error) error
}
