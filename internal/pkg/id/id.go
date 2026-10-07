package id

import (
	"github.com/google/uuid"
)

func NewV7() (uuid.UUID, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return uuid.Nil, ErrIDGenerateFailed().Wrap(err)
	}
	return id, nil
}
