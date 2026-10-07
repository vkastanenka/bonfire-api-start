package crypto

import (
	"bonfire-api/internal/pkg/errs"
	"crypto/sha256"

	"golang.org/x/crypto/bcrypt"
)

// Valid 60 character password hash.
const dummyHash = "$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy"

// HashPassword hashes a raw password using bcrypt with the default cost factor (bcrypt.DefaultCost).
// Passwords longer than 72 bytes are truncated by the hashing algorithm.
// Returns ErrHashPasswordFailed().Wrap(err) on failure.
func HashPassword(password string) (string, *errs.Error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", ErrHashPasswordFailed().Wrap(err)
	}
	return string(hash), nil
}

// CompareHashAndPassword verifies whether a raw password matches a hashed password.
// Returns ErrInvalidCredentials().Wrap(err) on failure.
func CompareHashAndPassword(hashedPassword, password string) *errs.Error {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	if err != nil {
		return ErrInvalidCredentials().Wrap(err)
	}
	return nil
}

// CompareDummyHashAndPassword verifies whether a raw password matches a dummy hashed password.
func CompareDummyHashAndPassword(password string) {
	_ = bcrypt.CompareHashAndPassword([]byte(dummyHash), []byte(password))
}

// HashToken generates a SHA-256 hash of a token or string.
func HashToken(token string) [32]byte {
	return sha256.Sum256([]byte(token))
}
