package repository

import (
	"bonfire-api/internal/db"
)

type Store interface {
	db.Querier
}
