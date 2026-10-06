package crypto

import (
	"bonfire-api/internal/pkg/errs"
	"crypto/sha256"

	"golang.org/x/crypto/bcrypt"
)

// Valid 60 character password hash.
const dummyHash = "$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy"

// HashPassword returns the string hash of a password at the default cost of the hashing algorithm.
func HashPassword(password string) (string, *errs.Error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", ErrHashPasswordFailed().Wrap(err)
	}
	return string(hash), nil
}

// CompareHashAndPassword compares a hashed password against a candidate password.
func CompareHashAndPassword(hashedPassword, password string) *errs.Error {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	if err != nil {
		return ErrInvalidCredentials().Wrap(err)
	}
	return nil
}

// CompareDummyHashAndPassword compares a fake password hash against a candidate password.
func CompareDummyHashAndPassword(password string) {
	_ = bcrypt.CompareHashAndPassword([]byte(dummyHash), []byte(password))
}

// HashToken returns the 32 byte hash of a token.
func HashToken(token string) [32]byte {
	return sha256.Sum256([]byte(token))
}
