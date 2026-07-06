package crypto

import (
	"crypto/sha256"
	"errors"

	"golang.org/x/crypto/bcrypt"
)

var dummyHash = []byte("$2a$10$784.8J6lZ.tYQvH4y.44Z.L33Wby0b9lD8nE1m5f6X2xWby0b9")
var dummyPass = []byte("dummy_password")

func HashPassword(password string) (string, error) {
	if len(password) == 0 {
		return "", errors.New("password cannot be empty")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(hash), err
}

func ComparePassword(hashedPassword string, password string) error {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	if err != nil && errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
		_ = bcrypt.CompareHashAndPassword(dummyHash, dummyPass)
	}
	return err
}

func HashToken(tokenStr string) []byte {
	hash := sha256.Sum256([]byte(tokenStr))
	return hash[:]
}
