package crypto

import (
	"bonfire-api/internal/errs"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// Valid 60-character bcrypt hash (cost 10) matching bcrypt.DefaultCost for timing mitigation.
const dummyHash = "$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy"

// HashPassword hashes a raw password using SHA-256 pre-hashing and bcrypt.
func HashPassword(password string) (string, error) {
	if len(password) == 0 {
		return "", errs.InvalidArgument("password cannot be empty")
	}

	sum := sha256.Sum256([]byte(password))
	preHash := hex.EncodeToString(sum[:])

	hash, err := bcrypt.GenerateFromPassword([]byte(preHash), bcrypt.DefaultCost)
	if err != nil {
		return "", errs.Internal("failed to hash password").Wrap(err)
	}

	return string(hash), nil
}

// ComparePasswords validates a plain candidate password against a stored bcrypt hash.
func ComparePasswords(hashedPassword, candidatePassword string) error {
	sum := sha256.Sum256([]byte(candidatePassword))
	preHash := hex.EncodeToString(sum[:])

	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(preHash))
	if err != nil {
		return errs.Unauthenticated("invalid credentials").Wrap(err)
	}

	return nil
}

// CompareDummyPassword runs bcrypt against a dummy hash to maintain constant CPU timing when a user is not found.
func CompareDummyPassword(password string) error {
	sum := sha256.Sum256([]byte(password))
	preHash := hex.EncodeToString(sum[:])

	_ = bcrypt.CompareHashAndPassword([]byte(dummyHash), []byte(preHash))
	return errs.Unauthenticated("invalid credentials")
}

// HashToken computes a fixed 32-byte SHA-256 hash of an API or authorization token string.
func HashToken(tokenStr string) [32]byte {
	return sha256.Sum256([]byte(tokenStr))
}

// ConstantWindow returns a deferrable function that delays execution until the target duration has elapsed.
func ConstantWindow(ctx context.Context, target time.Duration) func() {
	start := time.Now()

	return func() {
		remaining := target - time.Since(start)
		if remaining <= 0 {
			return
		}

		timer := time.NewTimer(remaining)
		defer timer.Stop()

		select {
		case <-timer.C:
		case <-ctx.Done():
		}
	}
}
