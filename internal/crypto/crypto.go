package crypto

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"

	"golang.org/x/crypto/bcrypt"
)

var (
	dummyHash = []byte("$2a$10$784.8J6lZ.tYQvH4y.44Z.L33Wby0b9lD8nE1m5f6X2xWby0b9")
	dummyPass = []byte("e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855")
)

func ComparePassword(hashedPassword string, password string) error {
	sum := sha256.Sum256([]byte(password))
	preHash := hex.EncodeToString(sum[:])

	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(preHash))

	if err != nil && errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
		_ = bcrypt.CompareHashAndPassword(dummyHash, dummyPass)
	}
	return err
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

func HashToken(tokenStr string) []byte {
	hash := sha256.Sum256([]byte(tokenStr))
	return hash[:]
}
