package crypto

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"math/big"
	"time"

	"golang.org/x/crypto/bcrypt"
)

const dummyHash = "$2a$10$784.8J6lZ.tYQvH4y.44Z.L33Wby0b9lD8nE1m5f6X2xWby0b9"

func ComparePassword(hashedPassword string, password string) error {
	sum := sha256.Sum256([]byte(password))
	preHash := hex.EncodeToString(sum[:])

	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(preHash))
}

func HashPassword(password string) (string, error) {
	if len(password) == 0 {
		return "", errors.New("password cannot be empty")
	}

	sum := sha256.Sum256([]byte(password))
	preHash := hex.EncodeToString(sum[:])

	hash, err := bcrypt.GenerateFromPassword([]byte(preHash), bcrypt.DefaultCost)
	return string(hash), err
}

func CompareDummyPassword(password string) {
	sum := sha256.Sum256([]byte(password))
	preHash := hex.EncodeToString(sum[:])

	_ = bcrypt.CompareHashAndPassword([]byte(dummyHash), []byte(preHash))
}

func HashToken(tokenStr string) []byte {
	hash := sha256.Sum256([]byte(tokenStr))
	return hash[:]
}

func ConstantWindow(target time.Duration) func() {
	start := time.Now()

	return func() {
		elapsed := time.Since(start)
		if elapsed < target {
			time.Sleep(target - elapsed)
		}
	}
}

// Standard alphanumeric set excluding easily confused characters (0, O, 1, I, L)
const defaultVerificationCodeCharset = "23456789ABCDEFGHJKMNPQRSTUVWXYZ"

// GenerateVerificationCode creates a cryptographically secure random alphanumeric code of a given length.
func GenerateVerificationCode(length int) (string, error) {
	bytes := make([]byte, length)
	charsetLen := big.NewInt(int64(len(defaultVerificationCodeCharset)))

	for i := 0; i < length; i++ {
		num, err := rand.Int(rand.Reader, charsetLen)
		if err != nil {
			return "", err
		}
		bytes[i] = defaultVerificationCodeCharset[num.Int64()]
	}

	return string(bytes), nil
}
