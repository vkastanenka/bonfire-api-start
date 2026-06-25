package crypto

import (
	"time"

	"golang.org/x/crypto/bcrypt"
)

// DummyHash is a pre-calculated valid bcrypt hash used to normalize verification timings.
// It corresponds to the text "dummy_password" generated at DefaultCost.
const DummyHash = "$2a$10$3v3vWwA1pbe6T63H/SHeS.U6zL77Wby0b9lD8nE1m5f6X2xWby0b9"

// HashPassword generates a secure bcrypt hash of a plain text string.
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

// VerifyPassword compares a bcrypt hash against its plain-text candidate.
func VerifyPassword(hash, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}

// DummyVerify consumes equivalent CPU runtime cycles to defeat timing attacks.
func DummyVerify() {
	_ = bcrypt.CompareHashAndPassword([]byte(DummyHash), []byte("dummy_password"))
}

// ConstantWindow measures the time elapsed from its invocation and, when the
// returned function is executed, delays the execution path to match the target duration.
// This is used to mitigate side-channel timing attacks on sensitive endpoints.
func ConstantWindow(target time.Duration) func() {
	start := time.Now()

	return func() {
		elapsed := time.Since(start)
		if elapsed < target {
			time.Sleep(target - elapsed)
		}
	}
}
