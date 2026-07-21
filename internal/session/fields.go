package session

import (
	"bytes"
	"errors"
)

var (
	ErrInvalidTokenHash  = errors.New("refresh token hash cannot be empty")
	ErrInvalidExpiration = errors.New("session expiration must be in the future")
	ErrInvalidClientIP   = errors.New("invalid client IP address")
)

type RefreshTokenHash struct {
	value []byte
}

func NewRefreshTokenHash(hash []byte) (RefreshTokenHash, error) {
	if len(hash) == 0 {
		return RefreshTokenHash{}, ErrInvalidTokenHash
	}
	buf := make([]byte, len(hash))
	copy(buf, hash)
	return RefreshTokenHash{value: buf}, nil
}

func (h RefreshTokenHash) Bytes() []byte {
	if len(h.value) == 0 {
		return nil
	}
	buf := make([]byte, len(h.value))
	copy(buf, h.value)
	return buf
}

func (h RefreshTokenHash) Equal(other RefreshTokenHash) bool {
	return bytes.Equal(h.value, other.value)
}

func (h RefreshTokenHash) IsValid() bool {
	return len(h.value) > 0
}
